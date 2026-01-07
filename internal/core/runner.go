package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type CmdSpec struct {
	Path string
	Args []string
	Dir  string
	Env  map[string]string

	StdoutLine func(string)
	StderrLine func(string)
}

type CmdResult struct {
	ExitCode int
	PID      int
	Duration time.Duration
}

func RunStreaming(ctx context.Context, spec CmdSpec) (CmdResult, error) {
	start := time.Now()

	cmd := exec.Command(spec.Path, spec.Args...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	cmd.Env = mergeEnv(os.Environ(), spec.Env)

	// Ensure we can signal the whole process group on cancel (macOS/Linux).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CmdResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return CmdResult{}, err
	}

	if err := cmd.Start(); err != nil {
		return CmdResult{}, err
	}
	pid := cmd.Process.Pid

	// Stream output
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go streamLines(stdout, spec.StdoutLine, stdoutDone)
	go streamLines(stderr, spec.StderrLine, stderrDone)

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	// Cancel handling
	cancelled := false
	select {
	case err := <-waitDone:
		<-stdoutDone
		<-stderrDone
		return finalizeResult(err, pid, time.Since(start))
	case <-ctx.Done():
		cancelled = true
		_ = syscall.Kill(-pid, syscall.SIGINT)

		select {
		case err := <-waitDone:
			<-stdoutDone
			<-stderrDone
			// Surface cancellation as context error if possible.
			if errors.Is(ctx.Err(), context.Canceled) {
				return CmdResult{ExitCode: exitCodeFromErr(err), PID: pid, Duration: time.Since(start)}, ctx.Err()
			}
			return finalizeResult(err, pid, time.Since(start))
		case <-time.After(3 * time.Second):
			_ = syscall.Kill(-pid, syscall.SIGTERM)
		}

		select {
		case err := <-waitDone:
			<-stdoutDone
			<-stderrDone
			if errors.Is(ctx.Err(), context.Canceled) {
				return CmdResult{ExitCode: exitCodeFromErr(err), PID: pid, Duration: time.Since(start)}, ctx.Err()
			}
			return finalizeResult(err, pid, time.Since(start))
		case <-time.After(3 * time.Second):
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			err := <-waitDone
			<-stdoutDone
			<-stderrDone
			if cancelled {
				return CmdResult{ExitCode: exitCodeFromErr(err), PID: pid, Duration: time.Since(start)}, ctx.Err()
			}
			return finalizeResult(err, pid, time.Since(start))
		}
	}
}

func streamLines(r io.Reader, onLine func(string), done chan<- struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(r)
	// Increase buffer for very long lines (xcodebuild can be chatty).
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		if onLine != nil {
			onLine(scanner.Text())
		}
	}
}

func finalizeResult(waitErr error, pid int, dur time.Duration) (CmdResult, error) {
	code := exitCodeFromErr(waitErr)
	res := CmdResult{ExitCode: code, PID: pid, Duration: dur}
	if waitErr == nil {
		return res, nil
	}
	return res, waitErr
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// On Unix this is syscall.WaitStatus.
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus()
		}
	}
	return 1
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	m := map[string]string{}
	for _, kv := range base {
		// Split at first '='
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	for k, v := range extra {
		m[k] = v
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
