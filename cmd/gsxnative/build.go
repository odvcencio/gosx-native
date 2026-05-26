package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"m31labs.dev/gosx-native/emit/android"
	"m31labs.dev/gosx-native/emit/ios"
	"m31labs.dev/gosx-native/target"
	"m31labs.dev/gosx/nir"
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
	source             string
	output             string
	project            string
	config             string
	iosOutput          string
	iosProject         string
	androidOutput      string
	androidProject     string
	xcodeProject       string
	scheme             string
	simulator          string
	configuration      string
	derivedDataPath    string
	archivePath        string
	exportOptions      string
	exportPath         string
	artifactManifest   string
	artifactPublishDir string
	signingConfig      string
	serverURL          string
	environment        string
	androidFlavor      string
	schemeSet          bool
	simulatorSet       bool
	configurationSet   bool
	release            bool
	codegenOnly        bool
	gradleTasks        stringList
	gradleProperties   stringList
	projectConfig      *projectConfig
	projectBaseDir     string
	signing            *releaseSigningConfig
	signingConfigPath  string
}

type projectConfig struct {
	Name               string                  `json:"name"`
	Module             string                  `json:"module"`
	Source             string                  `json:"source"`
	IOS                projectTargetConfig     `json:"ios"`
	Android            projectTargetConfig     `json:"android"`
	Environment        string                  `json:"environment,omitempty"`
	ArtifactManifest   string                  `json:"artifact_manifest,omitempty"`
	ArtifactPublishDir string                  `json:"artifact_publish_dir,omitempty"`
	SigningConfig      string                  `json:"signing_config,omitempty"`
	ServerURL          string                  `json:"server_url,omitempty"`
	Routes             []routeDeclaration      `json:"routes,omitempty"`
	DataLoaders        []endpointDeclaration   `json:"data_loaders,omitempty"`
	Actions            []endpointDeclaration   `json:"actions,omitempty"`
	Models             []modelDeclaration      `json:"models,omitempty"`
	Capabilities       []capabilityDeclaration `json:"capabilities,omitempty"`
	Bridges            []bridgeDeclaration     `json:"bridges,omitempty"`
}

type projectTargetConfig struct {
	Project          string   `json:"project"`
	Output           string   `json:"output"`
	SupportOutput    string   `json:"support_output,omitempty"`
	XcodeProject     string   `json:"xcode_project,omitempty"`
	Scheme           string   `json:"scheme,omitempty"`
	Simulator        string   `json:"simulator,omitempty"`
	Configuration    string   `json:"configuration,omitempty"`
	DerivedDataPath  string   `json:"derived_data_path,omitempty"`
	ArchivePath      string   `json:"archive_path,omitempty"`
	ExportOptions    string   `json:"export_options_plist,omitempty"`
	ExportPath       string   `json:"export_path,omitempty"`
	Flavor           string   `json:"flavor,omitempty"`
	GradleTasks      []string `json:"gradle_tasks,omitempty"`
	GradleProperties []string `json:"gradle_properties,omitempty"`
}

type releaseSigningConfig struct {
	IOS     iosSigningConfig     `json:"ios,omitempty"`
	Android androidSigningConfig `json:"android,omitempty"`
}

type iosSigningConfig struct {
	TeamID                   string `json:"team_id,omitempty"`
	BundleID                 string `json:"bundle_id,omitempty"`
	CodeSignStyle            string `json:"code_sign_style,omitempty"`
	ProvisioningProfile      string `json:"provisioning_profile,omitempty"`
	CodeSignIdentity         string `json:"code_sign_identity,omitempty"`
	ExportOptions            string `json:"export_options_plist,omitempty"`
	AllowProvisioningUpdates bool   `json:"allow_provisioning_updates,omitempty"`
}

type androidSigningConfig struct {
	StoreFile        string `json:"store_file,omitempty"`
	StorePasswordEnv string `json:"store_password_env,omitempty"`
	KeyAlias         string `json:"key_alias,omitempty"`
	KeyPasswordEnv   string `json:"key_password_env,omitempty"`
}

type routeDeclaration struct {
	Name      string             `json:"name"`
	Path      string             `json:"path"`
	Component string             `json:"component"`
	Params    []paramDeclaration `json:"params,omitempty"`
	Auth      string             `json:"auth,omitempty"`
}

type endpointDeclaration struct {
	Name            string             `json:"name"`
	Method          string             `json:"method"`
	Path            string             `json:"path"`
	Params          []paramDeclaration `json:"params,omitempty"`
	Input           []paramDeclaration `json:"input,omitempty"`
	Output          []paramDeclaration `json:"output,omitempty"`
	CacheTTLSeconds int                `json:"cache_ttl_seconds,omitempty"`
	Invalidates     []string           `json:"invalidates,omitempty"`
	Optimistic      string             `json:"optimistic,omitempty"`
	Auth            string             `json:"auth,omitempty"`
	RetryAttempts   int                `json:"retry_attempts,omitempty"`
	RetryBaseMillis int                `json:"retry_base_delay_millis,omitempty"`
	RetryMaxMillis  int                `json:"retry_max_delay_millis,omitempty"`
	NetworkPolicy   string             `json:"network_policy,omitempty"`
}

