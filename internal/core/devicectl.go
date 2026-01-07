package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Platform   string `json:"platform,omitempty"`
	OSVersion  string `json:"osVersion,omitempty"`
	Model      string `json:"model,omitempty"`
}

func DevicectlList(ctx context.Context, emit Emitter) ([]Device, error) {
	tmpDir := os.TempDir()
	outPath := filepath.Join(tmpDir, fmt.Sprintf("xcbolt-devices-%d.json", os.Getpid()))

	args := []string{"devicectl", "list", "devices", "--json-output", outPath}
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: args,
		StdoutLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
		StderrLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
	})
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	_ = os.Remove(outPath)

	var anyJSON any
	if err := json.Unmarshal(b, &anyJSON); err != nil {
		return nil, err
	}
	devs := extractDevices(anyJSON)
	return devs, nil
}

func DevicectlInstallApp(ctx context.Context, deviceID string, appPath string, emit Emitter) error {
	if deviceID == "" {
		return errors.New("missing --device udid")
	}
	if appPath == "" {
		return errors.New("missing app path")
	}
	args := []string{"devicectl", "device", "install", "app", "--device", deviceID, appPath}
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: args,
		StdoutLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
		StderrLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
	})
	return err
}

type LaunchResult struct {
	PID int
}

func DevicectlLaunchApp(ctx context.Context, deviceID string, bundleID string, console bool, emit Emitter) (LaunchResult, error) {
	if deviceID == "" || bundleID == "" {
		return LaunchResult{}, fmt.Errorf("deviceID and bundleID are required")
	}

	// Try a few command shapes for forward/backward compatibility.
	candidates := [][]string{
		// devicectl device launch app --device <id> <bundle>
		{"devicectl", "device", "launch", "app", "--device", deviceID, bundleID},
		// devicectl device process launch --device <id> --bundle-id <bundle>
		{"devicectl", "device", "process", "launch", "--device", deviceID, "--bundle-id", bundleID},
	}
	if console {
		for i := range candidates {
			candidates[i] = append(candidates[i], "--console")
		}
	}

	var lastErr error
	for _, args := range candidates {
		pid, err := runDevicectlLaunchCandidate(ctx, args, emit)
		if err == nil {
			return LaunchResult{PID: pid}, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("failed to launch app via devicectl")
	}
	return LaunchResult{}, lastErr
}

func runDevicectlLaunchCandidate(ctx context.Context, args []string, emit Emitter) (int, error) {
	var out strings.Builder
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: args,
		StdoutLine: func(s string) {
			out.WriteString(s)
			out.WriteString("\n")
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
		StderrLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("device", s))
			}
		},
	})
	if err != nil {
		return 0, err
	}
	// Best-effort PID parsing (varies across versions).
	pid := parseFirstInt(out.String())
	return pid, nil
}

func parseFirstInt(s string) int {
	// Simple scan: find first run of digits.
	cur := 0
	found := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			found = true
			cur = cur*10 + int(r-'0')
		} else if found {
			break
		}
	}
	if !found {
		return 0
	}
	return cur
}

// extractDevices walks arbitrary JSON and tries to find records with {name, identifier/udid}.
func extractDevices(v any) []Device {
	devs := []Device{}
	seen := map[string]bool{}

	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			name, _ := t["name"].(string)
			id := firstString(t, []string{"identifier", "udid", "deviceIdentifier", "deviceUDID", "deviceId", "id"})
			platform := firstString(t, []string{"platform", "productType", "os", "operatingSystem"})
			osv := firstString(t, []string{"osVersion", "os_version", "operatingSystemVersion", "systemVersion"})
			model := firstString(t, []string{"model", "modelName"})

			if name != "" && id != "" && len(id) >= 8 {
				key := id + "|" + name
				if !seen[key] {
					seen[key] = true
					devs = append(devs, Device{Name: name, Identifier: id, Platform: platform, OSVersion: osv, Model: model})
				}
			}

			for _, vv := range t {
				walk(vv)
			}
		case []any:
			for _, vv := range t {
				walk(vv)
			}
		}
	}
	walk(v)
	return devs
}

func firstString(m map[string]any, keys []string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s, _ := v.(string)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func DevicectlAvailable(ctx context.Context) bool {
	_, err := RunStreaming(ctx, CmdSpec{Path: "xcrun", Args: []string{"devicectl", "help"}})
	return err == nil
}

func DevicectlStop(ctx context.Context, deviceID string, pid int, bundleID string, emit Emitter) error {
	// Best-effort: try process terminate first, then app terminate.
	candidates := [][]string{}
	if pid > 0 {
		candidates = append(candidates, []string{"devicectl", "device", "process", "terminate", "--device", deviceID, "--pid", fmt.Sprintf("%d", pid)})
	}
	if bundleID != "" {
		candidates = append(candidates, []string{"devicectl", "device", "terminate", "app", "--device", deviceID, bundleID})
	}
	if len(candidates) == 0 {
		return errors.New("need pid or bundleID to stop")
	}
	var lastErr error
	for _, args := range candidates {
		_, err := RunStreaming(ctx, CmdSpec{
			Path: "xcrun",
			Args: args,
			StdoutLine: func(s string) {
				if emit != nil {
					emit.Emit(Log("device", s))
				}
			},
			StderrLine: func(s string) {
				if emit != nil {
					emit.Emit(Log("device", s))
				}
			},
		})
		if err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("failed to stop process")
	}
	return lastErr
}

// Some xcrun outputs can include carriage returns.
func normalizeLines(s string) string { return strings.ReplaceAll(s, "\r", "") }
