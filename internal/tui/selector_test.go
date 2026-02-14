package tui

import (
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	if !fuzzyMatch("iphone16pro", "ip16") {
		t.Fatalf("expected fuzzy match")
	}
	if fuzzyMatch("iphone", "xyz") {
		t.Fatalf("did not expect fuzzy match")
	}
}

func TestSelectorItemMatchScore(t *testing.T) {
	item := SelectorItem{Title: "iPhone 16 Pro", Description: "iOS 18.0"}
	if got := item.MatchScore("iPh"); got != 100 {
		t.Fatalf("prefix score = %d", got)
	}
	if got := item.MatchScore("16 Pro"); got != 80 {
		t.Fatalf("contains title score = %d", got)
	}
	if got := item.MatchScore("18.0"); got != 60 {
		t.Fatalf("description score = %d", got)
	}
	if got := item.MatchScore("iph16"); got != 40 {
		t.Fatalf("fuzzy score = %d", got)
	}
	if got := item.MatchScore("zzz"); got != 0 {
		t.Fatalf("no match score = %d", got)
	}
}

func TestNormalizeConfigurations(t *testing.T) {
	out := normalizeConfigurations([]string{"Debug", "Release", "Debug"}, "Release")
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}

	out2 := normalizeConfigurations(nil, "")
	if len(out2) < 2 || out2[0] != "Debug" || out2[1] != "Release" {
		t.Fatalf("unexpected defaults: %v", out2)
	}
}

func TestDestinationItemsIncludesLocalAndTargets(t *testing.T) {
	sims := []SimulatorInfo{{Name: "iPhone 16", UDID: "SIM-1", State: "Booted", RuntimeName: "iOS 18.0", PlatformFamily: "ios"}}
	devs := []DeviceInfo{{Name: "My iPhone", Identifier: "DEV-1", OSVersion: "18.0", PlatformFamily: "ios"}}
	items := DestinationItems(sims, devs)
	if len(items) < 4 {
		t.Fatalf("expected at least 4 items, got %d", len(items))
	}
	if items[0].ID != "macos" || items[1].ID != "catalyst" {
		t.Fatalf("expected local mac entries first, got %+v %+v", items[0], items[1])
	}
	foundBooted := false
	foundDevice := false
	for _, it := range items {
		if it.ID == "SIM-1" && it.Meta == "[booted]" {
			foundBooted = true
		}
		if it.ID == "DEV-1" && it.Meta == "[device]" {
			foundDevice = true
		}
	}
	if !foundBooted || !foundDevice {
		t.Fatalf("missing simulator/device items: %v", items)
	}
}

func TestSelectorFilterItemsByScore(t *testing.T) {
	items := []SelectorItem{
		{ID: "1", Title: "iPhone 16", Description: "iOS"},
		{ID: "2", Title: "Apple TV", Description: "tvOS"},
		{ID: "3", Title: "Vision Pro", Description: "visionOS"},
	}
	m := NewSelector("Select", items, 120, DefaultStyles())
	m.input.SetValue("iph")
	m.filterItems()
	if len(m.filtered) == 0 {
		t.Fatalf("expected filtered results")
	}
	if m.filtered[0].ID != "1" {
		t.Fatalf("expected best match first, got %v", m.filtered)
	}
}
