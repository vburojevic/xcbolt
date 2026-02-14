package core

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type BuildResult struct {
	ResultBundle string        `json:"resultBundle"`
	ExitCode     int           `json:"exitCode"`
	Duration     time.Duration `json:"duration"`
	AppPath      string        `json:"appPath,omitempty"`
	BundleID     string        `json:"bundleId,omitempty"`
}

type RunResult struct {
	ResultBundle string `json:"resultBundle"`
	AppPath      string `json:"appPath"`
	BundleID     string `json:"bundleId"`
	PID          int    `json:"pid,omitempty"`
	Target       string `json:"target"`
	UDID         string `json:"udid,omitempty"`
}

type TestResult struct {
	ResultBundle string        `json:"resultBundle"`
	ExitCode     int           `json:"exitCode"`
	Duration     time.Duration `json:"duration"`
	Summary      TestSummary   `json:"summary"`
}

func EnsureBuildDirs(cfg Config) error {
	if cfg.DerivedDataPath != "" {
		if err := os.MkdirAll(cfg.DerivedDataPath, 0o755); err != nil {
			return err
		}
	}
	if cfg.ResultBundlesPath != "" {
		if err := os.MkdirAll(cfg.ResultBundlesPath, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func Build(ctx context.Context, projectRoot string, cfg Config, emit Emitter) (BuildResult, Config, error) {
	if cfg2, err := ensureSchemeAndConfigFromFS(projectRoot, cfg, emit); err == nil {
		cfg = cfg2
	} else {
		emitMaybe(emit, Err("build", ErrorObject{
			Code:       "SCHEME_REQUIRED",
			Message:    "No scheme configured",
			Detail:     err.Error(),
			Suggestion: "Run `xcbolt init` or pass --scheme.",
		}))
		return BuildResult{}, cfg, err
	}

	cfg, _ = ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit)
	cfg.Destination = normalizeDestination(cfg.Destination)

	if err := EnsureBuildDirs(cfg); err != nil {
		return BuildResult{}, cfg, err
	}

	bundlePath := filepath.Join(cfg.ResultBundlesPath, time.Now().Format("20060102-150405")+".xcresult")
	args := baseXcodebuildArgs(projectRoot, cfg)
	args = append(args,
		"-derivedDataPath", cfg.DerivedDataPath,
		"-resultBundlePath", bundlePath,
		"build",
	)
	args = append(args, cfg.Xcodebuild.Options...)

	emitMaybe(emit, Status("build", "Build started", map[string]any{"resultBundle": bundlePath}))
	if cfg.Xcodebuild.DryRun {
		cmdLine := formatCmd("xcodebuild", args)
		emitMaybe(emit, Log("build", "Dry run: "+cmdLine))
		emitMaybe(emit, Result("build", true, map[string]any{"exitCode": 0, "resultBundle": bundlePath, "dryRun": true}))
		cfg.LastResultBundle = bundlePath
		return BuildResult{ResultBundle: bundlePath, ExitCode: 0, Duration: 0}, cfg, nil
	}
	sink := newXcodebuildLogSink(ctx, "build", cfg, emit)
	res, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       append([]string{"xcodebuild"}, args...),
		Dir:        projectRoot,
		Env:        cfg.Xcodebuild.Env,
		StdoutLine: sink.HandleLine,
		StderrLine: sink.HandleLine,
	})
	sink.Finalize(err, res.ExitCode)

	cfg.LastResultBundle = bundlePath
	if err != nil {
		emitMaybe(emit, Err("build", ErrorObject{
			Code:       "XCODEBUILD_FAILED",
			Message:    "xcodebuild failed",
			Detail:     err.Error(),
			Suggestion: "Run with --json to capture structured logs, or open the .xcresult bundle for details.",
		}))
		emitMaybe(emit, Result("build", false, map[string]any{"exitCode": res.ExitCode, "resultBundle": bundlePath}))
		return BuildResult{ResultBundle: bundlePath, ExitCode: res.ExitCode, Duration: res.Duration}, cfg, err
	}

	var appPath string
	var bundleID string
	if settings, err := ShowBuildSettings(ctx, projectRoot, cfg); err == nil {
		if settings["PRODUCT_BUNDLE_IDENTIFIER"] != "" {
			bundleID = settings["PRODUCT_BUNDLE_IDENTIFIER"]
		}
		if p, err := guessAppBundlePath(settings); err == nil {
			appPath = p
		}
	}
	if appPath != "" {
		cfg.LastBuiltAppBundle = appPath
	}

	emitMaybe(emit, Result("build", true, map[string]any{
		"exitCode":     0,
		"resultBundle": bundlePath,
		"durationMs":   res.Duration.Milliseconds(),
		"bundleId":     bundleID,
		"appPath":      appPath,
	}))
	return BuildResult{ResultBundle: bundlePath, ExitCode: 0, Duration: res.Duration, AppPath: appPath, BundleID: bundleID}, cfg, nil
}

