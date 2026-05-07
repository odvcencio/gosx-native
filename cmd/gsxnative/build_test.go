package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordedCommand struct {
	dir  string
	name string
	args []string
}

type fakeBuildRunner struct {
	commands []recordedCommand
}

func (f *fakeBuildRunner) Run(_ context.Context, dir, name string, args ...string) error {
	f.commands = append(f.commands, recordedCommand{
		dir:  dir,
		name: name,
		args: append([]string(nil), args...),
	})
	return nil
}

func useFakeBuildRunner(t *testing.T) *fakeBuildRunner {
	t.Helper()
	previous := buildRunner
	fake := &fakeBuildRunner{}
	buildRunner = fake
	t.Cleanup(func() {
		buildRunner = previous
	})
	return fake
}

func TestBuildAndroidRegeneratesSourceAndRunsGradle(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Kotlin: %v", err)
	}
	if !strings.Contains(string(data), "fun Counter(props: CounterProps)") {
		t.Fatalf("expected generated Counter composable, got:\n%s", data)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	if fake.commands[0].dir != project || fake.commands[0].name != "gradle" {
		t.Fatalf("unexpected Gradle command: %#v", fake.commands[0])
	}
	joinedArgs := strings.Join(fake.commands[0].args, " ")
	if !strings.Contains(joinedArgs, ":gsx-nativekit:assembleRelease") || !strings.Contains(joinedArgs, ":app:assembleDebug") {
		t.Fatalf("expected runtime/app assemble tasks, got %q", joinedArgs)
	}
}

func TestBuildAndroidWritesArtifactManifest(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.kt")
	manifestPath := filepath.Join(t.TempDir(), "gsxnative-artifacts.json")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--release",
		"--artifact-manifest", manifestPath,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read artifact manifest: %v", err)
	}
	var manifest buildArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse artifact manifest: %v", err)
	}
	if manifest.Version != 1 || len(manifest.Targets) != 1 {
		t.Fatalf("unexpected artifact manifest: %#v", manifest)
	}
	target := manifest.Targets[0]
	if target.Target != "android" || !target.Release || target.GeneratedOutput != output {
		t.Fatalf("unexpected android target manifest: %#v", target)
	}
	if !containsString(target.BuildTasks, ":app:assembleRelease") {
		t.Fatalf("expected release task in manifest: %#v", target.BuildTasks)
	}
	if !containsArtifact(target.ExpectedArtifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk/release/app-release.apk")) {
		t.Fatalf("expected release APK artifact in manifest: %#v", target.ExpectedArtifacts)
	}
}

func TestBuildAndroidReleaseEnvFlavorWritesManifest(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.kt")
	manifestPath := filepath.Join(t.TempDir(), "gsxnative-artifacts.json")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--release",
		"--env", "staging",
		"--flavor", "demo",
		"--artifact-manifest", manifestPath,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	joinedArgs := strings.Join(fake.commands[0].args, " ")
	if !strings.Contains(joinedArgs, "-PgsxEnvironment=staging") || !strings.Contains(joinedArgs, ":app:assembleDemoRelease") {
		t.Fatalf("expected environment property and flavor task, got %q", joinedArgs)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read artifact manifest: %v", err)
	}
	var manifest buildArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse artifact manifest: %v", err)
	}
	if manifest.Environment != "staging" || len(manifest.Targets) != 1 {
		t.Fatalf("unexpected artifact manifest: %#v", manifest)
	}
	target := manifest.Targets[0]
	if target.Environment != "staging" || target.Flavor != "demo" {
		t.Fatalf("unexpected target environment/flavor: %#v", target)
	}
	if !containsString(target.BuildProperties, "gsxEnvironment=staging") {
		t.Fatalf("expected environment Gradle property: %#v", target.BuildProperties)
	}
	if !containsArtifact(target.ExpectedArtifacts, "android_apk", filepath.Join(project, "app/build/outputs/apk/demo/release/app-demo-release.apk")) {
		t.Fatalf("expected flavored release APK artifact in manifest: %#v", target.ExpectedArtifacts)
	}
}

