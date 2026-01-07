package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SimDevice struct {
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
	Name        string `json:"name"`
	UDID        string `json:"udid"`
	// Some versions use availability string instead.
	Availability string `json:"availability"`
}

type SimRuntime struct {
	Name        string `json:"name"`
	Identifier  string `json:"identifier"`
	Version     string `json:"version"`
	IsAvailable bool   `json:"isAvailable"`
}

type SimDeviceType struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type simctlListJSON struct {
	Devices     map[string][]SimDevice `json:"devices"`
	Runtimes    []SimRuntime           `json:"runtimes"`
	DeviceTypes []SimDeviceType        `json:"devicetypes"`
}

type Simulator struct {
	Name        string `json:"name"`
	UDID        string `json:"udid"`
	State       string `json:"state"`
	RuntimeName string `json:"runtimeName"`
	RuntimeID   string `json:"runtimeId"`
	OSVersion   string `json:"osVersion,omitempty"`
	Available   bool   `json:"available"`
}

func SimctlList(ctx context.Context, emit Emitter) (simctlListJSON, error) {
	var out strings.Builder
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "list", "--json"},
		StdoutLine: func(s string) {
			out.WriteString(s)
			out.WriteString("\n")
		},
		StderrLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("simulator", s))
			}
		},
	})
	if err != nil {
		return simctlListJSON{}, err
	}
	var parsed simctlListJSON
	if err := json.Unmarshal([]byte(out.String()), &parsed); err != nil {
		return simctlListJSON{}, err
	}
	return parsed, nil
}

func FlattenSimulators(list simctlListJSON) []Simulator {
	runtimeName := map[string]string{}
	runtimeVersion := map[string]string{}
	for _, rt := range list.Runtimes {
		runtimeName[rt.Identifier] = rt.Name
		runtimeVersion[rt.Identifier] = rt.Version
	}

	out := []Simulator{}
	for runtimeID, devs := range list.Devices {
		for _, d := range devs {
			avail := d.IsAvailable
			if !avail && strings.Contains(strings.ToLower(d.Availability), "available") {
				avail = true
			}
			out = append(out, Simulator{
				Name:        d.Name,
				UDID:        d.UDID,
				State:       d.State,
				RuntimeName: runtimeName[runtimeID],
				RuntimeID:   runtimeID,
				OSVersion:   runtimeVersion[runtimeID],
				Available:   avail,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RuntimeName == out[j].RuntimeName {
			return out[i].Name < out[j].Name
		}
		return out[i].RuntimeName < out[j].RuntimeName
	})
	return out
}

func SimctlBoot(ctx context.Context, udid string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "boot", udid},
	})
	// boot returns error if already booted; ignore common cases
	if err != nil && strings.Contains(err.Error(), "Unable") {
		return nil
	}
	return err
}

func SimctlBootStatus(ctx context.Context, udid string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "bootstatus", udid, "-b"},
	})
	return err
}

func SimctlShutdown(ctx context.Context, udid string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "shutdown", udid},
	})
	return err
}

func SimctlErase(ctx context.Context, udid string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "erase", udid},
	})
	return err
}

func SimctlOpenSimulatorApp(ctx context.Context) error {
	// Best-effort: open Simulator.app
	_, err := RunStreaming(ctx, CmdSpec{Path: "open", Args: []string{"-a", "Simulator"}})
	return err
}

func SimctlOpenURL(ctx context.Context, udid string, url string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "openurl", udid, url},
	})
	return err
}

func SimctlScreenshot(ctx context.Context, udid string, outPath string) error {
	if outPath == "" {
		return errors.New("missing output path")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "io", udid, "screenshot", outPath},
	})
	return err
}

func SimctlCreate(ctx context.Context, name, deviceTypeID, runtimeID string) (string, error) {
	if name == "" || deviceTypeID == "" || runtimeID == "" {
		return "", fmt.Errorf("name, deviceTypeId, runtimeId are required")
	}
	var out strings.Builder
	_, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       []string{"simctl", "create", name, deviceTypeID, runtimeID},
		StdoutLine: func(s string) { out.WriteString(s) },
	})
	return strings.TrimSpace(out.String()), err
}

func SimctlDelete(ctx context.Context, udid string) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "delete", udid},
	})
	return err
}

func SimctlPrune(ctx context.Context) error {
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: []string{"simctl", "delete", "unavailable"},
	})
	return err
}
