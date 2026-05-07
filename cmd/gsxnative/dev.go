package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/odvcencio/gosx-native/target"
)

type devOptions struct {
	target          string
	once            bool
	build           bool
	install         bool
	launch          bool
	interval        time.Duration
	buildArgs       []string
	device          string
	androidDevice   string
	iosDevice       string
	androidPackage  string
	androidActivity string
	androidAPK      string
	iosBundleID     string
	iosAppPath      string
	adb             string
	xcrun           string
}

type watchedFile struct {
	size    int64
	modTime time.Time
}

func runDev(args []string) error {
	return runDevWithContext(context.Background(), args)
}

func runDevWithContext(ctx context.Context, args []string) error {
	opts, err := parseDevOptions(args)
	if err != nil {
		return err
	}
	if err := runDevCycle(ctx, opts); err != nil {
		return err
	}
	if opts.once {
		return nil
	}
	watchRoot, err := devWatchRoot(opts)
	if err != nil {
		return err
	}
	snapshot, err := snapshotDevFiles(watchRoot)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "watching %s for %s codegen\n", watchRoot, opts.target)
	ticker := time.NewTicker(opts.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			next, changed, err := changedDevFiles(watchRoot, snapshot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dev watch: %v\n", err)
				continue
			}
			if !changed {
				continue
			}
			if err := runDevCycle(ctx, opts); err != nil {
				fmt.Fprintf(os.Stderr, "dev build: %v\n", err)
			}
			snapshot = next
		}
	}
}

func parseDevOptions(args []string) (devOptions, error) {
	opts := devOptions{target: "all", interval: 500 * time.Millisecond, androidActivity: ".MainActivity", adb: "adb", xcrun: "xcrun"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if i == 0 && isDevTarget(arg) {
			opts.target = strings.ToLower(arg)
			continue
		}
		switch {
		case arg == "--once":
			opts.once = true
		case arg == "--build":
			opts.build = true
		case arg == "--install":
			opts.install = true
		case arg == "--launch":
			opts.launch = true
		case arg == "--target":
			i++
			if i >= len(args) {
				return devOptions{}, fmt.Errorf("flag needs an argument: --target")
			}
			if !isDevTarget(args[i]) {
				return devOptions{}, fmt.Errorf("unknown dev target: %s (supported: ios, android, all)", args[i])
			}
			opts.target = strings.ToLower(args[i])
		case strings.HasPrefix(arg, "--target="):
			value := strings.TrimPrefix(arg, "--target=")
			if !isDevTarget(value) {
				return devOptions{}, fmt.Errorf("unknown dev target: %s (supported: ios, android, all)", value)
			}
			opts.target = strings.ToLower(value)
		case arg == "--interval":
			i++
			if i >= len(args) {
				return devOptions{}, fmt.Errorf("flag needs an argument: --interval")
			}
			interval, err := time.ParseDuration(args[i])
			if err != nil || interval <= 0 {
				return devOptions{}, fmt.Errorf("invalid dev interval %q", args[i])
			}
			opts.interval = interval
		case strings.HasPrefix(arg, "--interval="):
			value := strings.TrimPrefix(arg, "--interval=")
			interval, err := time.ParseDuration(value)
			if err != nil || interval <= 0 {
				return devOptions{}, fmt.Errorf("invalid dev interval %q", value)
			}
			opts.interval = interval
		case arg == "--device":
			value, next, err := devFlagValue(args, i, "device")
			if err != nil {
				return devOptions{}, err
			}
			opts.device = value
			i = next
		case strings.HasPrefix(arg, "--device="):
			opts.device = strings.TrimPrefix(arg, "--device=")
		case arg == "--android-device":
			value, next, err := devFlagValue(args, i, "android-device")
			if err != nil {
				return devOptions{}, err
			}
			opts.androidDevice = value
			i = next
		case strings.HasPrefix(arg, "--android-device="):
			opts.androidDevice = strings.TrimPrefix(arg, "--android-device=")
		case arg == "--ios-device":
			value, next, err := devFlagValue(args, i, "ios-device")
			if err != nil {
				return devOptions{}, err
			}
			opts.iosDevice = value
			i = next
		case strings.HasPrefix(arg, "--ios-device="):
			opts.iosDevice = strings.TrimPrefix(arg, "--ios-device=")
		case arg == "--android-package":
			value, next, err := devFlagValue(args, i, "android-package")
			if err != nil {
				return devOptions{}, err
			}
			opts.androidPackage = value
			i = next
		case strings.HasPrefix(arg, "--android-package="):
			opts.androidPackage = strings.TrimPrefix(arg, "--android-package=")
		case arg == "--android-activity":
			value, next, err := devFlagValue(args, i, "android-activity")
			if err != nil {
				return devOptions{}, err
			}
			opts.androidActivity = value
			i = next
		case strings.HasPrefix(arg, "--android-activity="):
			opts.androidActivity = strings.TrimPrefix(arg, "--android-activity=")
		case arg == "--apk":
			value, next, err := devFlagValue(args, i, "apk")
			if err != nil {
				return devOptions{}, err
			}
			opts.androidAPK = value
			i = next
		case strings.HasPrefix(arg, "--apk="):
			opts.androidAPK = strings.TrimPrefix(arg, "--apk=")
		case arg == "--ios-bundle-id":
			value, next, err := devFlagValue(args, i, "ios-bundle-id")
			if err != nil {
				return devOptions{}, err
			}
			opts.iosBundleID = value
			i = next
		case strings.HasPrefix(arg, "--ios-bundle-id="):
			opts.iosBundleID = strings.TrimPrefix(arg, "--ios-bundle-id=")
		case arg == "--ios-app":
			value, next, err := devFlagValue(args, i, "ios-app")
			if err != nil {
				return devOptions{}, err
			}
			opts.iosAppPath = value
			i = next
		case strings.HasPrefix(arg, "--ios-app="):
			opts.iosAppPath = strings.TrimPrefix(arg, "--ios-app=")
		case arg == "--adb":
			value, next, err := devFlagValue(args, i, "adb")
			if err != nil {
				return devOptions{}, err
			}
			opts.adb = value
			i = next
		case strings.HasPrefix(arg, "--adb="):
			opts.adb = strings.TrimPrefix(arg, "--adb=")
		case arg == "--xcrun":
			value, next, err := devFlagValue(args, i, "xcrun")
			if err != nil {
				return devOptions{}, err
			}
			opts.xcrun = value
			i = next
		case strings.HasPrefix(arg, "--xcrun="):
			opts.xcrun = strings.TrimPrefix(arg, "--xcrun=")
		default:
			opts.buildArgs = append(opts.buildArgs, arg)
		}
	}
	if opts.launch {
		opts.install = true
	}
	if opts.install {
		opts.build = true
	}
	if opts.install && hasBuildFlag(opts.buildArgs, "codegen-only") {
		return devOptions{}, fmt.Errorf("dev install/launch cannot run with --codegen-only")
	}
	return opts, nil
}