func TestBuildAndroidReleaseSigningConfigWritesRedactedManifest(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	signingDir := t.TempDir()
	signingPath := filepath.Join(signingDir, "signing.json")
	if err := os.WriteFile(signingPath, []byte(`{
  "android": {
    "store_file": "release.jks",
    "store_password_env": "GSX_STORE_PASSWORD",
    "key_alias": "upload",
    "key_password_env": "GSX_KEY_PASSWORD"
  }
}
`), 0644); err != nil {
		t.Fatalf("write signing config: %v", err)
	}
	t.Setenv("GSX_STORE_PASSWORD", "store-secret")
	t.Setenv("GSX_KEY_PASSWORD", "key-secret")
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.kt")
	manifestPath := filepath.Join(t.TempDir(), "gsxnative-artifacts.json")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--release",
		"--signing-config", signingPath,
		"--artifact-manifest", manifestPath,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	joinedArgs := strings.Join(fake.commands[0].args, " ")
	if !strings.Contains(joinedArgs, "-PgsxSigningStoreFile="+filepath.Join(signingDir, "release.jks")) ||
		!strings.Contains(joinedArgs, "-PgsxSigningStorePassword=store-secret") ||
		!strings.Contains(joinedArgs, "-PgsxSigningKeyAlias=upload") ||
		!strings.Contains(joinedArgs, "-PgsxSigningKeyPassword=key-secret") {
		t.Fatalf("expected Android signing Gradle properties, got %q", joinedArgs)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read artifact manifest: %v", err)
	}
	if strings.Contains(string(data), "store-secret") || strings.Contains(string(data), "key-secret") {
		t.Fatalf("manifest leaked signing secrets:\n%s", data)
	}
	var manifest buildArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse artifact manifest: %v", err)
	}
	target := manifest.Targets[0]
	if !containsString(target.BuildProperties, "gsxSigningStorePassword=<redacted>") ||
		!containsString(target.BuildProperties, "gsxSigningKeyPassword=<redacted>") {
		t.Fatalf("expected redacted signing properties: %#v", target.BuildProperties)
	}
	if target.Signing == nil || target.Signing.StoreFile != filepath.Join(signingDir, "release.jks") ||
		target.Signing.StorePasswordEnv != "GSX_STORE_PASSWORD" || target.Signing.KeyAlias != "upload" ||
		target.Signing.KeyPasswordEnv != "GSX_KEY_PASSWORD" {
		t.Fatalf("unexpected signing summary: %#v", target.Signing)
	}
}

func TestBuildIOSRegeneratesSourceAndRunsXcodeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.swift")
	err = runBuild([]string{
		"ios",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--simulator", "iPhone 16",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Swift: %v", err)
	}
	if !strings.Contains(string(data), "struct Counter: GSXComponent") {
		t.Fatalf("expected generated Counter struct, got:\n%s", data)
	}
	if len(fake.commands) != 2 {
		t.Fatalf("expected xcodegen and xcodebuild commands, got %#v", fake.commands)
	}
	if fake.commands[0].dir != project || fake.commands[0].name != "xcodegen" {
		t.Fatalf("unexpected xcodegen command: %#v", fake.commands[0])
	}
	if fake.commands[1].dir != project || fake.commands[1].name != "xcodebuild" {
		t.Fatalf("unexpected xcodebuild command: %#v", fake.commands[1])
	}
	joinedArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(joinedArgs, "-scheme CounterDemo") || !strings.Contains(joinedArgs, "platform=iOS Simulator,name=iPhone 16") {
		t.Fatalf("unexpected xcodebuild args: %q", joinedArgs)
	}
}