type capabilityDeclaration struct {
	Name     string   `json:"name"`
	Targets  []string `json:"targets,omitempty"`
	Required bool     `json:"required,omitempty"`
}

type bridgeDeclaration struct {
	Service       string             `json:"service"`
	Method        string             `json:"method"`
	Path          string             `json:"path"`
	Input         []paramDeclaration `json:"input,omitempty"`
	Output        []paramDeclaration `json:"output,omitempty"`
	Auth          string             `json:"auth,omitempty"`
	RetryAttempts int                `json:"retry_attempts,omitempty"`
}

type modelDeclaration struct {
	Name   string             `json:"name"`
	Fields []paramDeclaration `json:"fields,omitempty"`
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
	if err := validateBuildVariantOptions(opts); err != nil {
		return err
	}
	if err := applyReleaseSigningConfig(&opts); err != nil {
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
		result, err := buildNativeTarget(ctx, root, target.Android, opts)
		if err != nil {
			return err
		}
		results, err := publishBuildArtifacts(opts, []nativeBuildResult{result})
		if err != nil {
			return err
		}
		return writeBuildArtifactManifest(opts, results)
	case "ios":
		result, err := buildNativeTarget(ctx, root, target.IOS, opts)
		if err != nil {
			return err
		}
		results, err := publishBuildArtifacts(opts, []nativeBuildResult{result})
		if err != nil {
			return err
		}
		return writeBuildArtifactManifest(opts, results)
	case "all":
		if opts.project != "" || opts.output != "" {
			return fmt.Errorf("build all does not accept --project or --output; use --ios-project/--android-project and --ios-output/--android-output")
		}
		iosResult, err := buildNativeTarget(ctx, root, target.IOS, opts)
		if err != nil {
			return err
		}
		androidResult, err := buildNativeTarget(ctx, root, target.Android, opts)
		if err != nil {
			return err
		}
		results, err := publishBuildArtifacts(opts, []nativeBuildResult{iosResult, androidResult})
		if err != nil {
			return err
		}
		return writeBuildArtifactManifest(opts, results)
	default:
		return fmt.Errorf("unknown build target: %s (supported: ios, android, all)", args[0])
	}
}

