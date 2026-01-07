package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

type DoctorReport struct {
	Checks []DoctorCheck `json:"checks"`
}

func Doctor(ctx context.Context, projectRoot string, emit Emitter) (DoctorReport, error) {
	rep := DoctorReport{Checks: []DoctorCheck{}}

	check := func(name string, fn func() (string, error), hint string) {
		emitMaybe(emit, Status("doctor", name, nil))
		out, err := fn()
		if err != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{Name: name, OK: false, Detail: err.Error(), Hint: hint})
			emitMaybe(emit, Warn("doctor", fmt.Sprintf("%s: %v", name, err)))
			return
		}
		rep.Checks = append(rep.Checks, DoctorCheck{Name: name, OK: true, Detail: out})
	}

	check("xcodebuild available", func() (string, error) {
		var b strings.Builder
		_, err := RunStreaming(ctx, CmdSpec{
			Path:       "xcrun",
			Args:       []string{"xcodebuild", "-version"},
			StdoutLine: func(s string) { b.WriteString(s + "\n") },
		})
		return strings.TrimSpace(b.String()), err
	}, "Install Xcode and ensure xcode-select points at it.")

	check("simctl available", func() (string, error) {
		_, err := RunStreaming(ctx, CmdSpec{Path: "xcrun", Args: []string{"simctl", "list", "--json"}})
		return "ok", err
	}, "Install Xcode and ensure simulators are available.")

	check("devicectl available", func() (string, error) {
		_, err := RunStreaming(ctx, CmdSpec{Path: "xcrun", Args: []string{"devicectl", "help"}})
		return "ok", err
	}, "Xcode 15+ is required for devicectl.")

	check("xcresulttool available", func() (string, error) {
		_, err := RunStreaming(ctx, CmdSpec{Path: "xcrun", Args: []string{"xcresulttool", "help"}})
		return "ok", err
	}, "xcresulttool is part of Xcode.")

	check("project config", func() (string, error) {
		path := filepath.Join(projectRoot, ".xcbolt", "config.json")
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		return path, nil
	}, "Run `xcbolt init` to create .xcbolt/config.json.")

	return rep, nil
}
