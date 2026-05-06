package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/odvcencio/gosx-native/emit/android"
	"github.com/odvcencio/gosx-native/emit/ios"
	"github.com/odvcencio/gosx-native/target"
	"github.com/odvcencio/gosx/nir"
)

type commandRunner interface {
	Run(ctx context.Context, dir, name string, args ...string) error
}

type osCommandRunner struct{}

func (osCommandRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

var buildRunner commandRunner = osCommandRunner{}

type stringList []string

func (l *stringList) String() string {
	return strings.Join(*l, ",")
}

func (l *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*l = append(*l, value)
	return nil
}

type buildOptions struct {
	source           string
	output           string
	project          string
	iosOutput        string
	iosProject       string
	androidOutput    string
	androidProject   string
	scheme           string
	simulator        string
	release          bool
	gradleTasks      stringList
	gradleProperties stringList
}

func runBuild(args []string) error {
	return runBuildWithContext(context.Background(), args)
}

func runBuildWithContext(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gsxnative build <ios|android|all> [flags]")
	}
	targetName := strings.ToLower(strings.TrimSpace(args[0]))
	opts, err := parseBuildOptions(targetName, args[1:])
	if err != nil {
		return err
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	switch targetName {
	case "android":
		return buildNativeTarget(ctx, root, target.Android, opts)
	case "ios":
		return buildNativeTarget(ctx, root, target.IOS, opts)
	case "all":
		if opts.project != "" || opts.output != "" {
			return fmt.Errorf("build all does not accept --project or --output; use --ios-project/--android-project and --ios-output/--android-output")
		}
		if err := buildNativeTarget(ctx, root, target.IOS, opts); err != nil {
			return err
		}
		return buildNativeTarget(ctx, root, target.Android, opts)
	default:
		return fmt.Errorf("unknown build target: %s (supported: ios, android, all)", args[0])
	}
}

func parseBuildOptions(targetName string, args []string) (buildOptions, error) {
	var opts buildOptions
	fs := flag.NewFlagSet("gsxnative build "+targetName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.source, "source", "", "GoSX source to compile before building")
	fs.StringVar(&opts.output, "output", "", "generated native source output for a single-target build")
	fs.StringVar(&opts.project, "project", "", "native project directory for a single-target build")
	fs.StringVar(&opts.iosOutput, "ios-output", "", "generated Swift output for build all")
	fs.StringVar(&opts.iosProject, "ios-project", "", "iOS project directory for build all")
	fs.StringVar(&opts.androidOutput, "android-output", "", "generated Kotlin output for build all")
	fs.StringVar(&opts.androidProject, "android-project", "", "Android project directory for build all")
	fs.StringVar(&opts.scheme, "scheme", "CounterDemo", "Xcode scheme for iOS builds")
	fs.StringVar(&opts.simulator, "simulator", defaultSimulatorName(), "iOS Simulator name")
	fs.BoolVar(&opts.release, "release", false, "build the release app variant when the target supports it")
	fs.Var(&opts.gradleTasks, "task", "Gradle task to run; repeatable")
	fs.Var(&opts.gradleProperties, "gradle-property", "Gradle project property without the -P prefix; repeatable")
	if err := fs.Parse(args); err != nil {
		return buildOptions{}, err
	}
	if fs.NArg() > 1 {
		return buildOptions{}, fmt.Errorf("usage: gsxnative build <ios|android|all> [flags] [project-dir]")
	}
	if fs.NArg() == 1 {
		if opts.project != "" {
			return buildOptions{}, fmt.Errorf("project specified both as --project and positional argument")
		}
		opts.project = fs.Arg(0)
	}
	return opts, nil
}

func buildNativeTarget(ctx context.Context, root string, tgt target.Target, opts buildOptions) error {
	cfg := nativeBuildConfig(root, tgt, opts)
	mod, err := compileFile(cfg.source)
	if err != nil {
		return err
	}
	if err := target.Validate(mod, tgt); err != nil {
		return err
	}
	source, err := emitNativeSource(tgt, mod)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.output, source, 0644); err != nil {
		return err
	}
	switch tgt {
	case target.Android:
		return buildAndroid(ctx, cfg)
	case target.IOS:
		return buildIOS(ctx, cfg)
	default:
		return fmt.Errorf("unknown target: %s", tgt)
	}
}