func parseBuildOptions(targetName string, args []string) (buildOptions, error) {
	opts := buildOptions{
		schemeSet:        flagWasProvided(args, "scheme"),
		simulatorSet:     flagWasProvided(args, "simulator"),
		configurationSet: flagWasProvided(args, "configuration"),
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
	fs.StringVar(&opts.configuration, "configuration", "", "Xcode configuration for iOS builds")
	fs.StringVar(&opts.derivedDataPath, "derived-data", "", "stable Xcode derived data path for debug builds and install/launch flows")
	fs.StringVar(&opts.archivePath, "archive-path", "", "iOS archive path for release builds")
	fs.StringVar(&opts.exportOptions, "export-options-plist", "", "ExportOptions.plist for iOS release export")
	fs.StringVar(&opts.exportPath, "export-path", "", "iOS export directory for release builds")
	fs.StringVar(&opts.artifactManifest, "artifact-manifest", "", "write a JSON manifest describing generated and native build artifacts")
	fs.StringVar(&opts.artifactPublishDir, "publish-dir", "", "copy expected native build artifacts into this directory after a successful build")
	fs.StringVar(&opts.signingConfig, "signing-config", "", "release signing config JSON; defaults to .gsxnative/signing.json when present")
	fs.StringVar(&opts.serverURL, "server-url", "", "dev/server base URL forwarded to native build metadata")
	fs.StringVar(&opts.environment, "env", "", "build environment name forwarded to native build tools")
	fs.StringVar(&opts.androidFlavor, "flavor", "", "Android product flavor for default app build tasks")
	fs.StringVar(&opts.androidFlavor, "android-flavor", "", "Android product flavor for default app build tasks")
	fs.BoolVar(&opts.release, "release", false, "build the release app variant when the target supports it")
	fs.BoolVar(&opts.codegenOnly, "codegen-only", false, "regenerate native sources without invoking Xcode or Gradle")
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
	if err := validateBuildVariantOptions(opts); err != nil {
		return buildOptions{}, err
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
	if opts.environment == "" && cfg.Environment != "" {
		opts.environment = cfg.Environment
	}
	if opts.artifactManifest == "" && cfg.ArtifactManifest != "" {
		opts.artifactManifest = resolveConfigPath(baseDir, cfg.ArtifactManifest)
	}
	if opts.artifactPublishDir == "" && cfg.ArtifactPublishDir != "" {
		opts.artifactPublishDir = resolveConfigPath(baseDir, cfg.ArtifactPublishDir)
	}
	if opts.signingConfig == "" && cfg.SigningConfig != "" {
		opts.signingConfig = resolveConfigPath(baseDir, cfg.SigningConfig)
	}
	if opts.serverURL == "" && cfg.ServerURL != "" {
		opts.serverURL = cfg.ServerURL
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
	if !opts.configurationSet && cfg.Configuration != "" {
		opts.configuration = cfg.Configuration
	}
	if opts.derivedDataPath == "" && cfg.DerivedDataPath != "" {
		opts.derivedDataPath = resolveConfigPath(baseDir, cfg.DerivedDataPath)
	}
	if opts.archivePath == "" && cfg.ArchivePath != "" {
		opts.archivePath = resolveConfigPath(baseDir, cfg.ArchivePath)
	}
	if opts.exportOptions == "" && cfg.ExportOptions != "" {
		opts.exportOptions = resolveConfigPath(baseDir, cfg.ExportOptions)
	}
	if opts.exportPath == "" && cfg.ExportPath != "" {
		opts.exportPath = resolveConfigPath(baseDir, cfg.ExportPath)
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
	if opts.androidFlavor == "" && cfg.Flavor != "" {
		opts.androidFlavor = cfg.Flavor
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

func applyReleaseSigningConfig(opts *buildOptions) error {
	if !opts.release {
		return nil
	}
	path := opts.signingConfig
	if path == "" {
		if found, ok, err := findDefaultSigningConfig(opts.projectBaseDir); err != nil {
			return err
		} else if ok {
			path = found
		}
	}
	if path == "" {
		return nil
	}
	signing, err := loadReleaseSigningConfig(path)
	if err != nil {
		return err
	}
	if err := validateReleaseSigningConfig(path, signing); err != nil {
		return err
	}
	opts.signing = signing
	opts.signingConfigPath = path
	if opts.exportOptions == "" && signing.IOS.ExportOptions != "" {
		opts.exportOptions = resolveConfigPath(filepath.Dir(path), signing.IOS.ExportOptions)
	}
	return nil
}

func loadReleaseSigningConfig(path string) (*releaseSigningConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read signing config %s: %w", path, err)
	}
	var cfg releaseSigningConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse signing config %s: %w", path, err)
	}
	return &cfg, nil
}

func validateReleaseSigningConfig(path string, cfg *releaseSigningConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Android.hasAny() {
		if strings.TrimSpace(cfg.Android.StoreFile) == "" {
			return fmt.Errorf("signing config %s android.store_file is required", path)
		}
		if strings.TrimSpace(cfg.Android.StorePasswordEnv) == "" {
			return fmt.Errorf("signing config %s android.store_password_env is required", path)
		}
		if strings.TrimSpace(cfg.Android.KeyAlias) == "" {
			return fmt.Errorf("signing config %s android.key_alias is required", path)
		}
	}
	return nil
}

func findDefaultSigningConfig(projectBaseDir string) (string, bool, error) {
	var starts []string
	if projectBaseDir != "" {
		starts = append(starts, projectBaseDir)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	starts = append(starts, wd)
	for _, start := range starts {
		for dir := start; ; dir = filepath.Dir(dir) {
			path := filepath.Join(dir, ".gsxnative", "signing.json")
			if _, err := os.Stat(path); err == nil {
				return path, true, nil
			} else if err != nil && !os.IsNotExist(err) {
				return "", false, err
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return "", false, nil
}

func (cfg androidSigningConfig) hasAny() bool {
	return strings.TrimSpace(cfg.StoreFile) != "" ||
		strings.TrimSpace(cfg.StorePasswordEnv) != "" ||
		strings.TrimSpace(cfg.KeyAlias) != "" ||
		strings.TrimSpace(cfg.KeyPasswordEnv) != ""
}

func (cfg iosSigningConfig) hasAny() bool {
	return strings.TrimSpace(cfg.TeamID) != "" ||
		strings.TrimSpace(cfg.BundleID) != "" ||
		strings.TrimSpace(cfg.CodeSignStyle) != "" ||
		strings.TrimSpace(cfg.ProvisioningProfile) != "" ||
		strings.TrimSpace(cfg.CodeSignIdentity) != "" ||
		strings.TrimSpace(cfg.ExportOptions) != "" ||
		cfg.AllowProvisioningUpdates
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

func validateBuildVariantOptions(opts buildOptions) error {
	if opts.environment != "" && strings.ContainsAny(opts.environment, " \t\r\n") {
		return fmt.Errorf("build environment %q must not contain whitespace", opts.environment)
	}
	if opts.androidFlavor != "" && !identifierPattern.MatchString(opts.androidFlavor) {
		return fmt.Errorf("android flavor %q has invalid name", opts.androidFlavor)
	}
	if opts.serverURL != "" {
		if err := validateServerURL(opts.serverURL); err != nil {
			return err
		}
	}
	return nil
}

func validateServerURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("server URL %q must be an absolute http(s) URL", raw)
	}
	switch parsed.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("server URL %q must use http or https", raw)
	}
}

func buildNativeTarget(ctx context.Context, root string, tgt target.Target, opts buildOptions) (nativeBuildResult, error) {
	cfg, err := nativeBuildConfig(root, tgt, opts)
	if err != nil {
		return nativeBuildResult{}, err
	}
	mod, err := compileFile(cfg.source)
	if err != nil {
		return nativeBuildResult{}, err
	}
	if err := target.Validate(mod, tgt); err != nil {
		return nativeBuildResult{}, err
	}
	if err := validateNativeImplementationsFile(cfg.source, tgt); err != nil {
		return nativeBuildResult{}, err
	}
	projectConfig, err := effectiveProjectConfigForSource(opts.projectConfig, cfg.source)
	if err != nil {
		return nativeBuildResult{}, err
	}
	if err := validateProjectDeclarations(projectConfig, mod); err != nil {
		return nativeBuildResult{}, err
	}
	source, err := emitNativeSource(tgt, mod)
	if err != nil {
		return nativeBuildResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.output), 0755); err != nil {
		return nativeBuildResult{}, err
	}
	if err := os.WriteFile(cfg.output, source, 0644); err != nil {
		return nativeBuildResult{}, err
	}
	if cfg.supportOutput != "" && projectConfig != nil {
		support, err := emitDeclarationSupport(tgt, projectConfig)
		if err != nil {
			return nativeBuildResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(cfg.supportOutput), 0755); err != nil {
			return nativeBuildResult{}, err
		}
		if err := os.WriteFile(cfg.supportOutput, support, 0644); err != nil {
			return nativeBuildResult{}, err
		}
	}
	result := nativeBuildResultFor(tgt, cfg)
	if cfg.codegenOnly {
		return result, nil
	}
	switch tgt {
	case target.Android:
		return result, buildAndroid(ctx, cfg)
	case target.IOS:
		return result, buildIOS(ctx, cfg)
	default:
		return nativeBuildResult{}, fmt.Errorf("unknown target: %s", tgt)
	}
}

type nativeBuild struct {
	source            string
	output            string
	supportOutput     string
	project           string
	xcodeProject      string
	scheme            string
	simulator         string
	configuration     string
	derivedDataPath   string
	archivePath       string
	exportOptions     string
	exportPath        string
	release           bool
	codegenOnly       bool
	gradleTasks       []string
	gradleProperties  []string
	environment       string
	serverURL         string
	androidFlavor     string
	projectConfig     *projectConfig
	signing           *releaseSigningConfig
	signingConfigPath string
}

type buildArtifactManifest struct {
	Version     int                 `json:"version"`
	Name        string              `json:"name,omitempty"`
	Module      string              `json:"module,omitempty"`
	Environment string              `json:"environment,omitempty"`
	Targets     []nativeBuildResult `json:"targets"`
}

type nativeBuildResult struct {
	Target             string              `json:"target"`
	Source             string              `json:"source"`
	Project            string              `json:"project"`
	GeneratedOutput    string              `json:"generated_output"`
	SupportOutput      string              `json:"support_output,omitempty"`
	Release            bool                `json:"release"`
	Environment        string              `json:"environment,omitempty"`
	ServerURL          string              `json:"server_url,omitempty"`
	Flavor             string              `json:"flavor,omitempty"`
	CodegenOnly        bool                `json:"codegen_only,omitempty"`
	BuildSystem        string              `json:"build_system,omitempty"`
	BuildTasks         []string            `json:"build_tasks,omitempty"`
	BuildProperties    []string            `json:"build_properties,omitempty"`
	Configuration      string              `json:"configuration,omitempty"`
	DerivedDataPath    string              `json:"derived_data_path,omitempty"`
	ArchivePath        string              `json:"archive_path,omitempty"`
	ExportPath         string              `json:"export_path,omitempty"`
	ExpectedArtifacts  []expectedArtifact  `json:"expected_artifacts,omitempty"`
	PublishedArtifacts []publishedArtifact `json:"published_artifacts,omitempty"`
	Signing            *signingSummary     `json:"signing,omitempty"`
}

type signingSummary struct {
	Config                   string `json:"config,omitempty"`
	TeamID                   string `json:"team_id,omitempty"`
	BundleID                 string `json:"bundle_id,omitempty"`
	CodeSignStyle            string `json:"code_sign_style,omitempty"`
	ProvisioningProfile      string `json:"provisioning_profile,omitempty"`
	CodeSignIdentity         string `json:"code_sign_identity,omitempty"`
	AllowProvisioningUpdates bool   `json:"allow_provisioning_updates,omitempty"`
	ExportOptions            string `json:"export_options_plist,omitempty"`
	StoreFile                string `json:"store_file,omitempty"`
	StorePasswordEnv         string `json:"store_password_env,omitempty"`
	KeyAlias                 string `json:"key_alias,omitempty"`
	KeyPasswordEnv           string `json:"key_password_env,omitempty"`
}

type expectedArtifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type publishedArtifact struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Path   string `json:"path"`
}

func nativeBuildConfig(root string, tgt target.Target, opts buildOptions) (nativeBuild, error) {
	cfg := nativeBuild{
		source:            firstNonEmpty(opts.source, repoDefault(root, "testdata/corpus/swift/counter.swift.gsx")),
		xcodeProject:      firstNonEmpty(opts.xcodeProject, "CounterDemo"),
		scheme:            firstNonEmpty(opts.scheme, "CounterDemo"),
		simulator:         firstNonEmpty(opts.simulator, defaultIOSDestination(opts.release)),
		configuration:     firstNonEmpty(opts.configuration, defaultIOSConfiguration(opts.release)),
		derivedDataPath:   opts.derivedDataPath,
		archivePath:       opts.archivePath,
		exportOptions:     opts.exportOptions,
		exportPath:        opts.exportPath,
		release:           opts.release,
		codegenOnly:       opts.codegenOnly,
		gradleTasks:       append([]string(nil), opts.gradleTasks...),
		gradleProperties:  append([]string(nil), opts.gradleProperties...),
		environment:       opts.environment,
		serverURL:         opts.serverURL,
		projectConfig:     opts.projectConfig,
		signing:           opts.signing,
		signingConfigPath: opts.signingConfigPath,
	}
	switch tgt {
	case target.Android:
		cfg.project = firstNonEmpty(opts.androidProject, opts.project, repoDefault(root, "examples/counter-android"))
		cfg.output = firstNonEmpty(opts.androidOutput, opts.output, filepath.Join(cfg.project, "app/src/main/kotlin/generated/Counter.kt"))
		cfg.androidFlavor = opts.androidFlavor
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

func nativeBuildResultFor(tgt target.Target, cfg nativeBuild) nativeBuildResult {
	result := nativeBuildResult{
		Target:          string(tgt),
		Source:          cfg.source,
		Project:         cfg.project,
		GeneratedOutput: cfg.output,
		SupportOutput:   cfg.supportOutput,
		Release:         cfg.release,
		Environment:     cfg.environment,
		ServerURL:       cfg.serverURL,
		Flavor:          cfg.androidFlavor,
		CodegenOnly:     cfg.codegenOnly,
	}
	switch tgt {
	case target.Android:
		tasks := androidBuildTasks(cfg)
		result.BuildSystem = gradleExecutable(cfg.project)
		result.BuildTasks = tasks
		result.BuildProperties = androidBuildPropertiesForManifest(cfg)
		result.Signing = androidSigningSummary(cfg)
		if !cfg.codegenOnly {
			result.ExpectedArtifacts = androidExpectedArtifacts(cfg.project, cfg.androidFlavor, tasks)
		}
	case target.IOS:
		result.BuildSystem = "xcodebuild"
		result.Configuration = cfg.configuration
		result.DerivedDataPath = cfg.derivedDataPath
		result.Signing = iosSigningSummary(cfg)
		if cfg.release {
			result.ArchivePath = iosArchivePath(cfg)
			if cfg.exportOptions != "" {
				result.ExportPath = iosExportPath(cfg)
			}
		}
		if !cfg.codegenOnly {
			result.ExpectedArtifacts = iosExpectedArtifacts(cfg)
		}
	}
	return result
}

func writeBuildArtifactManifest(opts buildOptions, results []nativeBuildResult) error {
	if opts.artifactManifest == "" {
		return nil
	}
	manifest := buildArtifactManifest{
		Version:     1,
		Environment: opts.environment,
		Targets:     results,
	}
	if opts.projectConfig != nil {
		manifest.Name = opts.projectConfig.Name
		manifest.Module = opts.projectConfig.Module
	}
	if err := os.MkdirAll(filepath.Dir(opts.artifactManifest), 0755); err != nil {
		return err
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return err
	}
	return os.WriteFile(opts.artifactManifest, buf.Bytes(), 0644)
}

func publishBuildArtifacts(opts buildOptions, results []nativeBuildResult) ([]nativeBuildResult, error) {
	if opts.artifactPublishDir == "" {
		return results, nil
	}
	published := append([]nativeBuildResult(nil), results...)
	for i := range published {
		targetDir := filepath.Join(opts.artifactPublishDir, published[i].Target)
		for _, artifact := range published[i].ExpectedArtifacts {
			destination := filepath.Join(targetDir, filepath.Base(artifact.Path))
			if err := copyBuildArtifact(artifact.Path, destination); err != nil {
				return nil, fmt.Errorf("publish %s artifact %s: %w", artifact.Kind, artifact.Path, err)
			}
			published[i].PublishedArtifacts = append(published[i].PublishedArtifacts, publishedArtifact{
				Kind:   artifact.Kind,
				Source: artifact.Path,
				Path:   destination,
			})
		}
	}
	return published, nil
}

func copyBuildArtifact(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyBuildArtifactDir(source, destination)
	}
	return copyBuildArtifactFile(source, destination, info.Mode())
}

func copyBuildArtifactDir(source, destination string) error {
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destination, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			return os.Symlink(link, targetPath)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyBuildArtifactFile(path, targetPath, info.Mode())
	})
}

func copyBuildArtifactFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func buildAndroid(ctx context.Context, cfg nativeBuild) error {
	tasks := androidBuildTasks(cfg)
	args, err := androidBuildArgs(cfg, tasks)
	if err != nil {
		return err
	}
	return buildRunner.Run(ctx, cfg.project, gradleExecutable(cfg.project), args...)
}

func androidBuildTasks(cfg nativeBuild) []string {
	tasks := cfg.gradleTasks
	if len(tasks) == 0 {
		tasks = []string{":gsx-nativekit:assembleRelease", ":app:assembleDebug"}
		if cfg.release {
			tasks = []string{":gsx-nativekit:assembleRelease", ":app:assembleRelease"}
		}
		if cfg.androidFlavor != "" {
			tasks[1] = ":app:" + androidAssembleTask(cfg.androidFlavor, cfg.release)
		}
	}
	return append([]string(nil), tasks...)
}

func androidBuildArgs(cfg nativeBuild, tasks []string) ([]string, error) {
	args := []string{"--no-daemon"}
	properties, err := androidBuildPropertiesForBuild(cfg)
	if err != nil {
		return nil, err
	}
	for _, property := range properties {
		args = append(args, "-P"+property)
	}
	args = append(args, tasks...)
	return args, nil
}

func androidBuildPropertiesForManifest(cfg nativeBuild) []string {
	properties := append([]string(nil), cfg.gradleProperties...)
	if cfg.environment != "" && !hasGradleProperty(properties, "gsxEnvironment") {
		properties = append(properties, "gsxEnvironment="+cfg.environment)
	}
	if cfg.serverURL != "" && !hasGradleProperty(properties, "gsxServerURL") {
		properties = append(properties, "gsxServerURL="+cfg.serverURL)
	}
	properties = appendAndroidSigningProperties(cfg, properties)
	return redactGradleProperties(properties)
}

func androidBuildPropertiesForBuild(cfg nativeBuild) ([]string, error) {
	properties := append([]string(nil), cfg.gradleProperties...)
	if cfg.environment != "" && !hasGradleProperty(properties, "gsxEnvironment") {
		properties = append(properties, "gsxEnvironment="+cfg.environment)
	}
	if cfg.serverURL != "" && !hasGradleProperty(properties, "gsxServerURL") {
		properties = append(properties, "gsxServerURL="+cfg.serverURL)
	}
	var err error
	properties, err = appendAndroidSigningPropertiesForBuild(cfg, properties)
	if err != nil {
		return nil, err
	}
	return properties, nil
}

func appendAndroidSigningPropertiesForBuild(cfg nativeBuild, properties []string) ([]string, error) {
	if !cfg.release || cfg.signing == nil || !cfg.signing.Android.hasAny() {
		return properties, nil
	}
	signing := cfg.signing.Android
	if !hasGradleProperty(properties, "gsxSigningStoreFile") {
		properties = append(properties, "gsxSigningStoreFile="+androidSigningStoreFile(cfg))
	}
	if !hasGradleProperty(properties, "gsxSigningStorePassword") {
		value, err := signingEnvValue(signing.StorePasswordEnv, "android.store_password_env")
		if err != nil {
			return nil, err
		}
		properties = append(properties, "gsxSigningStorePassword="+value)
	}
	if !hasGradleProperty(properties, "gsxSigningKeyAlias") {
		properties = append(properties, "gsxSigningKeyAlias="+signing.KeyAlias)
	}
	if !hasGradleProperty(properties, "gsxSigningKeyPassword") {
		keyPasswordEnv := firstNonEmpty(signing.KeyPasswordEnv, signing.StorePasswordEnv)
		value, err := signingEnvValue(keyPasswordEnv, "android.key_password_env")
		if err != nil {
			return nil, err
		}
		properties = append(properties, "gsxSigningKeyPassword="+value)
	}
	return properties, nil
}

func appendAndroidSigningProperties(cfg nativeBuild, properties []string) []string {
	if !cfg.release || cfg.signing == nil || !cfg.signing.Android.hasAny() {
		return properties
	}
	signing := cfg.signing.Android
	if !hasGradleProperty(properties, "gsxSigningStoreFile") {
		properties = append(properties, "gsxSigningStoreFile="+androidSigningStoreFile(cfg))
	}
	if !hasGradleProperty(properties, "gsxSigningStorePassword") {
		value := "<redacted:" + signing.StorePasswordEnv + ">"
		properties = append(properties, "gsxSigningStorePassword="+value)
	}
	if !hasGradleProperty(properties, "gsxSigningKeyAlias") {
		properties = append(properties, "gsxSigningKeyAlias="+signing.KeyAlias)
	}
	if !hasGradleProperty(properties, "gsxSigningKeyPassword") {
		keyPasswordEnv := firstNonEmpty(signing.KeyPasswordEnv, signing.StorePasswordEnv)
		value := "<redacted:" + keyPasswordEnv + ">"
		properties = append(properties, "gsxSigningKeyPassword="+value)
	}
	return properties
}

func signingEnvValue(name, field string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("signing config %s is required", field)
	}
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("signing environment variable %s is not set", name)
	}
	return value, nil
}

func androidSigningStoreFile(cfg nativeBuild) string {
	if cfg.signing == nil {
		return ""
	}
	storeFile := cfg.signing.Android.StoreFile
	if storeFile == "" || filepath.IsAbs(storeFile) || cfg.signingConfigPath == "" {
		return storeFile
	}
	return filepath.Clean(filepath.Join(filepath.Dir(cfg.signingConfigPath), storeFile))
}

func redactGradleProperties(properties []string) []string {
	redacted := make([]string, 0, len(properties))
	for _, property := range properties {
		name, _, ok := strings.Cut(property, "=")
		if !ok {
			name = property
		}
		if isSecretGradleProperty(name) {
			redacted = append(redacted, name+"=<redacted>")
			continue
		}
		redacted = append(redacted, property)
	}
	return redacted
}

func isSecretGradleProperty(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "gsxsigningstorepassword", "gsxsigningkeypassword":
		return true
	default:
		return false
	}
}