func devFlagValue(args []string, index int, name string) (string, int, error) {
	next := index + 1
	if next >= len(args) {
		return "", index, fmt.Errorf("flag needs an argument: --%s", name)
	}
	return args[next], next, nil
}

func isDevTarget(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ios", "android", "all":
		return true
	default:
		return false
	}
}

func runDevCycle(ctx context.Context, opts devOptions) error {
	buildArgs, err := devBuildArgs(opts)
	if err != nil {
		return err
	}
	if err := runDevBuild(ctx, opts, buildArgs); err != nil {
		return err
	}
	return runDevDeviceActions(ctx, opts, buildArgs)
}

func runDevBuild(ctx context.Context, opts devOptions, buildArgs []string) error {
	args := append([]string{opts.target}, buildArgs...)
	if !opts.build && !hasBuildFlag(args, "codegen-only") {
		args = append(args, "--codegen-only")
	}
	return runBuildWithContext(ctx, args)
}

func devBuildArgs(opts devOptions) ([]string, error) {
	args := append([]string(nil), opts.buildArgs...)
	if !opts.install || !devTargetsIOS(opts.target) || opts.iosAppPath != "" || hasBuildFlag(args, "derived-data") {
		return args, nil
	}
	derivedData, err := defaultDevDerivedDataPath(args)
	if err != nil {
		return nil, err
	}
	return append(args, "--derived-data", derivedData), nil
}

func hasBuildFlag(args []string, name string) bool {
	short := "-" + name
	long := "--" + name
	for _, arg := range args {
		if arg == short || arg == long || strings.HasPrefix(arg, short+"=") || strings.HasPrefix(arg, long+"=") {
			return true
		}
	}
	return false
}

