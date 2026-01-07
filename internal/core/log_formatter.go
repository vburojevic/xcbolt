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
	cmd       string
	emit      Emitter
	formatter *logFormatter
}

func newXcodebuildLogSink(ctx context.Context, cmd string, cfg Config, emit Emitter) *logSink {
	sink := &logSink{cmd: cmd, emit: emit}
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
			emitMaybe(emit, LogPretty(cmd, line))
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
	if s.formatter == nil {
		emitMaybe(s.emit, Log(s.cmd, line))
		return
	}
	emitMaybe(s.emit, LogRaw(s.cmd, line))
	s.formatter.WriteLine(line)
}

func (s *logSink) Close() {
	if s == nil || s.formatter == nil {
		return
	}
	if err := s.formatter.Close(); err != nil && !errors.Is(err, context.Canceled) {
		emitMaybe(s.emit, Warn(s.cmd, fmt.Sprintf("log formatter error: %v", err)))
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
