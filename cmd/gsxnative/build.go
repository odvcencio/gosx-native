package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	config           string
	iosOutput        string
	iosProject       string
	androidOutput    string
	androidProject   string
	xcodeProject     string
	scheme           string
	simulator        string
	schemeSet        bool
	simulatorSet     bool
	release          bool
	gradleTasks      stringList
	gradleProperties stringList
	projectConfig    *projectConfig
	projectBaseDir   string
}

type projectConfig struct {
	Name        string                `json:"name"`
	Module      string                `json:"module"`
	Source      string                `json:"source"`
	IOS         projectTargetConfig   `json:"ios"`
	Android     projectTargetConfig   `json:"android"`
	Routes      []routeDeclaration    `json:"routes,omitempty"`
	DataLoaders []endpointDeclaration `json:"data_loaders,omitempty"`
	Actions     []endpointDeclaration `json:"actions,omitempty"`
}

type projectTargetConfig struct {
	Project          string   `json:"project"`
	Output           string   `json:"output"`
	SupportOutput    string   `json:"support_output,omitempty"`
	XcodeProject     string   `json:"xcode_project,omitempty"`
	Scheme           string   `json:"scheme,omitempty"`
	Simulator        string   `json:"simulator,omitempty"`
	GradleTasks      []string `json:"gradle_tasks,omitempty"`
	GradleProperties []string `json:"gradle_properties,omitempty"`
}

type routeDeclaration struct {
	Name      string             `json:"name"`
	Path      string             `json:"path"`
	Component string             `json:"component"`
	Params    []paramDeclaration `json:"params,omitempty"`
}

type endpointDeclaration struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