func Test(ctx context.Context, projectRoot string, cfg Config, onlyTesting []string, skipTesting []string, emit Emitter) (TestResult, Config, error) {
	if cfg2, err := ensureSchemeAndConfigFromFS(projectRoot, cfg, emit); err == nil {
		cfg = cfg2
	} else {
		emitMaybe(emit, Err("test", ErrorObject{
			Code:       "SCHEME_REQUIRED",
			Message:    "No scheme configured",
			Detail:     err.Error(),
			Suggestion: "Run `xcbolt init` or pass --scheme.",
		}))
		return TestResult{}, cfg, err
	}

	cfg, _ = ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit)
	cfg.Destination = normalizeDestination(cfg.Destination)

	if err := EnsureBuildDirs(cfg); err != nil {
		return TestResult{}, cfg, err
	}

	bundlePath := filepath.Join(cfg.ResultBundlesPath, time.Now().Format("20060102-150405")+".xcresult")
	args := baseXcodebuildArgs(projectRoot, cfg)
	args = append(args,
		"-derivedDataPath", cfg.DerivedDataPath,
		"-resultBundlePath", bundlePath,
	)
	for _, o := range onlyTesting {
		args = append(args, "-only-testing:"+o)
	}
	for _, s := range skipTesting {
		args = append(args, "-skip-testing:"+s)
	}
	args = append(args, "test")
	args = append(args, cfg.Xcodebuild.Options...)

	emitMaybe(emit, Status("test", "Tests started", map[string]any{"resultBundle": bundlePath}))
	if cfg.Xcodebuild.DryRun {
		cmdLine := formatCmd("xcodebuild", args)
		emitMaybe(emit, Log("test", "Dry run: "+cmdLine))
		emitMaybe(emit, Result("test", true, map[string]any{"exitCode": 0, "resultBundle": bundlePath, "dryRun": true}))
		cfg.LastResultBundle = bundlePath
		return TestResult{ResultBundle: bundlePath, ExitCode: 0, Duration: 0}, cfg, nil
	}
	sink := newXcodebuildLogSink(ctx, "test", cfg, emit)
	res, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       append([]string{"xcodebuild"}, args...),
		Dir:        projectRoot,
		Env:        cfg.Xcodebuild.Env,
		StdoutLine: sink.HandleLine,
		StderrLine: sink.HandleLine,
	})
	sink.Finalize(err, res.ExitCode)

	cfg.LastResultBundle = bundlePath

	summary, sumErr := XcresultTestSummary(ctx, bundlePath)
	if sumErr != nil {
		emitMaybe(emit, Warn("test", "Could not parse xcresult test summary: "+sumErr.Error()))
	}
	tr := TestResult{ResultBundle: bundlePath, ExitCode: res.ExitCode, Duration: res.Duration, Summary: summary}

	if err != nil {
		emitMaybe(emit, Err("test", ErrorObject{
			Code:       "XCODEBUILD_TEST_FAILED",
			Message:    "xcodebuild test failed",
			Detail:     err.Error(),
			Suggestion: "Inspect the .xcresult bundle for structured failures.",
		}))
		emitMaybe(emit, Result("test", false, map[string]any{"exitCode": res.ExitCode, "resultBundle": bundlePath, "durationMs": res.Duration.Milliseconds()}))
		return tr, cfg, err
	}
	// Note: tests can fail while xcodebuild exits non-zero; if err is nil, exit code is 0.
	emitMaybe(emit, Result("test", true, map[string]any{"exitCode": 0, "resultBundle": bundlePath, "durationMs": res.Duration.Milliseconds()}))
	return tr, cfg, nil
}

