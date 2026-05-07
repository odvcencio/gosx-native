package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDevOnceRegeneratesSourceOnly(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "Counter.kt")
	err = runDev([]string{
		"android",
		"--once",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", t.TempDir(),
	})
	if err != nil {
		t.Fatalf("dev once: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated Kotlin: %v", err)
	}
	if !strings.Contains(string(data), "fun Counter(props: CounterProps)") {
		t.Fatalf("expected generated Counter composable, got:\n%s", data)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected dev once to skip native tools, got %#v", fake.commands)
	}
}

func TestParseDevOptionsSupportsTargetFlag(t *testing.T) {
	opts, err := parseDevOptions([]string{"--target=ios", "--once", "--build", "--launch", "--device=booted", "--interval=25ms", "--source", "src/app.gsx"})
	if err != nil {
		t.Fatalf("parse dev options: %v", err)
	}
	if opts.target != "ios" || !opts.once || !opts.build || !opts.install || !opts.launch || opts.device != "booted" || opts.interval != 25*time.Millisecond {
		t.Fatalf("unexpected options: %#v", opts)
	}
	if got := strings.Join(opts.buildArgs, " "); got != "--source src/app.gsx" {
		t.Fatalf("unexpected forwarded build args: %q", got)
	}
}

func TestDevBuildModeRunsNativeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "Counter.swift")
	project := t.TempDir()
	err = runDev([]string{
		"ios",
		"--once",
		"--build",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--simulator", "iPhone 16",
	})
	if err != nil {
		t.Fatalf("dev build: %v", err)
	}
	if len(fake.commands) != 2 {
		t.Fatalf("expected xcodegen and xcodebuild commands, got %#v", fake.commands)
	}
	if fake.commands[0].name != "xcodegen" || fake.commands[1].name != "xcodebuild" {
		t.Fatalf("unexpected native commands: %#v", fake.commands)
	}
	if joinedArgs := strings.Join(fake.commands[1].args, " "); !strings.Contains(joinedArgs, "platform=iOS Simulator,name=iPhone 16") {
		t.Fatalf("expected simulator destination to reach xcodebuild, got %q", joinedArgs)
	}
}

func TestDevLaunchAndroidInstallsAndStartsApp(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	apkPath := filepath.Join(project, "app/build/outputs/apk/debug/app-debug.apk")
	if err := os.MkdirAll(filepath.Dir(apkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apkPath, []byte("debug-apk"), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "Counter.kt")
	err = runDev([]string{
		"android",
		"--once",
		"--launch",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--device", "emulator-5554",
		"--android-package", "com.example.sample",
	})
	if err != nil {
		t.Fatalf("dev launch android: %v", err)
	}
	if len(fake.commands) != 3 {
		t.Fatalf("expected Gradle, adb install, and adb launch, got %#v", fake.commands)
	}
	if fake.commands[0].name != "gradle" || fake.commands[1].name != "adb" || fake.commands[2].name != "adb" {
		t.Fatalf("unexpected command sequence: %#v", fake.commands)
	}
	installArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(installArgs, "-s emulator-5554 install -r "+apkPath) {
		t.Fatalf("unexpected adb install args: %q", installArgs)
	}
	launchArgs := strings.Join(fake.commands[2].args, " ")
	if !strings.Contains(launchArgs, "shell am start -n com.example.sample/.MainActivity") {
		t.Fatalf("unexpected adb launch args: %q", launchArgs)
	}
}

func TestDevLaunchIOSUsesStableDerivedDataAndSimctl(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "Counter.swift")
	err = runDev([]string{
		"ios",
		"--once",
		"--launch",
		"--source", filepath.Join(root, "testdata/corpus/go/counter.gsx"),
		"--output", output,
		"--project", project,
		"--scheme", "CounterDemo",
		"--device", "booted",
		"--ios-bundle-id", "com.example.CounterDemo",
	})
	if err != nil {
		t.Fatalf("dev launch ios: %v", err)
	}
	if len(fake.commands) != 4 {
		t.Fatalf("expected xcodegen, xcodebuild, simctl install, and simctl launch, got %#v", fake.commands)
	}
	buildArgs := strings.Join(fake.commands[1].args, " ")
	derivedData := filepath.Join(project, "build", "gsxnative-derived-data")
	if !strings.Contains(buildArgs, "-derivedDataPath "+derivedData) {
		t.Fatalf("expected stable derived data in xcodebuild args, got %q", buildArgs)
	}
	appPath := filepath.Join(derivedData, "Build", "Products", "Debug-iphonesimulator", "CounterDemo.app")
	installArgs := strings.Join(fake.commands[2].args, " ")
	if fake.commands[2].name != "xcrun" || !strings.Contains(installArgs, "simctl install booted "+appPath) {
		t.Fatalf("unexpected simctl install: %#v", fake.commands[2])
	}
	launchArgs := strings.Join(fake.commands[3].args, " ")
	if fake.commands[3].name != "xcrun" || !strings.Contains(launchArgs, "simctl launch booted com.example.CounterDemo") {
		t.Fatalf("unexpected simctl launch: %#v", fake.commands[3])
	}
}

func TestDevInstallRejectsCodegenOnly(t *testing.T) {
	_, err := parseDevOptions([]string{"android", "--install", "--codegen-only"})
	if err == nil || !strings.Contains(err.Error(), "codegen-only") {
		t.Fatalf("expected codegen-only conflict, got %v", err)
	}
}

func TestDevSnapshotDetectsSourceChanges(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "app.gsx")
	if err := os.WriteFile(source, []byte("package app\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotDevFiles(dir)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snapshot) != 1 {
		t.Fatalf("expected one watched file, got %#v", snapshot)
	}
	time.Sleep(time.Millisecond)
	if err := os.WriteFile(source, []byte("package app\n// updated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, changed, err := changedDevFiles(dir, snapshot)
	if err != nil {
		t.Fatalf("changed files: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed source file")
	}
}
