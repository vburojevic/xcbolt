package core

import (
	"strings"
	"testing"
)

func TestNormalizePlatformFamily(t *testing.T) {
	tests := map[string]PlatformFamily{
		"ios":         PlatformIOS,
		"iOS":         PlatformIOS,
		"iphoneos":    PlatformIOS,
		"ipad":        PlatformIPadOS,
		"iPadOS":      PlatformIPadOS,
		"tvOS":        PlatformTvOS,
		"apple tv":    PlatformTvOS,
		"xros":        PlatformVisionOS,
		"vision-os":   PlatformVisionOS,
		"watch os":    PlatformWatchOS,
		"mac":         PlatformMacOS,
		"maccatalyst": PlatformCatalyst,
		"unknown":     PlatformUnknown,
	}
	for in, want := range tests {
		if got := NormalizePlatformFamily(in); got != want {
			t.Fatalf("NormalizePlatformFamily(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeTargetType(t *testing.T) {
	tests := map[string]TargetType{
		"":          TargetAuto,
		"auto":      TargetAuto,
		"sim":       TargetSimulator,
		"simulator": TargetSimulator,
		"dev":       TargetDevice,
		"device":    TargetDevice,
		"local":     TargetLocal,
		"host":      TargetLocal,
		"wat":       TargetAuto,
	}
	for in, want := range tests {
		if got := NormalizeTargetType(in); got != want {
			t.Fatalf("NormalizeTargetType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlatformStringForDestination(t *testing.T) {
	tests := []struct {
		family PlatformFamily
		type_  TargetType
		want   string
	}{
		{PlatformIOS, TargetSimulator, "iOS Simulator"},
		{PlatformIPadOS, TargetSimulator, "iOS Simulator"},
		{PlatformIOS, TargetDevice, "iOS"},
		{PlatformTvOS, TargetDevice, "tvOS"},
		{PlatformTvOS, TargetSimulator, "tvOS Simulator"},
		{PlatformVisionOS, TargetSimulator, "visionOS Simulator"},
		{PlatformWatchOS, TargetDevice, "watchOS"},
		{PlatformMacOS, TargetLocal, "macOS"},
		{PlatformCatalyst, TargetLocal, "macOS"},
	}
	for _, tc := range tests {
		got := PlatformStringForDestination(tc.family, tc.type_)
		if got != tc.want {
			t.Fatalf("PlatformStringForDestination(%q,%q) = %q, want %q", tc.family, tc.type_, got, tc.want)
		}
	}
}

func TestInferPlatformFamilyFromRuntime(t *testing.T) {
	tests := []struct {
		runtimeID   string
		runtimeName string
		deviceName  string
		want        PlatformFamily
	}{
		{"com.apple.CoreSimulator.SimRuntime.iOS-18-0", "iOS 18.0", "iPhone 16", PlatformIOS},
		{"com.apple.CoreSimulator.SimRuntime.iOS-18-0", "iOS 18.0", "iPad Pro", PlatformIPadOS},
		{"com.apple.CoreSimulator.SimRuntime.tvOS-18-0", "tvOS 18.0", "Apple TV", PlatformTvOS},
		{"com.apple.CoreSimulator.SimRuntime.watchOS-11-0", "watchOS 11.0", "Apple Watch", PlatformWatchOS},
		{"com.apple.CoreSimulator.SimRuntime.xrOS-2-0", "visionOS 2.0", "Apple Vision Pro", PlatformVisionOS},
	}
	for _, tc := range tests {
		got := InferPlatformFamilyFromRuntime(tc.runtimeID, tc.runtimeName, tc.deviceName)
		if got != tc.want {
			t.Fatalf("InferPlatformFamilyFromRuntime(%q, %q, %q) = %q, want %q", tc.runtimeID, tc.runtimeName, tc.deviceName, got, tc.want)
		}
	}
}

func TestInferPlatformFamilyFromDevice(t *testing.T) {
	tests := []struct {
		platform string
		model    string
		name     string
		want     PlatformFamily
	}{
		{"iOS", "iPhone15,3", "Vedran iPhone", PlatformIOS},
		{"iOS", "iPad13,5", "iPad Pro", PlatformIPadOS},
		{"tvOS", "AppleTV11,1", "Apple TV", PlatformTvOS},
		{"visionOS", "RealityDevice", "Vision Pro", PlatformVisionOS},
		{"watchOS", "Watch7,4", "Apple Watch", PlatformWatchOS},
		{"macOS", "Mac14,6", "My Mac", PlatformMacOS},
	}
	for _, tc := range tests {
		got := InferPlatformFamilyFromDevice(tc.platform, tc.model, tc.name)
		if got != tc.want {
			t.Fatalf("InferPlatformFamilyFromDevice(%q,%q,%q) = %q, want %q", tc.platform, tc.model, tc.name, got, tc.want)
		}
	}
}

func TestResolveDestinationByExplicitTarget(t *testing.T) {
	cfg := Config{Destination: Destination{PlatformFamily: PlatformTvOS, TargetType: TargetSimulator, ID: "tv-sim-1"}}
	candidates := []DestinationCandidate{
		{ID: "ios-sim-1", Name: "iPhone", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", Available: true},
		{ID: "tv-sim-1", Name: "Apple TV", PlatformFamily: PlatformTvOS, TargetType: TargetSimulator, Platform: "tvOS Simulator", Available: true},
	}

	out, err := resolveDestination(cfg, candidates)
	if err != nil {
		t.Fatalf("resolveDestination: %v", err)
	}
	if out.Destination.Kind != DestSimulator {
		t.Fatalf("kind = %q, want %q", out.Destination.Kind, DestSimulator)
	}
	if out.Destination.Platform != "tvOS Simulator" {
		t.Fatalf("platform = %q", out.Destination.Platform)
	}
	if out.Destination.ID != "tv-sim-1" {
		t.Fatalf("id = %q", out.Destination.ID)
	}
}

func TestResolveDestinationAutoPickPrefersBootedSimulator(t *testing.T) {
	cfg := Config{Destination: Destination{Kind: DestAuto, TargetType: TargetAuto}}
	candidates := []DestinationCandidate{
		{ID: "sim-shutdown", Name: "iPhone 15", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", State: "Shutdown", Available: true},
		{ID: "dev-1", Name: "iPhone", PlatformFamily: PlatformIOS, TargetType: TargetDevice, Platform: "iOS", Available: true},
		{ID: "sim-booted", Name: "iPhone 16", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", State: "Booted", Available: true},
	}

	out, err := resolveDestination(cfg, candidates)
	if err != nil {
		t.Fatalf("resolveDestination: %v", err)
	}
	if out.Destination.ID != "sim-booted" {
		t.Fatalf("picked id = %q, want sim-booted", out.Destination.ID)
	}
}

func TestResolveDestinationLocalMac(t *testing.T) {
	cfg := Config{Destination: Destination{TargetType: TargetLocal, PlatformFamily: PlatformMacOS}}
	out, err := resolveDestination(cfg, nil)
	if err != nil {
		t.Fatalf("resolveDestination: %v", err)
	}
	if out.Destination.Kind != DestMacOS {
		t.Fatalf("kind = %q, want %q", out.Destination.Kind, DestMacOS)
	}
	if out.Destination.Name != "My Mac" {
		t.Fatalf("name = %q", out.Destination.Name)
	}
}

func TestResolveDestinationLocalCatalyst(t *testing.T) {
	cfg := Config{Destination: Destination{TargetType: TargetLocal, PlatformFamily: PlatformCatalyst}}
	out, err := resolveDestination(cfg, nil)
	if err != nil {
		t.Fatalf("resolveDestination: %v", err)
	}
	if out.Destination.Kind != DestCatalyst {
		t.Fatalf("kind = %q, want %q", out.Destination.Kind, DestCatalyst)
	}
	if out.Destination.Name != "My Mac (Catalyst)" {
		t.Fatalf("name = %q", out.Destination.Name)
	}
}

func TestResolveDestinationAmbiguousName(t *testing.T) {
	cfg := Config{Destination: Destination{PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Name: "iPhone 16"}}
	candidates := []DestinationCandidate{
		{ID: "sim-1", Name: "iPhone 16", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", Available: true},
		{ID: "sim-2", Name: "iPhone 16", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", Available: true},
	}
	_, err := resolveDestination(cfg, candidates)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
}

func TestResolveDestinationTargetNotFound(t *testing.T) {
	cfg := Config{Destination: Destination{PlatformFamily: PlatformIOS, TargetType: TargetSimulator, ID: "missing-id"}}
	candidates := []DestinationCandidate{
		{ID: "sim-1", Name: "iPhone 16", PlatformFamily: PlatformIOS, TargetType: TargetSimulator, Platform: "iOS Simulator", Available: true},
	}
	_, err := resolveDestination(cfg, candidates)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestNormalizeDestinationLegacyFields(t *testing.T) {
	in := Destination{Kind: DestDevice, UDID: "device-1", Platform: "iOS"}
	out := normalizeDestination(in)
	if out.TargetType != TargetDevice {
		t.Fatalf("targetType = %q", out.TargetType)
	}
	if out.PlatformFamily != PlatformIOS {
		t.Fatalf("platformFamily = %q", out.PlatformFamily)
	}
	if out.ID != "device-1" {
		t.Fatalf("id = %q", out.ID)
	}
}

func TestBuildDestinationStringMatrix(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"ios simulator", Config{Destination: Destination{Kind: DestSimulator, TargetType: TargetSimulator, PlatformFamily: PlatformIOS, ID: "sim1"}}, "platform=iOS Simulator,id=sim1"},
		{"ipados simulator", Config{Destination: Destination{Kind: DestSimulator, TargetType: TargetSimulator, PlatformFamily: PlatformIPadOS, ID: "sim2"}}, "platform=iOS Simulator,id=sim2"},
		{"tvos device", Config{Destination: Destination{Kind: DestDevice, TargetType: TargetDevice, PlatformFamily: PlatformTvOS, ID: "dev1"}}, "platform=tvOS,id=dev1"},
		{"visionos device", Config{Destination: Destination{Kind: DestDevice, TargetType: TargetDevice, PlatformFamily: PlatformVisionOS, ID: "dev2"}}, "platform=visionOS,id=dev2"},
		{"watchos simulator", Config{Destination: Destination{Kind: DestSimulator, TargetType: TargetSimulator, PlatformFamily: PlatformWatchOS, ID: "sim3"}}, "platform=watchOS Simulator,id=sim3"},
		{"macos", Config{Destination: Destination{Kind: DestMacOS, TargetType: TargetLocal, PlatformFamily: PlatformMacOS}}, "platform=macOS"},
		{"catalyst", Config{Destination: Destination{Kind: DestCatalyst, TargetType: TargetLocal, PlatformFamily: PlatformCatalyst}}, "platform=macOS,variant=Mac Catalyst"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildDestinationString(tc.cfg)
			if got != tc.want {
				t.Fatalf("BuildDestinationString = %q, want %q", got, tc.want)
			}
		})
	}
}
