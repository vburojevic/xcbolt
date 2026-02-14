package cli

import (
	"testing"

	"github.com/xcbolt/xcbolt/internal/core"
)

func TestApplyOverridesUnknownPlatform(t *testing.T) {
	cfg := core.Config{}
	err := applyOverrides(&cfg, "", "", "not-a-platform", "", "", "")
	if err == nil {
		t.Fatalf("expected unknown platform error")
	}
}

func TestApplyOverridesUnknownTargetType(t *testing.T) {
	cfg := core.Config{}
	err := applyOverrides(&cfg, "", "", "", "", "bad", "")
	if err == nil {
		t.Fatalf("expected unknown target type error")
	}
}

func TestApplyOverridesLocalMac(t *testing.T) {
	cfg := core.Config{}
	err := applyOverrides(&cfg, "App", "Debug", "macos", "", "local", "")
	if err != nil {
		t.Fatalf("applyOverrides: %v", err)
	}
	if cfg.Scheme != "App" || cfg.Configuration != "Debug" {
		t.Fatalf("scheme/config not applied: %+v", cfg)
	}
	if cfg.Destination.Kind != core.DestMacOS {
		t.Fatalf("kind = %q", cfg.Destination.Kind)
	}
	if cfg.Destination.ID != "" || cfg.Destination.UDID != "" {
		t.Fatalf("expected no ID/UDID for local target: %+v", cfg.Destination)
	}
}

func TestApplyOverridesWatchCompanion(t *testing.T) {
	cfg := core.Config{}
	err := applyOverrides(&cfg, "", "", "watchos", "WATCH-UDID", "device", "PHONE-UDID")
	if err != nil {
		t.Fatalf("applyOverrides: %v", err)
	}
	if cfg.Destination.PlatformFamily != core.PlatformWatchOS {
		t.Fatalf("platform family = %q", cfg.Destination.PlatformFamily)
	}
	if cfg.Destination.TargetType != core.TargetDevice {
		t.Fatalf("target type = %q", cfg.Destination.TargetType)
	}
	if cfg.Destination.ID != "WATCH-UDID" {
		t.Fatalf("id = %q", cfg.Destination.ID)
	}
	if cfg.Destination.CompanionTargetID != "PHONE-UDID" {
		t.Fatalf("companion target = %q", cfg.Destination.CompanionTargetID)
	}
}

func TestApplyOverridesSimulatorTarget(t *testing.T) {
	cfg := core.Config{}
	err := applyOverrides(&cfg, "", "", "ios", "SIM-UDID", "simulator", "")
	if err != nil {
		t.Fatalf("applyOverrides: %v", err)
	}
	if cfg.Destination.TargetType != core.TargetSimulator {
		t.Fatalf("target type = %q", cfg.Destination.TargetType)
	}
	if cfg.Destination.Kind != core.DestSimulator {
		t.Fatalf("kind = %q", cfg.Destination.Kind)
	}
	if cfg.Destination.PlatformFamily != core.PlatformIOS {
		t.Fatalf("platform family = %q", cfg.Destination.PlatformFamily)
	}
	if cfg.Destination.ID != "SIM-UDID" || cfg.Destination.UDID != "SIM-UDID" {
		t.Fatalf("id/udid mismatch: %+v", cfg.Destination)
	}
}
