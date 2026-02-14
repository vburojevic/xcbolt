package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"howett.net/plist"
)

func writeTestAppBundle(t *testing.T, appPath, bundleID string, isWatch bool, companionBundleID string) AppBundleInfo {
	t.Helper()
	if err := os.MkdirAll(appPath, 0o755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	m := map[string]any{
		"CFBundleIdentifier": bundleID,
		"CFBundleName":       "TestApp",
		"CFBundleExecutable": "TestExec",
	}
	if isWatch {
		m["WKWatchKitApp"] = true
	}
	if companionBundleID != "" {
		m["WKCompanionAppBundleIdentifier"] = companionBundleID
	}
	b, err := plist.Marshal(m, plist.XMLFormat)
	if err != nil {
		t.Fatalf("marshal plist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appPath, "Info.plist"), b, 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}
	info, err := ReadAppBundleInfo(appPath)
	if err != nil {
		t.Fatalf("read app info: %v", err)
	}
	return info
}

func TestDiscoverAppBundlesDepth(t *testing.T) {
	root := t.TempDir()
	level1 := filepath.Join(root, "A.app")
	level4 := filepath.Join(root, "a", "b", "c", "d", "Deep.app")
	_ = os.MkdirAll(level1, 0o755)
	_ = os.MkdirAll(level4, 0o755)

	bundles, err := discoverAppBundles(root, 3)
	if err != nil {
		t.Fatalf("discoverAppBundles: %v", err)
	}
	if len(bundles) != 1 || bundles[0] != level1 {
		t.Fatalf("unexpected bundles: %v", bundles)
	}
}

func TestFindWatchAppForCompanionNested(t *testing.T) {
	root := t.TempDir()
	companionPath := filepath.Join(root, "Products", "Phone.app")
	companionBundleID := "com.example.phone"
	writeTestAppBundle(t, companionPath, companionBundleID, false, "")
	watchPath := filepath.Join(companionPath, "Watch", "WatchApp.app")
	writeTestAppBundle(t, watchPath, "com.example.watch", true, companionBundleID)

	gotPath, gotInfo, err := findWatchAppForCompanion(companionPath, companionBundleID)
	if err != nil {
		t.Fatalf("findWatchAppForCompanion: %v", err)
	}
	if gotPath != watchPath {
		t.Fatalf("path = %q, want %q", gotPath, watchPath)
	}
	if !gotInfo.IsWatchApp {
		t.Fatalf("expected watch app info")
	}
}

func TestFindCompanionAppNearWatchByBundleID(t *testing.T) {
	root := t.TempDir()
	products := filepath.Join(root, "Products")
	watchPath := filepath.Join(products, "WatchOnly.app")
	writeTestAppBundle(t, watchPath, "com.example.watch", true, "com.example.phone")
	phonePath := filepath.Join(products, "Phone.app")
	writeTestAppBundle(t, phonePath, "com.example.phone", false, "")
	otherPath := filepath.Join(products, "Other.app")
	writeTestAppBundle(t, otherPath, "com.other", false, "")

	gotPath, gotInfo, err := findCompanionAppNearWatch(watchPath, "com.example.phone")
	if err != nil {
		t.Fatalf("findCompanionAppNearWatch: %v", err)
	}
	if gotPath != phonePath {
		t.Fatalf("path = %q, want %q", gotPath, phonePath)
	}
	if gotInfo.BundleID != "com.example.phone" {
		t.Fatalf("bundle id = %q", gotInfo.BundleID)
	}
}

func TestResolveWatchDeviceDeploymentFromCompanionBuild(t *testing.T) {
	root := t.TempDir()
	companionPath := filepath.Join(root, "Build", "Phone.app")
	companionBundleID := "com.example.phone"
	companionInfo := writeTestAppBundle(t, companionPath, companionBundleID, false, "")
	watchPath := filepath.Join(companionPath, "Watch", "Watch.app")
	watchBundleID := "com.example.watch"
	writeTestAppBundle(t, watchPath, watchBundleID, true, companionBundleID)

	oldResolver := resolveCompanionDeviceIDForWatch
	resolveCompanionDeviceIDForWatch = func(ctx context.Context, target string, emit Emitter) (string, error) {
		if target != "my-phone" {
			t.Fatalf("target = %q", target)
		}
		return "iphone-udid", nil
	}
	defer func() { resolveCompanionDeviceIDForWatch = oldResolver }()

	dst := Destination{PlatformFamily: PlatformWatchOS, TargetType: TargetDevice, CompanionTargetID: "my-phone"}
	dep, err := resolveWatchDeviceDeployment(context.Background(), dst, companionPath, companionInfo, nil)
	if err != nil {
		t.Fatalf("resolveWatchDeviceDeployment: %v", err)
	}
	if dep.CompanionDeviceID != "iphone-udid" {
		t.Fatalf("companion device = %q", dep.CompanionDeviceID)
	}
	if dep.WatchInfo.BundleID != watchBundleID {
		t.Fatalf("watch bundle = %q", dep.WatchInfo.BundleID)
	}
	if dep.CompanionInfo.BundleID != companionBundleID {
		t.Fatalf("companion bundle = %q", dep.CompanionInfo.BundleID)
	}
}

func TestResolveWatchDeviceDeploymentFromWatchBuild(t *testing.T) {
	root := t.TempDir()
	products := filepath.Join(root, "Products")
	watchPath := filepath.Join(products, "Watch.app")
	companionBundleID := "com.example.phone"
	watchInfo := writeTestAppBundle(t, watchPath, "com.example.watch", true, companionBundleID)
	companionPath := filepath.Join(products, "Phone.app")
	writeTestAppBundle(t, companionPath, companionBundleID, false, "")

	oldResolver := resolveCompanionDeviceIDForWatch
	resolveCompanionDeviceIDForWatch = func(ctx context.Context, target string, emit Emitter) (string, error) {
		return "iphone-udid", nil
	}
	defer func() { resolveCompanionDeviceIDForWatch = oldResolver }()

	dst := Destination{PlatformFamily: PlatformWatchOS, TargetType: TargetDevice, CompanionTargetID: "my-phone"}
	dep, err := resolveWatchDeviceDeployment(context.Background(), dst, watchPath, watchInfo, nil)
	if err != nil {
		t.Fatalf("resolveWatchDeviceDeployment: %v", err)
	}
	if dep.CompanionAppPath != companionPath {
		t.Fatalf("companion app path = %q", dep.CompanionAppPath)
	}
	if dep.WatchAppPath != watchPath {
		t.Fatalf("watch app path = %q", dep.WatchAppPath)
	}
}

func TestResolveWatchDeviceDeploymentFailsOnMismatch(t *testing.T) {
	root := t.TempDir()
	products := filepath.Join(root, "Products")
	watchPath := filepath.Join(products, "Watch.app")
	watchInfo := writeTestAppBundle(t, watchPath, "com.example.watch", true, "com.example.phone")
	companionPath := filepath.Join(products, "Phone.app")
	writeTestAppBundle(t, companionPath, "com.other.phone", false, "")

	oldResolver := resolveCompanionDeviceIDForWatch
	resolveCompanionDeviceIDForWatch = func(ctx context.Context, target string, emit Emitter) (string, error) {
		return "iphone-udid", nil
	}
	defer func() { resolveCompanionDeviceIDForWatch = oldResolver }()

	dst := Destination{PlatformFamily: PlatformWatchOS, TargetType: TargetDevice, CompanionTargetID: "my-phone"}
	_, err := resolveWatchDeviceDeployment(context.Background(), dst, watchPath, watchInfo, nil)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestResolveWatchDeviceDeploymentRequiresCompanionTarget(t *testing.T) {
	root := t.TempDir()
	companionPath := filepath.Join(root, "Phone.app")
	companionInfo := writeTestAppBundle(t, companionPath, "com.example.phone", false, "")

	dst := Destination{PlatformFamily: PlatformWatchOS, TargetType: TargetDevice}
	_, err := resolveWatchDeviceDeployment(context.Background(), dst, companionPath, companionInfo, nil)
	if err == nil {
		t.Fatalf("expected missing companion target error")
	}
}