func androidExpectedArtifacts(project, flavor string, tasks []string) []expectedArtifact {
	var artifacts []expectedArtifact
	for _, task := range tasks {
		if strings.Contains(task, ":") && !strings.Contains(task, ":app:") && !strings.HasPrefix(task, "app:") {
			continue
		}
		name := taskName(task)
		switch {
		case name == "assembleDebug":
			artifacts = appendArtifact(artifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk/debug/app-debug.apk"))
		case name == "assembleRelease":
			artifacts = appendArtifact(artifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk/release/app-release.apk"))
		case name == "bundleRelease":
			artifacts = appendArtifact(artifacts, "android_aab", filepath.Join(project, "app/build/outputs/bundle/release/app-release.aab"))
		case flavor != "" && name == androidAssembleTask(flavor, false):
			artifacts = appendArtifact(artifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk", flavor, "debug", "app-"+flavor+"-debug.apk"))
		case flavor != "" && name == androidAssembleTask(flavor, true):
			artifacts = appendArtifact(artifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk", flavor, "release", "app-"+flavor+"-release.apk"))
		case flavor != "" && name == androidBundleTask(flavor):
			artifacts = appendArtifact(artifacts, "android_aab", filepath.Join(project, "app/build/outputs/bundle", flavor+"Release", "app-"+flavor+"-release.aab"))
		}
	}
	return artifacts
}

func hasGradleProperty(properties []string, name string) bool {
	prefix := name + "="
	for _, property := range properties {
		if property == name || strings.HasPrefix(property, prefix) {
			return true
		}
	}
	return false
}

func androidAssembleTask(flavor string, release bool) string {
	buildType := "Debug"
	if release {
		buildType = "Release"
	}
	if flavor == "" {
		return "assemble" + buildType
	}
	return "assemble" + gradleVariantFlavor(flavor) + buildType
}

func androidBundleTask(flavor string) string {
	if flavor == "" {
		return "bundleRelease"
	}
	return "bundle" + gradleVariantFlavor(flavor) + "Release"
}

func taskName(task string) string {
	if _, name, ok := strings.Cut(task, ":app:"); ok {
		return name
	}
	if strings.HasPrefix(task, "app:") {
		return strings.TrimPrefix(task, "app:")
	}
	if index := strings.LastIndex(task, ":"); index >= 0 {
		return task[index+1:]
	}
	return task
}

func gradleVariantFlavor(flavor string) string {
	if flavor == "" {
		return ""
	}
	return strings.ToUpper(flavor[:1]) + flavor[1:]
}

func buildIOS(ctx context.Context, cfg nativeBuild) error {
	if err := buildRunner.Run(ctx, cfg.project, "xcodegen", "generate"); err != nil {
		return err
	}
	derivedData := cfg.derivedDataPath
	cleanupDerivedData := false
	if derivedData == "" {
		var err error
		derivedData, err = os.MkdirTemp("", "gsxnative-derived-data-*")
		if err != nil {
			return err
		}
		cleanupDerivedData = true
	} else if err := os.MkdirAll(derivedData, 0755); err != nil {
		return err
	}
	if cleanupDerivedData {
		defer os.RemoveAll(derivedData)
	}
	action := "build"
	archivePath := iosArchivePath(cfg)
	if cfg.release {
		action = "archive"
	}
	args := []string{
		"-project", filepath.Join(cfg.project, cfg.xcodeProject+".xcodeproj"),
		"-scheme", cfg.scheme,
		"-destination", iosBuildDestination(cfg.simulator),
		"-derivedDataPath", derivedData,
	}
	if cfg.configuration != "" {
		args = append(args, "-configuration", cfg.configuration)
	}
	if archivePath != "" {
		args = append(args, "-archivePath", archivePath)
	}
	if cfg.signing != nil && cfg.signing.IOS.AllowProvisioningUpdates {
		args = append(args, "-allowProvisioningUpdates")
	}
	if cfg.environment != "" {
		args = append(args, "GSX_ENVIRONMENT="+cfg.environment)
	}
	if cfg.serverURL != "" {
		args = append(args, "GSX_SERVER_URL="+cfg.serverURL)
	}
	args = append(args, iosSigningBuildSettings(cfg)...)
	args = append(args, action)
	if err := buildRunner.Run(ctx, cfg.project, "xcodebuild", args...); err != nil {
		return err
	}
	if !cfg.release || cfg.exportOptions == "" {
		return nil
	}
	exportArgs := []string{
		"-exportArchive",
		"-archivePath", archivePath,
		"-exportOptionsPlist", cfg.exportOptions,
		"-exportPath", iosExportPath(cfg),
	}
	if cfg.signing != nil && cfg.signing.IOS.AllowProvisioningUpdates {
		exportArgs = append(exportArgs, "-allowProvisioningUpdates")
	}
	return buildRunner.Run(ctx, cfg.project, "xcodebuild", exportArgs...)
}

func iosSigningBuildSettings(cfg nativeBuild) []string {
	if !cfg.release || cfg.signing == nil || !cfg.signing.IOS.hasAny() {
		return nil
	}
	signing := cfg.signing.IOS
	var settings []string
	if signing.TeamID != "" {
		settings = append(settings, "DEVELOPMENT_TEAM="+signing.TeamID)
	}
	if signing.BundleID != "" {
		settings = append(settings, "PRODUCT_BUNDLE_IDENTIFIER="+signing.BundleID)
	}
	if signing.CodeSignStyle != "" {
		settings = append(settings, "CODE_SIGN_STYLE="+signing.CodeSignStyle)
	}
	if signing.ProvisioningProfile != "" {
		settings = append(settings, "PROVISIONING_PROFILE_SPECIFIER="+signing.ProvisioningProfile)
	}
	if signing.CodeSignIdentity != "" {
		settings = append(settings, "CODE_SIGN_IDENTITY="+signing.CodeSignIdentity)
	}
	return settings
}

func iosSigningSummary(cfg nativeBuild) *signingSummary {
	if !cfg.release || cfg.signing == nil || !cfg.signing.IOS.hasAny() {
		return nil
	}
	signing := cfg.signing.IOS
	return &signingSummary{
		Config:                   cfg.signingConfigPath,
		TeamID:                   signing.TeamID,
		BundleID:                 signing.BundleID,
		CodeSignStyle:            signing.CodeSignStyle,
		ProvisioningProfile:      signing.ProvisioningProfile,
		CodeSignIdentity:         signing.CodeSignIdentity,
		AllowProvisioningUpdates: signing.AllowProvisioningUpdates,
		ExportOptions:            cfg.exportOptions,
	}
}

func androidSigningSummary(cfg nativeBuild) *signingSummary {
	if !cfg.release || cfg.signing == nil || !cfg.signing.Android.hasAny() {
		return nil
	}
	signing := cfg.signing.Android
	return &signingSummary{
		Config:           cfg.signingConfigPath,
		StoreFile:        androidSigningStoreFile(cfg),
		StorePasswordEnv: signing.StorePasswordEnv,
		KeyAlias:         signing.KeyAlias,
		KeyPasswordEnv:   firstNonEmpty(signing.KeyPasswordEnv, signing.StorePasswordEnv),
	}
}

func iosArchivePath(cfg nativeBuild) string {
	if !cfg.release {
		return cfg.archivePath
	}
	return firstNonEmpty(cfg.archivePath, filepath.Join(cfg.project, "build", cfg.scheme+".xcarchive"))
}

func iosExportPath(cfg nativeBuild) string {
	return firstNonEmpty(cfg.exportPath, filepath.Join(cfg.project, "build", "export"))
}

func iosExpectedArtifacts(cfg nativeBuild) []expectedArtifact {
	if !cfg.release {
		if cfg.derivedDataPath == "" {
			return nil
		}
		return []expectedArtifact{{Kind: "ios_simulator_app", Path: iosSimulatorAppPath(cfg)}}
	}
	artifacts := []expectedArtifact{{Kind: "ios_archive", Path: iosArchivePath(cfg)}}
	if cfg.exportOptions != "" {
		artifacts = append(artifacts, expectedArtifact{Kind: "ios_export", Path: iosExportPath(cfg)})
	}
	return artifacts
}

func iosSimulatorAppPath(cfg nativeBuild) string {
	configuration := cfg.configuration
	if configuration == "" {
		configuration = "Debug"
	}
	return filepath.Join(cfg.derivedDataPath, "Build", "Products", configuration+"-iphonesimulator", cfg.scheme+".app")
}

func appendArtifact(artifacts []expectedArtifact, kind, path string) []expectedArtifact {
	for _, artifact := range artifacts {
		if artifact.Kind == kind && artifact.Path == path {
			return artifacts
		}
	}
	return append(artifacts, expectedArtifact{Kind: kind, Path: path})
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
	if destination := os.Getenv("IOS_SIMULATOR_DESTINATION"); destination != "" {
		return destination
	}
	if name := os.Getenv("IOS_SIMULATOR_NAME"); name != "" {
		return name
	}
	return "generic/platform=iOS Simulator"
}

func defaultIOSDestination(release bool) string {
	if release {
		return "generic/platform=iOS"
	}
	return defaultSimulatorName()
}

func defaultIOSConfiguration(release bool) string {
	if release {
		return "Release"
	}
	return ""
}

func iosBuildDestination(simulator string) string {
	simulator = strings.TrimSpace(simulator)
	if simulator == "" {
		return "generic/platform=iOS Simulator"
	}
	if strings.HasPrefix(simulator, "generic/") || strings.Contains(simulator, "platform=") {
		return simulator
	}
	return "platform=iOS Simulator,name=" + simulator
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