func Run(ctx context.Context, projectRoot string, cfg Config, console bool, emit Emitter) (RunResult, Config, error) {
	if cfg2, err := ensureSchemeAndConfigFromFS(projectRoot, cfg, emit); err == nil {
		cfg = cfg2
	} else {
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "SCHEME_REQUIRED",
			Message:    "No scheme configured",
			Detail:     err.Error(),
			Suggestion: "Run `xcbolt init` or pass --scheme.",
		}))
		return RunResult{}, cfg, err
	}

	if cfg2, err := ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit); err == nil {
		cfg = cfg2
	} else {
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "DESTINATION_REQUIRED",
			Message:    "No destination available",
			Detail:     err.Error(),
			Suggestion: "Select a simulator/device or create one with `xcbolt simulator`.",
		}))
		return RunResult{}, cfg, err
	}
	cfg.Destination = normalizeDestination(cfg.Destination)
	if cfg.Destination.Kind == DestAuto {
		err := errors.New("destination is still auto; unable to determine target")
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "DESTINATION_REQUIRED",
			Message:    "No destination configured",
			Detail:     err.Error(),
			Suggestion: "Pick a destination in TUI (d) or pass --platform + --target + --target-type.",
		}))
		return RunResult{}, cfg, err
	}
	emitMaybe(emit, Status("run", "Resolved destination", destinationMetadata(cfg.Destination)))
	if cfg.Xcodebuild.DryRun {
		args := baseXcodebuildArgs(projectRoot, cfg)
		args = append(args,
			"-derivedDataPath", cfg.DerivedDataPath,
			"-resultBundlePath", filepath.Join(cfg.ResultBundlesPath, time.Now().Format("20060102-150405")+".xcresult"),
			"build",
		)
		args = append(args, cfg.Xcodebuild.Options...)
		cmdLine := formatCmd("xcodebuild", args)
		emitMaybe(emit, Status("run", "Dry run enabled; skipping build/install/launch", nil))
		emitMaybe(emit, Log("run", "Dry run: "+cmdLine))
		return RunResult{Target: string(cfg.Destination.Kind), UDID: cfg.Destination.UDID}, cfg, nil
	}

	// Always build first (run implies build).
	buildRes, cfg, err := Build(ctx, projectRoot, cfg, emit)
	if err != nil {
		return RunResult{}, cfg, err
	}

	appPath := buildRes.AppPath
	if appPath != "" {
		if _, statErr := os.Stat(appPath); statErr != nil {
			appPath = ""
		}
	}
	if appPath == "" {
		settings, err := ShowBuildSettings(ctx, projectRoot, cfg)
		if err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "BUILD_SETTINGS_FAILED",
				Message:    "Failed to read build settings",
				Detail:     err.Error(),
				Suggestion: "Check scheme/configuration and destination.",
			}))
			return RunResult{}, cfg, err
		}

		appPath, err = guessAppBundlePath(settings)
		if err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "APP_BUNDLE_NOT_FOUND",
				Message:    "Unable to locate built app bundle",
				Detail:     err.Error(),
				Suggestion: "Ensure the scheme builds an app target.",
			}))
			return RunResult{}, cfg, err
		}
	}
	if _, statErr := os.Stat(appPath); statErr != nil {
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "APP_BUNDLE_MISSING",
			Message:    "Built app bundle is missing",
			Detail:     statErr.Error(),
			Suggestion: "Clean and rebuild, or verify the scheme produces an .app.",
		}))
		return RunResult{}, cfg, statErr
	}
	cfg.LastBuiltAppBundle = appPath

	appInfo, err := ReadAppBundleInfo(appPath)
	if err != nil {
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "APP_BUNDLE_INFO_FAILED",
			Message:    "Failed to read app Info.plist",
			Detail:     err.Error(),
			Suggestion: "Verify the built .app is valid.",
		}))
		return RunResult{}, cfg, err
	}
	if appInfo.BundleID == "" {
		err := errors.New("could not determine bundle id from Info.plist")
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "BUNDLE_ID_MISSING",
			Message:    "Bundle ID missing",
			Detail:     err.Error(),
			Suggestion: "Ensure PRODUCT_BUNDLE_IDENTIFIER is set for the app target.",
		}))
		return RunResult{}, cfg, err
	}

	launchEnv := consoleLaunchEnv(cfg, console)

	if cfg.Destination.PlatformFamily == PlatformWatchOS && cfg.Destination.Kind == DestDevice && strings.TrimSpace(cfg.Destination.CompanionTargetID) == "" {
		err := errors.New("watchOS physical runs require a paired companion target")
		emitMaybe(emit, Err("run", ErrorObject{
			Code:       "WATCH_COMPANION_REQUIRED",
			Message:    "Companion target required for watchOS device runs",
			Detail:     err.Error(),
			Suggestion: "Run with --companion-target <paired-iphone-udid-or-name>.",
		}))
		return RunResult{}, cfg, err
	}

	switch cfg.Destination.Kind {
	case DestSimulator:
		udid := cfg.Destination.UDID
		if udid == "" {
			return RunResult{}, cfg, errors.New("missing simulator udid")
		}
		emitMaybe(emit, Status("run", "Booting simulator", map[string]any{"udid": udid}))
		_ = SimctlBoot(ctx, udid)
		_ = SimctlOpenSimulatorApp(ctx)
		if err := SimctlBootStatus(ctx, udid); err != nil {
			return RunResult{}, cfg, err
		}
		emitMaybe(emit, Status("run", "Installing app", map[string]any{"app": appPath}))
		if _, err := RunStreaming(ctx, CmdSpec{
			Path:       "xcrun",
			Args:       []string{"simctl", "install", udid, appPath},
			StdoutLine: func(s string) { emitMaybe(emit, Log("run", s)) },
			StderrLine: func(s string) { emitMaybe(emit, Log("run", s)) },
		}); err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "SIM_INSTALL_FAILED",
				Message:    "Failed to install app on simulator",
				Detail:     err.Error(),
				Suggestion: "Try resetting the simulator or cleaning DerivedData.",
			}))
			return RunResult{}, cfg, err
		}

		var logCancel context.CancelFunc
		if console && shouldStreamUnifiedLogs(cfg) {
			predicate := simLogPredicate(cfg, appInfo)
			if predicate != "" {
				logCtx, cancel := context.WithCancel(ctx)
				logCancel = cancel
				go func() {
					if err := SimctlLogStream(logCtx, udid, predicate, emit); err != nil && !errors.Is(err, context.Canceled) {
						emitMaybe(emit, Warn("run", "simctl log stream failed: "+err.Error()))
					}
				}()
			}
		}

		launchArgs := []string{"simctl", "launch"}
		if console {
			launchArgs = append(launchArgs, "--console")
		}
		launchArgs = append(launchArgs, udid, appInfo.BundleID)
		// Remaining arguments are passed to the app.
		launchArgs = append(launchArgs, cfg.Launch.Options...)

		emitMaybe(emit, Status("run", "Launching app", map[string]any{"bundleId": appInfo.BundleID}))
		var out strings.Builder
		var errOut strings.Builder
		res, err := RunStreaming(ctx, CmdSpec{
			Path: "xcrun",
			Args: launchArgs,
			Env:  simctlChildEnv(launchEnv),
			StdoutLine: func(s string) {
				out.WriteString(s)
				out.WriteString("\n")
				if msg, ok := formatAppConsoleLine(appInfo, 0, false, s, !shouldStreamSystemLogs(cfg), shouldStreamUnifiedLogs(cfg)); ok {
					emitMaybe(emit, LogStream("run", msg, "app"))
				}
			},
			StderrLine: func(s string) {
				errOut.WriteString(s)
				errOut.WriteString("\n")
				if msg, ok := formatAppConsoleLine(appInfo, 0, true, s, !shouldStreamSystemLogs(cfg), shouldStreamUnifiedLogs(cfg)); ok {
					emitMaybe(emit, LogStream("run", msg, "app"))
				}
			},
		})
		if logCancel != nil {
			logCancel()
		}
		pid := parseSimctlLaunchPID(out.String())
		if err != nil {
			if errors.Is(err, context.Canceled) {
				emitMaybe(emit, Status("run", "Run canceled", map[string]any{"bundleId": appInfo.BundleID}))
				return RunResult{}, cfg, err
			}
			if pid > 0 {
				emitMaybe(emit, Status("run", "App exited", map[string]any{"pid": pid, "bundleId": appInfo.BundleID, "exitCode": res.ExitCode}))
				emitMaybe(emit, Result("run", true, map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
				return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: pid, Target: "simulator", UDID: udid}, cfg, nil
			}
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "SIM_LAUNCH_FAILED",
				Message:    "Failed to launch app on simulator",
				Detail:     err.Error(),
				Suggestion: "Check simulator state and app bundle id.",
			}))
			return RunResult{}, cfg, err
		}
		if pid == 0 {
			pid = parseFirstInt(out.String())
		}
		dst := cfg.Destination
		dst.ID = udid
		dst.UDID = udid
		_, _ = AddSessionWithDestination(projectRoot, appInfo.BundleID, pid, dst)
		emitMaybe(emit, Status("run", "Running", map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
		emitMaybe(emit, Result("run", true, map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
		return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: pid, Target: "simulator", UDID: udid}, cfg, nil

	case DestDevice:
		udid := cfg.Destination.UDID
		if udid == "" {
			return RunResult{}, cfg, errors.New("missing device udid")
		}
		if cfg.Destination.PlatformFamily == PlatformWatchOS {
			watchDeploy, err := resolveWatchDeviceDeployment(ctx, cfg.Destination, appPath, appInfo, emit)
			if err != nil {
				emitMaybe(emit, Err("run", ErrorObject{
					Code:       "WATCH_DEPLOYMENT_FAILED",
					Message:    "Failed to prepare watchOS companion deployment",
					Detail:     err.Error(),
					Suggestion: "Provide --companion-target and ensure build outputs both iPhone and Watch .app bundles.",
				}))
				return RunResult{}, cfg, err
			}
			cfg.Destination.CompanionTargetID = watchDeploy.CompanionDeviceID

			emitMaybe(emit, Status("run", "Installing companion app on paired iPhone", map[string]any{
				"companionDevice": watchDeploy.CompanionDeviceID,
				"app":             watchDeploy.CompanionAppPath,
			}))
			if err := DevicectlInstallApp(ctx, watchDeploy.CompanionDeviceID, watchDeploy.CompanionAppPath, emit); err != nil {
				emitMaybe(emit, Err("run", ErrorObject{
					Code:       "WATCH_COMPANION_INSTALL_FAILED",
					Message:    "Failed to install companion app on paired iPhone",
					Detail:     err.Error(),
					Suggestion: "Ensure paired iPhone is connected/unlocked and signing is valid.",
				}))
				return RunResult{}, cfg, err
			}

			emitMaybe(emit, Status("run", "Installing watch app on device", map[string]any{"udid": udid, "app": watchDeploy.WatchAppPath}))
			if err := DevicectlInstallApp(ctx, udid, watchDeploy.WatchAppPath, emit); err != nil {
				emitMaybe(emit, Err("run", ErrorObject{
					Code:       "WATCH_INSTALL_FAILED",
					Message:    "Failed to install watch app on watch device",
					Detail:     err.Error(),
					Suggestion: "Verify the watch target bundle is built and the watch is paired/unlocked.",
				}))
				return RunResult{}, cfg, err
			}

			emitMaybe(emit, Status("run", "Launching watch app on device", map[string]any{"bundleId": watchDeploy.WatchInfo.BundleID, "console": console}))
			lr, err := DevicectlLaunchApp(ctx, udid, watchDeploy.WatchInfo.BundleID, console, launchEnv, watchDeploy.WatchInfo, !shouldStreamSystemLogs(cfg), emit)
			if err != nil {
				emitMaybe(emit, Err("run", ErrorObject{
					Code:       "WATCH_LAUNCH_FAILED",
					Message:    "Failed to launch watch app on device",
					Detail:     err.Error(),
					Suggestion: "Check watch device connectivity and companion installation state.",
				}))
				return RunResult{}, cfg, err
			}
			dst := cfg.Destination
			dst.ID = udid
			dst.UDID = udid
			dst.CompanionTargetID = watchDeploy.CompanionDeviceID
			dst.CompanionBundleID = watchDeploy.CompanionInfo.BundleID
			_, _ = AddSessionWithDestination(projectRoot, watchDeploy.WatchInfo.BundleID, lr.PID, dst)
			emitMaybe(emit, Status("run", "Running", map[string]any{"pid": lr.PID, "bundleId": watchDeploy.WatchInfo.BundleID}))
			emitMaybe(emit, Result("run", true, map[string]any{"pid": lr.PID, "bundleId": watchDeploy.WatchInfo.BundleID, "companionTargetId": watchDeploy.CompanionDeviceID}))
			return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: watchDeploy.WatchAppPath, BundleID: watchDeploy.WatchInfo.BundleID, PID: lr.PID, Target: "device", UDID: udid}, cfg, nil
		}
		emitMaybe(emit, Status("run", "Installing app on device", map[string]any{"udid": udid}))
		if err := DevicectlInstallApp(ctx, udid, appPath, emit); err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "DEVICE_INSTALL_FAILED",
				Message:    "Failed to install app on device",
				Detail:     err.Error(),
				Suggestion: "Ensure device is trusted/unlocked and provisioning is valid.",
			}))
			return RunResult{}, cfg, err
		}
		emitMaybe(emit, Status("run", "Launching app on device", map[string]any{"bundleId": appInfo.BundleID, "console": console}))
		lr, err := DevicectlLaunchApp(ctx, udid, appInfo.BundleID, console, launchEnv, appInfo, !shouldStreamSystemLogs(cfg), emit)
		if err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "DEVICE_LAUNCH_FAILED",
				Message:    "Failed to launch app on device",
				Detail:     err.Error(),
				Suggestion: "Check device logs and app signing.",
			}))
			return RunResult{}, cfg, err
		}
		dst := cfg.Destination
		dst.ID = udid
		dst.UDID = udid
		_, _ = AddSessionWithDestination(projectRoot, appInfo.BundleID, lr.PID, dst)
		emitMaybe(emit, Status("run", "Running", map[string]any{"pid": lr.PID, "bundleId": appInfo.BundleID}))
		emitMaybe(emit, Result("run", true, map[string]any{"pid": lr.PID, "bundleId": appInfo.BundleID}))
		return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: lr.PID, Target: "device", UDID: udid}, cfg, nil

	case DestMacOS, DestCatalyst:
		if appInfo.Executable == "" {
			err := errors.New("missing CFBundleExecutable in app bundle")
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "APP_EXECUTABLE_MISSING",
				Message:    "Missing app executable",
				Detail:     err.Error(),
				Suggestion: "Ensure the target builds a macOS app bundle.",
			}))
			return RunResult{}, cfg, err
		}
		execPath := filepath.Join(appPath, "Contents", "MacOS", appInfo.Executable)
		if _, statErr := os.Stat(execPath); statErr != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "APP_EXECUTABLE_MISSING",
				Message:    "App executable not found",
				Detail:     statErr.Error(),
				Suggestion: "Ensure the target builds a macOS app bundle.",
			}))
			return RunResult{}, cfg, statErr
		}

		emitMaybe(emit, Status("run", "Launching app on Mac", map[string]any{"app": appPath}))
		cmd := exec.Command(execPath, cfg.Launch.Options...)
		cmd.Env = mergeEnv(os.Environ(), launchEnv)
		if err := cmd.Start(); err != nil {
			emitMaybe(emit, Err("run", ErrorObject{
				Code:       "MAC_LAUNCH_FAILED",
				Message:    "Failed to launch app on Mac",
				Detail:     err.Error(),
				Suggestion: "Check app bundle and permissions.",
			}))
			return RunResult{}, cfg, err
		}
		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
			_ = cmd.Process.Release()
		}
		_, _ = AddSessionWithDestination(projectRoot, appInfo.BundleID, pid, cfg.Destination)
		emitMaybe(emit, Status("run", "Running", map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
		emitMaybe(emit, Result("run", true, map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
		return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: pid, Target: string(cfg.Destination.Kind)}, cfg, nil

	default:
		return RunResult{}, cfg, fmt.Errorf("run not implemented for destination kind %q", cfg.Destination.Kind)
	}
}

func consoleLaunchEnv(cfg Config, console bool) map[string]string {
	env := map[string]string{}
	for k, v := range cfg.Launch.Env {
		env[k] = v
	}
	if !console {
		return env
	}
	if _, disabled := env["IDE_DISABLED_OS_ACTIVITY_DT_MODE"]; !disabled {
		if _, ok := env["OS_ACTIVITY_DT_MODE"]; !ok {
			env["OS_ACTIVITY_DT_MODE"] = "enable"
		}
	}
	if _, ok := env["NSUnbufferedIO"]; !ok {
		env["NSUnbufferedIO"] = "YES"
	}
	return env
}

func shouldStreamUnifiedLogs(cfg Config) bool {
	if cfg.Launch.StreamUnifiedLogs == nil {
		return true
	}
	return *cfg.Launch.StreamUnifiedLogs
}

func shouldStreamSystemLogs(cfg Config) bool {
	if cfg.Launch.StreamSystemLogs == nil {
		return false
	}
	return *cfg.Launch.StreamSystemLogs
}

func simLogPredicate(cfg Config, info AppBundleInfo) string {
	exec := info.Executable
	bundle := info.BundleID
	systemLogs := shouldStreamSystemLogs(cfg)

	if exec != "" {
		if systemLogs {
			return fmt.Sprintf("process == \"%s\"", exec)
		}
		if bundle != "" {
			return fmt.Sprintf("process == \"%s\" AND (subsystem == \"%s\" OR subsystem BEGINSWITH \"%s.\")", exec, bundle, bundle)
		}
		return fmt.Sprintf("process == \"%s\"", exec)
	}

	if bundle != "" {
		return fmt.Sprintf("subsystem == \"%s\" OR subsystem BEGINSWITH \"%s.\"", bundle, bundle)
	}
	return ""
}

func simctlChildEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range env {
		out["SIMCTL_CHILD_"+k] = v
	}
	return out
}