func runDevDeviceActions(ctx context.Context, opts devOptions, buildArgs []string) error {
	if !opts.install {
		return nil
	}
	for _, tgt := range devDeviceTargets(opts.target) {
		cfg, err := devNativeBuildConfig(tgt, buildArgs)
		if err != nil {
			return err
		}
		switch tgt {
		case target.Android:
			if err := runAndroidDeviceActions(ctx, opts, cfg); err != nil {
				return err
			}
		case target.IOS:
			if err := runIOSDeviceActions(ctx, opts, cfg); err != nil {
				return err
			}
		}
	}
	return nil
}

func runAndroidDeviceActions(ctx context.Context, opts devOptions, cfg nativeBuild) error {
	apk := opts.androidAPK
	if apk == "" {
		artifact, ok := firstExpectedArtifact(nativeBuildResultFor(target.Android, cfg), "android_apk")
		if !ok {
			return fmt.Errorf("could not determine Android APK for dev install; pass --apk or build an APK task")
		}
		apk = artifact.Path
	}
	packageName := ""
	if opts.launch {
		packageName = firstNonEmpty(opts.androidPackage, cfg.projectConfigAndroidPackage(), readAndroidApplicationID(cfg.project))
		if packageName == "" {
			return fmt.Errorf("could not determine Android package for dev launch; pass --android-package")
		}
	}
	adbArgs := androidADBPrefix(opts)
	if err := buildRunner.Run(ctx, cfg.project, firstNonEmpty(opts.adb, "adb"), append(adbArgs, "install", "-r", apk)...); err != nil {
		return err
	}
	if !opts.launch {
		return nil
	}
	activity := firstNonEmpty(opts.androidActivity, ".MainActivity")
	component := androidLaunchComponent(packageName, activity)
	return buildRunner.Run(ctx, cfg.project, firstNonEmpty(opts.adb, "adb"), append(adbArgs, "shell", "am", "start", "-n", component)...)
}

func runIOSDeviceActions(ctx context.Context, opts devOptions, cfg nativeBuild) error {
	appPath := opts.iosAppPath
	if appPath == "" {
		if cfg.derivedDataPath == "" {
			return fmt.Errorf("could not determine iOS app path for dev install; pass --ios-app or --derived-data")
		}
		appPath = iosSimulatorAppPath(cfg)
	}
	device := firstNonEmpty(opts.iosDevice, opts.device, "booted")
	xcrun := firstNonEmpty(opts.xcrun, "xcrun")
	bundleID := ""
	if opts.launch {
		bundleID = firstNonEmpty(opts.iosBundleID, cfg.projectConfigIOSBundleID(), readXcodeGenBundleID(cfg.project, cfg.scheme))
		if bundleID == "" {
			return fmt.Errorf("could not determine iOS bundle id for dev launch; pass --ios-bundle-id")
		}
	}
	if err := buildRunner.Run(ctx, cfg.project, xcrun, "simctl", "install", device, appPath); err != nil {
		return err
	}
	if !opts.launch {
		return nil
	}
	return buildRunner.Run(ctx, cfg.project, xcrun, "simctl", "launch", device, bundleID)
}

func devNativeBuildConfig(tgt target.Target, buildArgs []string) (nativeBuild, error) {
	targetName := string(tgt)
	opts, err := parseBuildOptions(targetName, buildArgs)
	if err != nil {
		return nativeBuild{}, err
	}
	if err := applyBuildProjectConfig(&opts, targetName); err != nil {
		return nativeBuild{}, err
	}
	if err := applyReleaseSigningConfig(&opts); err != nil {
		return nativeBuild{}, err
	}
	root, err := repoRoot()
	if err != nil && !opts.hasNativeDefaults(targetName) {
		return nativeBuild{}, err
	}
	if err != nil {
		root = ""
	}
	return nativeBuildConfig(root, tgt, opts)
}

func defaultDevDerivedDataPath(buildArgs []string) (string, error) {
	cfg, err := devNativeBuildConfig(target.IOS, buildArgs)
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg.project, "build", "gsxnative-derived-data"), nil
}