func TestBuildIOSForwardsEnvironmentBuildSetting(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.swift")
	err = runBuild([]string{
		"ios",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--env", "staging",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 2 {
		t.Fatalf("expected xcodegen and xcodebuild commands, got %#v", fake.commands)
	}
	joinedArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(joinedArgs, "GSX_ENVIRONMENT=staging") {
		t.Fatalf("expected iOS environment build setting, got %q", joinedArgs)
	}
}

func TestBuildIOSReleaseSigningConfigForwardsXcodeSettings(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	signingDir := t.TempDir()
	signingPath := filepath.Join(signingDir, "signing.json")
	exportOptions := filepath.Join(signingDir, "ExportOptions.plist")
	if err := os.WriteFile(signingPath, []byte(`{
  "ios": {
    "team_id": "ABCDE12345",
    "bundle_id": "com.example.signed",
    "code_sign_style": "Manual",
    "provisioning_profile": "GSX Release",
    "code_sign_identity": "Apple Distribution",
    "export_options_plist": "ExportOptions.plist",
    "allow_provisioning_updates": true
  }
}
`), 0644); err != nil {
		t.Fatalf("write signing config: %v", err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.swift")
	manifestPath := filepath.Join(t.TempDir(), "gsxnative-artifacts.json")
	err = runBuild([]string{
		"ios",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--release",
		"--signing-config", signingPath,
		"--artifact-manifest", manifestPath,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 3 {
		t.Fatalf("expected xcodegen, archive, and export commands, got %#v", fake.commands)
	}
	archiveArgs := strings.Join(fake.commands[1].args, " ")
	for _, want := range []string{
		"-allowProvisioningUpdates",
		"DEVELOPMENT_TEAM=ABCDE12345",
		"PRODUCT_BUNDLE_IDENTIFIER=com.example.signed",
		"CODE_SIGN_STYLE=Manual",
		"PROVISIONING_PROFILE_SPECIFIER=GSX Release",
		"CODE_SIGN_IDENTITY=Apple Distribution",
	} {
		if !strings.Contains(archiveArgs, want) {
			t.Fatalf("expected archive arg %q in %q", want, archiveArgs)
		}
	}
	exportArgs := strings.Join(fake.commands[2].args, " ")
	if !strings.Contains(exportArgs, "-exportOptionsPlist "+exportOptions) ||
		!strings.Contains(exportArgs, "-allowProvisioningUpdates") {
		t.Fatalf("unexpected export args: %q", exportArgs)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read artifact manifest: %v", err)
	}
	var manifest buildArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse artifact manifest: %v", err)
	}
	target := manifest.Targets[0]
	if target.Signing == nil || target.Signing.TeamID != "ABCDE12345" ||
		target.Signing.BundleID != "com.example.signed" || target.Signing.ExportOptions != exportOptions ||
		!target.Signing.AllowProvisioningUpdates {
		t.Fatalf("unexpected iOS signing summary: %#v", target.Signing)
	}
}

func TestBuildIOSReleaseArchivesAndExports(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.swift")
	archivePath := filepath.Join(project, "build", "CounterDemo.xcarchive")
	exportOptions := filepath.Join(project, "ExportOptions.plist")
	exportPath := filepath.Join(project, "build", "exported")
	err = runBuild([]string{
		"ios",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--release",
		"--archive-path", archivePath,
		"--export-options-plist", exportOptions,
		"--export-path", exportPath,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(fake.commands) != 3 {
		t.Fatalf("expected xcodegen, archive, and export commands, got %#v", fake.commands)
	}
	archiveArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(archiveArgs, "-configuration Release") ||
		!strings.Contains(archiveArgs, "-destination generic/platform=iOS") ||
		!strings.Contains(archiveArgs, "-archivePath "+archivePath) ||
		!strings.Contains(archiveArgs, " archive") {
		t.Fatalf("unexpected archive args: %q", archiveArgs)
	}
	exportArgs := strings.Join(fake.commands[2].args, " ")
	if !strings.Contains(exportArgs, "-exportArchive") ||
		!strings.Contains(exportArgs, "-archivePath "+archivePath) ||
		!strings.Contains(exportArgs, "-exportOptionsPlist "+exportOptions) ||
		!strings.Contains(exportArgs, "-exportPath "+exportPath) {
		t.Fatalf("unexpected export args: %q", exportArgs)
	}
}

func TestIOSBuildDestinationDefaultsToGenericSimulator(t *testing.T) {
	t.Setenv("IOS_SIMULATOR_DESTINATION", "")
	t.Setenv("IOS_SIMULATOR_NAME", "")

	if got := defaultSimulatorName(); got != "generic/platform=iOS Simulator" {
		t.Fatalf("unexpected default simulator destination: %q", got)
	}
	if got := iosBuildDestination(defaultSimulatorName()); got != "generic/platform=iOS Simulator" {
		t.Fatalf("unexpected generic destination: %q", got)
	}
	if got := iosBuildDestination("iPhone 16"); got != "platform=iOS Simulator,name=iPhone 16" {
		t.Fatalf("unexpected named simulator destination: %q", got)
	}
	if got := iosBuildDestination("platform=iOS Simulator,name=Any iOS Simulator Device"); got != "platform=iOS Simulator,name=Any iOS Simulator Device" {
		t.Fatalf("unexpected explicit destination: %q", got)
	}
	if got := defaultIOSDestination(true); got != "generic/platform=iOS" {
		t.Fatalf("unexpected release destination: %q", got)
	}
	if got := defaultIOSConfiguration(true); got != "Release" {
		t.Fatalf("unexpected release configuration: %q", got)
	}
}

func TestBuildCodegenOnlySkipsNativeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "Counter.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", t.TempDir(),
		"--codegen-only",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Kotlin: %v", err)
	}
	if !strings.Contains(string(data), "fun Counter(props: CounterProps)") {
		t.Fatalf("expected generated Counter composable, got:\n%s", data)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected no native commands in codegen-only mode, got %#v", fake.commands)
	}
}

func TestBuildAndroidScene3DRegeneratesSourceAndRunsGradle(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "SceneDemo.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/scene3d.gsx"),
		"--output", output,
		"--project", project,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Kotlin: %v", err)
	}
	if !strings.Contains(string(data), "GSXScene3D(scene = GSXScene3DScene(") {
		t.Fatalf("expected generated Scene3D surface, got:\n%s", data)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
}

func TestBuildAndroidScene3DSpreadRegeneratesSourceAndRunsGradle(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "SpreadDemo.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/scene3d_spread.gsx"),
		"--output", output,
		"--project", project,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Kotlin: %v", err)
	}
	generated := string(data)
	if !strings.Contains(generated, "val mesh: Map<String, Any?>") || !strings.Contains(generated, "gsxScene3DSpreadString(props.mesh") {
		t.Fatalf("expected generated Scene3D spread surface, got:\n%s", data)
	}
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
}

func TestBuildInvalidScene3DStopsBeforeNativeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "SceneDemo.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/scene3d_html_invalid.gsx"),
		"--output", output,
		"--project", t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected Scene3D build failure")
	}
	if !strings.Contains(err.Error(), "Scene3D <Html> requires literal html, markup, content, or static children") {
		t.Fatalf("expected Scene3D diagnostic, got %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected no native commands after validation failure, got %#v", fake.commands)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("expected no generated output, stat err: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsArtifact(artifacts []expectedArtifact, kind, path string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind && artifact.Path == path {
			return true
		}
	}
	return false
}