type nativeBuild struct {
	source           string
	output           string
	project          string
	scheme           string
	simulator        string
	release          bool
	gradleTasks      []string
	gradleProperties []string
}

func nativeBuildConfig(root string, tgt target.Target, opts buildOptions) nativeBuild {
	cfg := nativeBuild{
		source:           firstNonEmpty(opts.source, filepath.Join(root, "testdata/corpus/swift/counter.swift.gsx")),
		scheme:           opts.scheme,
		simulator:        opts.simulator,
		release:          opts.release,
		gradleTasks:      append([]string(nil), opts.gradleTasks...),
		gradleProperties: append([]string(nil), opts.gradleProperties...),
	}
	switch tgt {
	case target.Android:
		cfg.project = firstNonEmpty(opts.androidProject, opts.project, filepath.Join(root, "examples/counter-android"))
		cfg.output = firstNonEmpty(opts.androidOutput, opts.output, filepath.Join(cfg.project, "app/src/main/kotlin/generated/Counter.kt"))
	case target.IOS:
		cfg.project = firstNonEmpty(opts.iosProject, opts.project, filepath.Join(root, "examples/counter-ios"))
		cfg.output = firstNonEmpty(opts.iosOutput, opts.output, filepath.Join(cfg.project, "CounterDemo/Generated/Counter.swift"))
	}
	return cfg
}

func emitNativeSource(tgt target.Target, mod *nir.Module) ([]byte, error) {
	var buf bytes.Buffer
	switch tgt {
	case target.Android:
		if err := android.Emit(mod, &buf); err != nil {
			return nil, err
		}
	case target.IOS:
		if err := ios.Emit(mod, &buf); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown target: %s", tgt)
	}
	return buf.Bytes(), nil
}

func buildAndroid(ctx context.Context, cfg nativeBuild) error {
	tasks := cfg.gradleTasks
	if len(tasks) == 0 {
		tasks = []string{":gsx-nativekit:assembleRelease", ":app:assembleDebug"}
		if cfg.release {
			tasks = []string{":gsx-nativekit:assembleRelease", ":app:assembleRelease"}
		}
	}
	args := []string{"--no-daemon"}
	for _, property := range cfg.gradleProperties {
		args = append(args, "-P"+property)
	}
	args = append(args, tasks...)
	return buildRunner.Run(ctx, cfg.project, gradleExecutable(cfg.project), args...)
}

func buildIOS(ctx context.Context, cfg nativeBuild) error {
	if err := buildRunner.Run(ctx, cfg.project, "xcodegen", "generate"); err != nil {
		return err
	}
	derivedData, err := os.MkdirTemp("", "gsxnative-derived-data-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(derivedData)
	action := "build"
	args := []string{
		"-project", filepath.Join(cfg.project, "CounterDemo.xcodeproj"),
		"-scheme", cfg.scheme,
		"-destination", "platform=iOS Simulator,name=" + cfg.simulator,
		"-derivedDataPath", derivedData,
		action,
	}
	return buildRunner.Run(ctx, cfg.project, "xcodebuild", args...)
}

func gradleExecutable(project string) string {
	wrapper := filepath.Join(project, "gradlew")
	if _, err := os.Stat(wrapper); err == nil {
		return "./gradlew"
	}
	return "gradle"
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil && strings.Contains(string(data), "module github.com/odvcencio/gosx-native") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find gosx-native repository root from %s", dir)
		}
		dir = parent
	}
}

func defaultSimulatorName() string {
	if name := os.Getenv("IOS_SIMULATOR_NAME"); name != "" {
		return name
	}
	return "iPhone 16"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