type paramDeclaration struct {
	Name string `json:"name"`
	Type string `json:"type"`
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
	if err := applyBuildProjectConfig(&opts, targetName); err != nil {
		return err
	}
	root, err := repoRoot()
	if err != nil && !opts.hasNativeDefaults(targetName) {
		return err
	}
	if err != nil {
		root = ""
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
	opts := buildOptions{
		schemeSet:    flagWasProvided(args, "scheme"),
		simulatorSet: flagWasProvided(args, "simulator"),
	}
	fs := flag.NewFlagSet("gsxnative build "+targetName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.config, "config", "", "gosx-native project config")
	fs.StringVar(&opts.source, "source", "", "GoSX source to compile before building")
	fs.StringVar(&opts.output, "output", "", "generated native source output for a single-target build")
	fs.StringVar(&opts.project, "project", "", "native project directory for a single-target build")
	fs.StringVar(&opts.iosOutput, "ios-output", "", "generated Swift output for build all")
	fs.StringVar(&opts.iosProject, "ios-project", "", "iOS project directory for build all")
	fs.StringVar(&opts.androidOutput, "android-output", "", "generated Kotlin output for build all")
	fs.StringVar(&opts.androidProject, "android-project", "", "Android project directory for build all")
	fs.StringVar(&opts.xcodeProject, "xcode-project", "", "Xcode project name without .xcodeproj")
	fs.StringVar(&opts.scheme, "scheme", "", "Xcode scheme for iOS builds")
	fs.StringVar(&opts.simulator, "simulator", "", "iOS Simulator name")
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

func (opts buildOptions) hasNativeDefaults(targetName string) bool {
	switch targetName {
	case "android":
		return opts.source != "" && opts.androidProject != "" && opts.androidOutput != "" ||
			opts.source != "" && opts.project != "" && opts.output != ""
	case "ios":
		return opts.source != "" && opts.iosProject != "" && opts.iosOutput != "" ||
			opts.source != "" && opts.project != "" && opts.output != ""
	case "all":
		return opts.source != "" && opts.iosProject != "" && opts.iosOutput != "" &&
			opts.androidProject != "" && opts.androidOutput != ""
	default:
		return false
	}
}

func applyBuildProjectConfig(opts *buildOptions, targetName string) error {
	cfg, baseDir, ok, err := loadProjectConfig(opts.config)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	opts.projectConfig = cfg
	opts.projectBaseDir = baseDir
	if opts.source == "" && cfg.Source != "" {
		opts.source = resolveConfigPath(baseDir, cfg.Source)
	}
	if targetName == "ios" || targetName == "all" {
		applyIOSTargetConfig(opts, cfg.IOS, baseDir)
	}
	if targetName == "android" || targetName == "all" {
		applyAndroidTargetConfig(opts, cfg.Android, baseDir)
	}
	return nil
}

func applyIOSTargetConfig(opts *buildOptions, cfg projectTargetConfig, baseDir string) {
	if opts.iosProject == "" && opts.project == "" && cfg.Project != "" {
		opts.iosProject = resolveConfigPath(baseDir, cfg.Project)
	}
	if opts.iosOutput == "" && opts.output == "" && opts.project == "" && cfg.Output != "" {
		opts.iosOutput = resolveConfigPath(baseDir, cfg.Output)
	}
	if !opts.schemeSet && cfg.Scheme != "" {
		opts.scheme = cfg.Scheme
	}
	if opts.xcodeProject == "" && cfg.XcodeProject != "" {
		opts.xcodeProject = cfg.XcodeProject
	}
	if !opts.simulatorSet && cfg.Simulator != "" {
		opts.simulator = cfg.Simulator
	}
}

func applyAndroidTargetConfig(opts *buildOptions, cfg projectTargetConfig, baseDir string) {
	if opts.androidProject == "" && opts.project == "" && cfg.Project != "" {
		opts.androidProject = resolveConfigPath(baseDir, cfg.Project)
	}
	if opts.androidOutput == "" && opts.output == "" && opts.project == "" && cfg.Output != "" {
		opts.androidOutput = resolveConfigPath(baseDir, cfg.Output)
	}
	if len(opts.gradleTasks) == 0 && len(cfg.GradleTasks) > 0 {
		opts.gradleTasks = append(opts.gradleTasks, cfg.GradleTasks...)
	}
	if len(opts.gradleProperties) == 0 && len(cfg.GradleProperties) > 0 {
		opts.gradleProperties = append(opts.gradleProperties, cfg.GradleProperties...)
	}
}

func loadProjectConfig(path string) (*projectConfig, string, bool, error) {
	if path == "" {
		found, ok, err := findProjectConfig()
		if err != nil {
			return nil, "", false, err
		}
		if !ok {
			return nil, "", false, nil
		}
		path = found
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false, fmt.Errorf("read project config %s: %w", path, err)
	}
	var cfg projectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, "", false, fmt.Errorf("parse project config %s: %w", path, err)
	}
	return &cfg, filepath.Dir(path), true, nil
}

func findProjectConfig() (string, bool, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	for {
		path := filepath.Join(dir, "gosxnative.json")
		if _, err := os.Stat(path); err == nil {
			return path, true, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", false, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}

func resolveConfigPath(baseDir, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

func flagWasProvided(args []string, name string) bool {
	short := "-" + name
	long := "--" + name
	for _, arg := range args {
		if arg == short || arg == long || strings.HasPrefix(arg, short+"=") || strings.HasPrefix(arg, long+"=") {
			return true
		}
	}
	return false
}

func buildNativeTarget(ctx context.Context, root string, tgt target.Target, opts buildOptions) error {
	cfg, err := nativeBuildConfig(root, tgt, opts)
	if err != nil {
		return err
	}
	mod, err := compileFile(cfg.source)
	if err != nil {
		return err
	}
	if err := target.Validate(mod, tgt); err != nil {
		return err
	}
	if err := validateProjectDeclarations(opts.projectConfig, mod); err != nil {
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
	if cfg.supportOutput != "" && cfg.projectConfig != nil {
		support, err := emitDeclarationSupport(tgt, cfg.projectConfig)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(cfg.supportOutput), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(cfg.supportOutput, support, 0644); err != nil {
			return err
		}
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
	supportOutput    string
	project          string
	xcodeProject     string
	scheme           string
	simulator        string
	release          bool
	gradleTasks      []string
	gradleProperties []string
	projectConfig    *projectConfig
}

func nativeBuildConfig(root string, tgt target.Target, opts buildOptions) (nativeBuild, error) {
	cfg := nativeBuild{
		source:           firstNonEmpty(opts.source, repoDefault(root, "testdata/corpus/swift/counter.swift.gsx")),
		xcodeProject:     firstNonEmpty(opts.xcodeProject, "CounterDemo"),
		scheme:           firstNonEmpty(opts.scheme, "CounterDemo"),
		simulator:        firstNonEmpty(opts.simulator, defaultSimulatorName()),
		release:          opts.release,
		gradleTasks:      append([]string(nil), opts.gradleTasks...),
		gradleProperties: append([]string(nil), opts.gradleProperties...),
		projectConfig:    opts.projectConfig,
	}
	switch tgt {
	case target.Android:
		cfg.project = firstNonEmpty(opts.androidProject, opts.project, repoDefault(root, "examples/counter-android"))
		cfg.output = firstNonEmpty(opts.androidOutput, opts.output, filepath.Join(cfg.project, "app/src/main/kotlin/generated/Counter.kt"))
		if opts.projectConfig != nil {
			cfg.supportOutput = resolveConfigPath(opts.projectBaseDir, firstNonEmpty(opts.projectConfig.Android.SupportOutput, defaultSupportOutput(tgt, cfg.output)))
		}
	case target.IOS:
		cfg.project = firstNonEmpty(opts.iosProject, opts.project, repoDefault(root, "examples/counter-ios"))
		cfg.output = firstNonEmpty(opts.iosOutput, opts.output, filepath.Join(cfg.project, "CounterDemo/Generated/Counter.swift"))
		if opts.projectConfig != nil {
			cfg.supportOutput = resolveConfigPath(opts.projectBaseDir, firstNonEmpty(opts.projectConfig.IOS.SupportOutput, defaultSupportOutput(tgt, cfg.output)))
		}
	}
	if cfg.source == "" {
		return nativeBuild{}, fmt.Errorf("missing source; pass --source or run from a directory with gosxnative.json")
	}
	if cfg.project == "" {
		return nativeBuild{}, fmt.Errorf("missing native project for %s; pass --project or run from a directory with gosxnative.json", tgt)
	}
	if cfg.output == "" {
		return nativeBuild{}, fmt.Errorf("missing generated output for %s; pass --output or run from a directory with gosxnative.json", tgt)
	}
	return cfg, nil
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
		"-project", filepath.Join(cfg.project, cfg.xcodeProject+".xcodeproj"),
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

func repoDefault(root, path string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, path)
}