func devDeviceTargets(targetName string) []target.Target {
	switch strings.ToLower(targetName) {
	case "android":
		return []target.Target{target.Android}
	case "ios":
		return []target.Target{target.IOS}
	case "all":
		return []target.Target{target.IOS, target.Android}
	default:
		return nil
	}
}

func devTargetsIOS(targetName string) bool {
	for _, tgt := range devDeviceTargets(targetName) {
		if tgt == target.IOS {
			return true
		}
	}
	return false
}

func androidADBPrefix(opts devOptions) []string {
	device := firstNonEmpty(opts.androidDevice, opts.device)
	if device == "" {
		return nil
	}
	return []string{"-s", device}
}

func firstExpectedArtifact(result nativeBuildResult, kind string) (expectedArtifact, bool) {
	for _, artifact := range result.ExpectedArtifacts {
		if artifact.Kind == kind {
			return artifact, true
		}
	}
	return expectedArtifact{}, false
}

func (cfg nativeBuild) projectConfigAndroidPackage() string {
	if cfg.projectConfig == nil {
		return ""
	}
	return strings.TrimSpace(cfg.projectConfig.Module)
}

func (cfg nativeBuild) projectConfigIOSBundleID() string {
	if cfg.projectConfig == nil || strings.TrimSpace(cfg.projectConfig.Module) == "" || strings.TrimSpace(cfg.projectConfig.Name) == "" {
		return ""
	}
	return strings.TrimSpace(cfg.projectConfig.Module) + "." + strings.TrimSpace(cfg.projectConfig.Name)
}

func readAndroidApplicationID(project string) string {
	data, err := os.ReadFile(filepath.Join(project, "app", "build.gradle.kts"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "applicationId") {
			continue
		}
		if _, value, ok := strings.Cut(line, "="); ok {
			return strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return ""
}

func readXcodeGenBundleID(project, scheme string) string {
	data, err := os.ReadFile(filepath.Join(project, "project.yml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "bundleIdPrefix:") {
			continue
		}
		prefix := strings.TrimSpace(strings.TrimPrefix(line, "bundleIdPrefix:"))
		prefix = strings.Trim(prefix, `"`)
		if prefix == "" || scheme == "" {
			return prefix
		}
		return prefix + "." + scheme
	}
	return ""
}

func androidLaunchComponent(packageName, activity string) string {
	activity = strings.TrimSpace(activity)
	if strings.Contains(activity, "/") {
		return activity
	}
	return packageName + "/" + activity
}

func devWatchRoot(opts devOptions) (string, error) {
	buildOpts, err := parseBuildOptions(opts.target, opts.buildArgs)
	if err != nil {
		return "", err
	}
	if err := applyBuildProjectConfig(&buildOpts, opts.target); err != nil {
		return "", err
	}
	if buildOpts.projectBaseDir != "" {
		return buildOpts.projectBaseDir, nil
	}
	if buildOpts.source == "" {
		root, err := repoRoot()
		if err != nil {
			return "", fmt.Errorf("missing source; pass --source or run from a directory with gosxnative.json")
		}
		buildOpts.source = repoDefault(root, "testdata/corpus/swift/counter.swift.gsx")
	}
	info, err := os.Stat(buildOpts.source)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return buildOpts.source, nil
	}
	return filepath.Dir(buildOpts.source), nil
}

func changedDevFiles(root string, previous map[string]watchedFile) (map[string]watchedFile, bool, error) {
	next, err := snapshotDevFiles(root)
	if err != nil {
		return nil, false, err
	}
	if len(previous) != len(next) {
		return next, true, nil
	}
	for path, state := range next {
		old, ok := previous[path]
		if !ok || old.size != state.size || !old.modTime.Equal(state.modTime) {
			return next, true, nil
		}
	}
	return next, false, nil
}

func snapshotDevFiles(root string) (map[string]watchedFile, error) {
	files := make(map[string]watchedFile)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".gradle", "DerivedData", "build":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !isDevSourceFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files[path] = watchedFile{size: info.Size(), modTime: info.ModTime()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func isDevSourceFile(path string) bool {
	name := filepath.Base(path)
	if name == "gosxnative.json" || name == "capabilities.json" {
		return true
	}
	return strings.HasSuffix(name, ".gsx") ||
		strings.HasSuffix(name, ".swift.gsx") ||
		strings.HasSuffix(name, ".kt.gsx")
}
