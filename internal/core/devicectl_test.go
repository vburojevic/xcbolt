package core

import "testing"

func TestParseFirstInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 0},
		{"pid: 123", 123},
		{"x99y88", 99},
		{"0", 0},
	}
	for _, tc := range tests {
		if got := parseFirstInt(tc.in); got != tc.want {
			t.Fatalf("parseFirstInt(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestExtractPID(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
	}{
		{"pid float", map[string]any{"pid": float64(321)}, 321},
		{"pid string", map[string]any{"processIdentifier": "456"}, 456},
		{"nested", map[string]any{"data": map[string]any{"processId": float64(789)}}, 789},
		{"none", map[string]any{"foo": "bar"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractPID(tc.in); got != tc.want {
				t.Fatalf("extractPID = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestExtractDevicesParsesPlatformAndPairing(t *testing.T) {
	payload := map[string]any{
		"devices": []any{
			map[string]any{
				"name":                      "Apple Watch",
				"identifier":                "WATCH-12345678",
				"platform":                  "watchOS",
				"operatingSystemVersion":    "11.0",
				"modelName":                 "Watch9,4",
				"pairedDeviceIdentifier":    "PHONE-ABCDEFGH",
				"companionBundleIdentifier": "com.example.phone",
			},
			map[string]any{
				"name":       "My iPhone",
				"identifier": "PHONE-ABCDEFGH",
				"platform":   "iOS",
				"osVersion":  "18.0",
				"model":      "iPhone16,2",
			},
		},
	}

	devs := extractDevices(payload)
	if len(devs) != 2 {
		t.Fatalf("len(devs) = %d, want 2", len(devs))
	}
	var watch Device
	for _, d := range devs {
		if d.PlatformFamily == PlatformWatchOS {
			watch = d
		}
	}
	if watch.Identifier == "" {
		t.Fatalf("watch device not parsed: %#v", devs)
	}
	if watch.PairedDeviceID != "PHONE-ABCDEFGH" {
		t.Fatalf("paired device id = %q", watch.PairedDeviceID)
	}
	if watch.CompanionAppID != "com.example.phone" {
		t.Fatalf("companion app id = %q", watch.CompanionAppID)
	}
}

func TestFirstString(t *testing.T) {
	m := map[string]any{"a": "", "b": "value", "c": "ignored"}
	if got := firstString(m, []string{"a", "b", "c"}); got != "value" {
		t.Fatalf("firstString = %q", got)
	}
}
