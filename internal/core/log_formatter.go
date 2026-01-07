package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type LogFormat string

const (
	LogFormatAuto       LogFormat = "auto"
	LogFormatRaw        LogFormat = "raw"
	LogFormatXcpretty   LogFormat = "xcpretty"
	LogFormatXcbeautify LogFormat = "xcbeautify"
)

type logFormatter struct {
	name     string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	outDone  chan struct{}
	errDone  chan struct{}
	mu       sync.Mutex
	closed   bool
	writeErr error
}

func startLogFormatter(ctx context.Context, name string, args []string, onLine func(string), onErr func(string)) (*logFormatter, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	outDone := make(chan struct{})
	errDone := make(chan struct{})
	go streamLines(stdout, onLine, outDone)
	go streamLines(stderr, onErr, errDone)

	return &logFormatter{
		name:    name,
		cmd:     cmd,
		stdin:   stdin,
		outDone: outDone,
		errDone: errDone,
	}, nil
}

func (f *logFormatter) WriteLine(line string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return
	}
	if _, err := io.WriteString(f.stdin, line+"\n"); err != nil && f.writeErr == nil {
		f.writeErr = err
	}
}

func (f *logFormatter) Failed() bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writeErr != nil
}

func (f *logFormatter) Close() error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return nil
	}
	f.closed = true
	f.mu.Unlock()

	_ = f.stdin.Close()
	err := f.cmd.Wait()
	<-f.outDone
	<-f.errDone
	if f.writeErr != nil {
		return f.writeErr
	}
	return err
}

type logSink struct {
	cmd           string
	emit          Emitter
	formatter     *logFormatter
	bufferSize    int
	rawBuf        []logLine
	prettyLines   int
	switchedToRaw bool
	mu            sync.Mutex
}

func newXcodebuildLogSink(ctx context.Context, cmd string, cfg Config, emit Emitter) *logSink {
	sink := &logSink{cmd: cmd, emit: emit, bufferSize: 200}
	format := normalizeLogFormat(cfg.Xcodebuild.LogFormat)
	forceRaw := isNDJSONEmitter(emit)
	if forceRaw {
		format = LogFormatRaw
	}

	args := cfg.Xcodebuild.LogFormatArgs
	start := func(name string) (*logFormatter, error) {
		if _, err := exec.LookPath(name); err != nil {
			return nil, err
		}
		return startLogFormatter(ctx, name, args, func(line string) {
			sink.handlePrettyLine(line)
		}, func(line string) {
			if strings.TrimSpace(line) == "" {
				return
			}
			emitMaybe(emit, Warn(cmd, fmt.Sprintf("%s: %s", name, line)))
		})
	}

	switch format {
	case LogFormatRaw:
		return sink
	case LogFormatAuto:
		if f, err := start(string(LogFormatXcpretty)); err == nil {
			sink.formatter = f
			return sink
		}
		if f, err := start(string(LogFormatXcbeautify)); err == nil {
			sink.formatter = f
			return sink
		}
	case LogFormatXcpretty:
		if f, err := start(string(LogFormatXcpretty)); err == nil {
			sink.formatter = f
			return sink
		} else if !forceRaw {
			emitMaybe(emit, Warn(cmd, fmt.Sprintf("xcpretty not available (%v); falling back to xcbeautify/raw", err)))
		}
		if f, err := start(string(LogFormatXcbeautify)); err == nil {
			sink.formatter = f
			return sink
		}
	case LogFormatXcbeautify:
		if f, err := start(string(LogFormatXcbeautify)); err == nil {
			sink.formatter = f
			return sink
		} else if !forceRaw {
			emitMaybe(emit, Warn(cmd, fmt.Sprintf("xcbeautify not available (%v); falling back to raw", err)))
		}
	default:
		if !forceRaw {
			emitMaybe(emit, Warn(cmd, fmt.Sprintf("unknown log format %q; falling back to raw", cfg.Xcodebuild.LogFormat)))
		}
	}

	return sink
}

