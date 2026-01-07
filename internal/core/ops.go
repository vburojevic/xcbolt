package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BuildResult struct {
	ResultBundle string        `json:"resultBundle"`
	ExitCode     int           `json:"exitCode"`
	Duration     time.Duration `json:"duration"`
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
	cfg, _ = ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit)

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
	sink := newXcodebuildLogSink(ctx, "build", cfg, emit)
	res, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       append([]string{"xcodebuild"}, args...),
		Dir:        projectRoot,
		Env:        cfg.Xcodebuild.Env,
		StdoutLine: sink.HandleLine,
		StderrLine: sink.HandleLine,
	})
	sink.Close()

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

	emitMaybe(emit, Result("build", true, map[string]any{"exitCode": 0, "resultBundle": bundlePath, "durationMs": res.Duration.Milliseconds()}))
	return BuildResult{ResultBundle: bundlePath, ExitCode: 0, Duration: res.Duration}, cfg, nil
}

func Test(ctx context.Context, projectRoot string, cfg Config, onlyTesting []string, skipTesting []string, emit Emitter) (TestResult, Config, error) {
	cfg, _ = ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit)

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
	sink := newXcodebuildLogSink(ctx, "test", cfg, emit)
	res, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       append([]string{"xcodebuild"}, args...),
		Dir:        projectRoot,
		Env:        cfg.Xcodebuild.Env,
		StdoutLine: sink.HandleLine,
		StderrLine: sink.HandleLine,
	})
	sink.Close()

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
	cfg, _ = ResolveDestinationIfNeeded(ctx, projectRoot, cfg, emit)

	// Always build first (run implies build).
	_, cfg, err := Build(ctx, projectRoot, cfg, emit)
	if err != nil {
		return RunResult{}, cfg, err
	}

	settings, err := ShowBuildSettings(ctx, projectRoot, cfg)
	if err != nil {
		return RunResult{}, cfg, err
	}

	appPath, err := guessAppBundlePath(settings)
	if err != nil {
		return RunResult{}, cfg, err
	}
	cfg.LastBuiltAppBundle = appPath

	appInfo, err := ReadAppBundleInfo(appPath)
	if err != nil {
		return RunResult{}, cfg, err
	}
	if appInfo.BundleID == "" {
		return RunResult{}, cfg, errors.New("could not determine bundle id from Info.plist")
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
			return RunResult{}, cfg, err
		}

		launchArgs := []string{"simctl", "launch"}
		if console {
			launchArgs = append(launchArgs, "--console")
		}
		// env support
		for k, v := range cfg.Launch.Env {
			launchArgs = append(launchArgs, "--env", fmt.Sprintf("%s=%s", k, v))
		}
		launchArgs = append(launchArgs, udid, appInfo.BundleID)
		// Remaining arguments are passed to the app.
		launchArgs = append(launchArgs, cfg.Launch.Options...)

		emitMaybe(emit, Status("run", "Launching app", map[string]any{"bundleId": appInfo.BundleID}))
		var out strings.Builder
		res, err := RunStreaming(ctx, CmdSpec{
			Path: "xcrun",
			Args: launchArgs,
			StdoutLine: func(s string) {
				out.WriteString(s)
				out.WriteString("\n")
				emitMaybe(emit, Log("run", s))
			},
			StderrLine: func(s string) { emitMaybe(emit, Log("run", s)) },
		})
		_ = res
		if err != nil {
			return RunResult{}, cfg, err
		}
		pid := parseFirstInt(out.String())
		_, _ = AddSession(projectRoot, appInfo.BundleID, pid, "simulator", udid)
		emitMaybe(emit, Result("run", true, map[string]any{"pid": pid, "bundleId": appInfo.BundleID}))
		return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: pid, Target: "simulator", UDID: udid}, cfg, nil

	case DestDevice:
		udid := cfg.Destination.UDID
		if udid == "" {
			return RunResult{}, cfg, errors.New("missing device udid")
		}
		emitMaybe(emit, Status("run", "Installing app on device", map[string]any{"udid": udid}))
		if err := DevicectlInstallApp(ctx, udid, appPath, emit); err != nil {
			return RunResult{}, cfg, err
		}
		emitMaybe(emit, Status("run", "Launching app on device", map[string]any{"bundleId": appInfo.BundleID, "console": console}))
		lr, err := DevicectlLaunchApp(ctx, udid, appInfo.BundleID, console, emit)
		if err != nil {
			return RunResult{}, cfg, err
		}
		_, _ = AddSession(projectRoot, appInfo.BundleID, lr.PID, "device", udid)
		emitMaybe(emit, Result("run", true, map[string]any{"pid": lr.PID, "bundleId": appInfo.BundleID}))
		return RunResult{ResultBundle: cfg.LastResultBundle, AppPath: appPath, BundleID: appInfo.BundleID, PID: lr.PID, Target: "device", UDID: udid}, cfg, nil

	default:
		return RunResult{}, cfg, fmt.Errorf("run not implemented for destination kind %q", cfg.Destination.Kind)
	}
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

func ResolveDestinationIfNeeded(ctx context.Context, projectRoot string, cfg Config, emit Emitter) (Config, error) {
	if cfg.Destination.Kind != DestAuto {
		return cfg, nil
	}

	list, err := SimctlList(ctx, emit)
	if err != nil {
		return cfg, err
	}
	sims := FlattenSimulators(list)
	// Prefer booted available iOS simulators; else newest iOS available.
	booted := []Simulator{}
	avail := []Simulator{}
	for _, s := range sims {
		if !s.Available {
			continue
		}
		if strings.Contains(strings.ToLower(s.RuntimeName), "ios") {
			avail = append(avail, s)
			if strings.ToLower(s.State) == "booted" {
				booted = append(booted, s)
			}
		}
	}
	pick := func(list []Simulator) (Simulator, bool) {
		if len(list) == 0 {
			return Simulator{}, false
		}
		// Sort by OSVersion desc (string compare is okay for most iOS versions if format is "17.2" etc),
		// then by name.
		sort.Slice(list, func(i, j int) bool {
			if list[i].OSVersion == list[j].OSVersion {
				return list[i].Name < list[j].Name
			}
			return list[i].OSVersion > list[j].OSVersion
		})
		return list[0], true
	}
	if s, ok := pick(booted); ok {
		cfg.Destination = Destination{Kind: DestSimulator, UDID: s.UDID, Name: s.Name, Platform: "iOS Simulator", OS: s.OSVersion}
		emitMaybe(emit, Status("context", "Auto-selected booted simulator", map[string]any{"name": s.Name, "udid": s.UDID, "os": s.OSVersion}))
		return cfg, nil
	}
	if s, ok := pick(avail); ok {
		cfg.Destination = Destination{Kind: DestSimulator, UDID: s.UDID, Name: s.Name, Platform: "iOS Simulator", OS: s.OSVersion}
		emitMaybe(emit, Status("context", "Auto-selected simulator", map[string]any{"name": s.Name, "udid": s.UDID, "os": s.OSVersion}))
		return cfg, nil
	}

	return cfg, errors.New("no available iOS simulators found")
}