var mirroredUnifiedLogRE = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2}\.\d{3})\s+(\S+)\s+(\S+)\[([^\]]+)\]\s+\[([^\]]+)\]\s+(.*)$`)
var simctlLaunchPIDRE = regexp.MustCompile(`(?i)\bpid\b[^0-9]*([0-9]+)`)

type mirroredUnifiedLine struct {
	Time      string
	Level     string
	Process   string
	PidThread string
	Subsystem string
	Message   string
}

func parseMirroredUnifiedLog(line string) (mirroredUnifiedLine, bool) {
	m := mirroredUnifiedLogRE.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) != 8 {
		return mirroredUnifiedLine{}, false
	}
	return mirroredUnifiedLine{
		Time:      m[2],
		Level:     m[3],
		Process:   m[4],
		PidThread: m[5],
		Subsystem: m[6],
		Message:   m[7],
	}, true
}

func parseSimctlLaunchPID(output string) int {
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(strings.ToLower(line), "pid") {
			continue
		}
		if m := simctlLaunchPIDRE.FindStringSubmatch(line); len(m) == 2 {
			var pid int
			_, _ = fmt.Sscanf(m[1], "%d", &pid)
			if pid > 0 {
				return pid
			}
		}
	}
	return 0
}

func formatMirroredUnifiedLine(m mirroredUnifiedLine) string {
	level := m.Level
	if level == "" {
		level = "I"
	}
	return fmt.Sprintf("%s %s %s[%s]\n%s", m.Time, level, m.Process, m.PidThread, m.Message)
}

func formatAppConsoleLine(info AppBundleInfo, pid int, stderr bool, line string, filterSystem bool, dedupeUnified bool) (string, bool) {
	if line == "" {
		return "", false
	}
	if mirrored, ok := parseMirroredUnifiedLog(line); ok {
		if filterSystem && !subsystemMatchesApp(info, mirrored.Subsystem) {
			return "", false
		}
		if dedupeUnified {
			return "", false
		}
		return formatMirroredUnifiedLine(mirrored), true
	}

	level := appConsoleLevel(stderr, line)
	proc := appDisplayName(info)
	pidPart := "0"
	if pid > 0 {
		pidPart = fmt.Sprintf("%d", pid)
	}
	timePart := time.Now().Format("15:04:05.000")
	return fmt.Sprintf("%s %s %s[%s:0]\n%s", timePart, level, proc, pidPart, strings.TrimSpace(line)), true
}

func appConsoleLevel(stderr bool, line string) string {
	if !stderr {
		return "I"
	}
	if isFatalAppLogLine(line) {
		return "F"
	}
	// Most stderr lines are non-fatal; classify as warning to avoid false "errors".
	return "W"
}

func isFatalAppLogLine(line string) bool {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "fatal error"),
		strings.Contains(lower, "terminating app due to uncaught exception"),
		strings.Contains(lower, "uncaught exception"),
		strings.Contains(lower, "precondition failed"),
		strings.Contains(lower, "assertion failed"),
		strings.Contains(lower, "libc++abi: terminating"),
		strings.Contains(lower, "sigabrt"),
		strings.Contains(lower, "sigsegv"),
		strings.Contains(lower, "abort() called"),
		strings.Contains(lower, "dyld: library not loaded"):
		return true
	}
	return false
}

func subsystemMatchesApp(info AppBundleInfo, subsystem string) bool {
	if subsystem == "" {
		return false
	}
	bundle := info.BundleID
	if bundle == "" {
		return true
	}
	return subsystem == bundle || strings.HasPrefix(subsystem, bundle+".")
}

func appDisplayName(info AppBundleInfo) string {
	if info.DisplayName != "" {
		return info.DisplayName
	}
	if info.BundleName != "" {
		return info.BundleName
	}
	if info.Executable != "" {
		return info.Executable
	}
	if info.BundleID != "" {
		return info.BundleID
	}
	return "App"
}

func baseXcodebuildArgs(projectRoot string, cfg Config) []string {
	args := []string{}
	if cfg.Workspace != "" {
		args = append(args, "-workspace", filepath.Join(projectRoot, cfg.Workspace))
	} else if cfg.Project != "" {
		args = append(args, "-project", filepath.Join(projectRoot, cfg.Project))
	}
	if cfg.Scheme != "" {
		args = append(args, "-scheme", cfg.Scheme)
	}
	if cfg.Configuration != "" {
		args = append(args, "-configuration", cfg.Configuration)
	}
	if dest := BuildDestinationString(cfg); dest != "" {
		args = append(args, "-destination", dest)
	}
	return args
}

func formatCmd(path string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, path)
	for _, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			a = strings.ReplaceAll(a, "\"", "\\\"")
			parts = append(parts, "\""+a+"\"")
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

func emitMaybe(e Emitter, ev Event) {
	if e != nil {
		e.Emit(ev)
	}
}

func guessAppBundlePath(settings BuildSettings) (string, error) {
	buildDir := settings["TARGET_BUILD_DIR"]
	if buildDir == "" {
		buildDir = settings["BUILT_PRODUCTS_DIR"]
	}
	if buildDir == "" {
		return "", errors.New("could not determine TARGET_BUILD_DIR from build settings")
	}
	name := settings["WRAPPER_NAME"]
	if name == "" {
		name = settings["FULL_PRODUCT_NAME"]
	}
	if name == "" {
		prod := settings["PRODUCT_NAME"]
		if prod != "" {
			name = prod + ".app"
		}
	}
	if name == "" {
		return "", errors.New("could not determine app bundle name from build settings")
	}
	return filepath.Join(buildDir, name), nil
}

type watchDeviceDeployment struct {
	CompanionDeviceID string
	CompanionAppPath  string
	CompanionInfo     AppBundleInfo
	WatchAppPath      string
	WatchInfo         AppBundleInfo
}

var resolveCompanionDeviceIDForWatch = resolveCompanionDeviceID

func resolveWatchDeviceDeployment(ctx context.Context, dst Destination, builtAppPath string, builtInfo AppBundleInfo, emit Emitter) (watchDeviceDeployment, error) {
	companionTarget := strings.TrimSpace(dst.CompanionTargetID)
	if companionTarget == "" {
		return watchDeviceDeployment{}, errors.New("missing companion target")
	}
	companionDeviceID, err := resolveCompanionDeviceIDForWatch(ctx, companionTarget, emit)
	if err != nil {
		return watchDeviceDeployment{}, err
	}

	companionAppPath := builtAppPath
	companionInfo := builtInfo
	watchAppPath := builtAppPath
	watchInfo := builtInfo

	if builtInfo.IsWatchApp {
		watchAppPath = builtAppPath
		watchInfo = builtInfo
		companionAppPath, companionInfo, err = findCompanionAppNearWatch(builtAppPath, builtInfo.CompanionBundleID)
		if err != nil {
			return watchDeviceDeployment{}, err
		}
	} else {
		companionAppPath = builtAppPath
		companionInfo = builtInfo
		watchAppPath, watchInfo, err = findWatchAppForCompanion(companionAppPath, companionInfo.BundleID)
		if err != nil {
			return watchDeviceDeployment{}, err
		}
	}

	if watchInfo.BundleID == "" {
		return watchDeviceDeployment{}, errors.New("watch app bundle id is missing")
	}
	if companionInfo.BundleID == "" {
		return watchDeviceDeployment{}, errors.New("companion app bundle id is missing")
	}
	if watchInfo.CompanionBundleID != "" && watchInfo.CompanionBundleID != companionInfo.BundleID {
		return watchDeviceDeployment{}, fmt.Errorf("watch app companion id %q does not match selected companion %q", watchInfo.CompanionBundleID, companionInfo.BundleID)
	}

	return watchDeviceDeployment{
		CompanionDeviceID: companionDeviceID,
		CompanionAppPath:  companionAppPath,
		CompanionInfo:     companionInfo,
		WatchAppPath:      watchAppPath,
		WatchInfo:         watchInfo,
	}, nil
}

func resolveCompanionDeviceID(ctx context.Context, target string, emit Emitter) (string, error) {
	candidates, err := ListDestinationCandidates(ctx, emit)
	if err != nil {
		return "", err
	}
	matches := []DestinationCandidate{}
	for _, c := range candidates {
		if c.TargetType != TargetDevice {
			continue
		}
		if c.PlatformFamily != PlatformIOS && c.PlatformFamily != PlatformIPadOS {
			continue
		}
		if strings.EqualFold(c.ID, target) || strings.EqualFold(c.Name, target) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("companion target %q not found among connected iPhone/iPad devices", target)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.Name+" ("+m.ID+")")
		}
		return "", fmt.Errorf("companion target %q is ambiguous: %s", target, strings.Join(names, ", "))
	}
	return matches[0].ID, nil
}

func findCompanionAppNearWatch(watchAppPath, companionBundleID string) (string, AppBundleInfo, error) {
	buildDir := filepath.Dir(watchAppPath)
	paths, err := discoverAppBundles(buildDir, 4)
	if err != nil {
		return "", AppBundleInfo{}, err
	}
	candidates := []struct {
		path string
		info AppBundleInfo
	}{}
	for _, p := range paths {
		if p == watchAppPath {
			continue
		}
		info, err := ReadAppBundleInfo(p)
		if err != nil || info.BundleID == "" || info.IsWatchApp {
			continue
		}
		candidates = append(candidates, struct {
			path string
			info AppBundleInfo
		}{path: p, info: info})
	}
	if companionBundleID != "" {
		for _, c := range candidates {
			if c.info.BundleID == companionBundleID {
				return c.path, c.info, nil
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0].path, candidates[0].info, nil
	}
	return "", AppBundleInfo{}, errors.New("unable to resolve companion iOS app bundle near watch app")
}

func findWatchAppForCompanion(companionAppPath, companionBundleID string) (string, AppBundleInfo, error) {
	watchDir := filepath.Join(companionAppPath, "Watch")
	nested, _ := discoverAppBundles(watchDir, 2)
	for _, p := range nested {
		info, err := ReadAppBundleInfo(p)
		if err != nil || !info.IsWatchApp {
			continue
		}
		if info.CompanionBundleID != "" && companionBundleID != "" && info.CompanionBundleID != companionBundleID {
			continue
		}
		return p, info, nil
	}

	buildDir := filepath.Dir(companionAppPath)
	paths, err := discoverAppBundles(buildDir, 4)
	if err != nil {
		return "", AppBundleInfo{}, err
	}
	candidates := []struct {
		path string
		info AppBundleInfo
	}{}
	for _, p := range paths {
		info, err := ReadAppBundleInfo(p)
		if err != nil || !info.IsWatchApp {
			continue
		}
		if info.CompanionBundleID != "" && companionBundleID != "" && info.CompanionBundleID != companionBundleID {
			continue
		}
		candidates = append(candidates, struct {
			path string
			info AppBundleInfo
		}{path: p, info: info})
	}
	if len(candidates) == 1 {
		return candidates[0].path, candidates[0].info, nil
	}
	if len(candidates) == 0 {
		return "", AppBundleInfo{}, errors.New("watch app bundle not found (expected companion/Watch/*.app or nearby watch app output)")
	}
	return "", AppBundleInfo{}, errors.New("multiple watch app bundles found; refine scheme/build output")
}

func discoverAppBundles(root string, maxDepth int) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}
	root = filepath.Clean(root)
	out := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && rel != "." {
			depth := strings.Count(rel, string(os.PathSeparator)) + 1
			if depth > maxDepth {
				return filepath.SkipDir
			}
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".app") {
			out = append(out, path)
			return filepath.SkipDir
		}
		return nil
	})
	return out, err
}