func (s *logSink) HandleLine(line string) {
	if s == nil {
		return
	}
	if strings.TrimSpace(line) == "" {
		return
	}

	var (
		emitRaw      bool
		emitLogRaw   bool
		emitWarning  string
		useFormatter bool
	)

	s.mu.Lock()
	s.appendRaw(line, false)

	if s.formatter != nil && s.formatter.Failed() && !s.switchedToRaw {
		s.switchedToRaw = true
		emitWarning = "log formatter stopped; falling back to raw output"
	}

	if s.switchedToRaw || s.formatter == nil {
		emitRaw = true
		useFormatter = false
	} else {
		emitLogRaw = true
		useFormatter = true
		if isXcodebuildErrorLine(line) {
			emitRaw = true
			s.markLastEmitted()
		}
	}
	s.mu.Unlock()

	if emitWarning != "" {
		emitMaybe(s.emit, Warn(s.cmd, emitWarning))
	}
	if emitLogRaw {
		emitMaybe(s.emit, LogRaw(s.cmd, line))
	}
	if emitRaw {
		emitMaybe(s.emit, Log(s.cmd, line))
	}
	if useFormatter {
		s.formatter.WriteLine(line)
	}
}

func (s *logSink) Finalize(runErr error, exitCode int) {
	if s == nil || s.formatter == nil {
		return
	}
	closeErr := s.formatter.Close()

	s.mu.Lock()
	shouldFlush := !s.switchedToRaw && (s.prettyLines == 0 || closeErr != nil || runErr != nil || exitCode != 0)
	buffer := s.copyUnemitted()
	s.mu.Unlock()

	if closeErr != nil && !errors.Is(closeErr, context.Canceled) {
		emitMaybe(s.emit, Warn(s.cmd, fmt.Sprintf("log formatter error: %v", closeErr)))
	}
	if shouldFlush {
		reason := ""
		switch {
		case s.prettyLines == 0:
			reason = "log formatter produced no output; showing raw logs"
		case closeErr != nil:
			reason = "log formatter failed; showing raw logs"
		case runErr != nil || exitCode != 0:
			reason = "xcodebuild failed; showing raw logs"
		}
		if reason != "" {
			emitMaybe(s.emit, Warn(s.cmd, reason))
		}
		for _, line := range buffer {
			emitMaybe(s.emit, Log(s.cmd, line))
		}
	}
}

func normalizeLogFormat(v string) LogFormat {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return LogFormatAuto
	case string(LogFormatAuto):
		return LogFormatAuto
	case string(LogFormatRaw):
		return LogFormatRaw
	case string(LogFormatXcpretty):
		return LogFormatXcpretty
	case string(LogFormatXcbeautify):
		return LogFormatXcbeautify
	default:
		return LogFormat(v)
	}
}

func isNDJSONEmitter(emit Emitter) bool {
	if emit == nil {
		return false
	}
	_, ok := emit.(*NDJSONEmitter)
	return ok
}

type logLine struct {
	text    string
	emitted bool
}

func (s *logSink) appendRaw(line string, emitted bool) {
	if s.bufferSize <= 0 {
		return
	}
	s.rawBuf = append(s.rawBuf, logLine{text: line, emitted: emitted})
	if len(s.rawBuf) > s.bufferSize {
		s.rawBuf = s.rawBuf[len(s.rawBuf)-s.bufferSize:]
	}
}

func (s *logSink) markLastEmitted() {
	if len(s.rawBuf) == 0 {
		return
	}
	s.rawBuf[len(s.rawBuf)-1].emitted = true
}

func (s *logSink) copyUnemitted() []string {
	if len(s.rawBuf) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.rawBuf))
	for _, l := range s.rawBuf {
		if l.emitted {
			continue
		}
		out = append(out, l.text)
	}
	return out
}

func (s *logSink) handlePrettyLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	s.mu.Lock()
	s.prettyLines++
	s.mu.Unlock()
	emitMaybe(s.emit, LogPretty(s.cmd, line))
}

func isXcodebuildErrorLine(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "error:") {
		return true
	}
	if strings.Contains(lower, "fatal error:") {
		return true
	}
	if strings.Contains(lower, "clang: error:") {
		return true
	}
	if strings.Contains(lower, "ld: error") || strings.Contains(lower, "linker command failed") {
		return true
	}
	if strings.Contains(lower, "command swiftcompile failed") || strings.Contains(lower, "command compilec failed") {
		return true
	}
	if strings.Contains(lower, "codesign error") || strings.Contains(lower, "provisioning profile") {
		return true
	}
	if strings.Contains(lower, "no such module") {
		return true
	}
	if strings.Contains(lower, "failed with exit code") {
		return true
	}
	return false
}
