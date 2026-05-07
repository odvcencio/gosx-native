package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitScaffoldsNativeProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{dir, "--name", "SampleApp", "--module", "com.example.sample"}); err != nil {
		t.Fatalf("init: %v", err)
	}

	assertFileContains(t, filepath.Join(dir, "src/app.gsx"), "func Home(props HomeProps) Node")
	assertFileContains(t, filepath.Join(dir, "ios/project.yml"), "name: SampleApp")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/SampleAppApp.swift"), "GSXRouter(initial: GSXRoute(\"home\"))")
	assertFileContains(t, filepath.Join(dir, "ios/SampleApp/Generated/App.g.swift"), "public struct Home: GSXComponent")
	assertFileContains(t, filepath.Join(dir, "android/settings.gradle.kts"), "project(\":gsx-nativekit\").projectDir")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/com/example/sample/MainActivity.kt"), "rememberGSXRouter(GSXRoute(\"home\"))")
	assertFileContains(t, filepath.Join(dir, "android/app/src/main/kotlin/generated/App.kt"), "fun Home(props: HomeProps)")

	var cfg projectConfig
	data, err := os.ReadFile(filepath.Join(dir, "gosxnative.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Source != "src/app.gsx" || cfg.IOS.Scheme != "SampleApp" || cfg.Android.Project != "android" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestInitRefusesNonEmptyDirectoryWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runInit([]string{"--name", "SampleApp", dir})
	if err == nil {
		t.Fatalf("expected non-empty directory error")
	}
	if !strings.Contains(err.Error(), "is not empty") {
		t.Fatalf("expected non-empty directory error, got %v", err)
	}
}

func TestBuildUsesDiscoveredProjectConfig(t *testing.T) {
	fake := useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	if err := runBuild([]string{"android"}); err != nil {
		t.Fatalf("build from config: %v", err)
	}

	generated := filepath.Join(dir, "android/app/src/main/kotlin/generated/App.kt")
	assertFileContains(t, generated, "GSXScene3D(scene = GSXScene3DScene(")
	if len(fake.commands) != 1 {
		t.Fatalf("expected one Gradle command, got %#v", fake.commands)
	}
	if fake.commands[0].dir != filepath.Join(dir, "android") {
		t.Fatalf("expected Gradle to run in generated Android project, got %#v", fake.commands[0])
	}
}

func TestBuildIOSUsesDiscoveredProjectConfig(t *testing.T) {
	fake := useFakeBuildRunner(t)
	dir := filepath.Join(t.TempDir(), "sample-app")
	if err := runInit([]string{"--name", "SampleApp", "--module", "com.example.sample", dir}); err != nil {
		t.Fatalf("init: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	if err := runBuild([]string{"ios"}); err != nil {
		t.Fatalf("build from config: %v", err)
	}

	if len(fake.commands) != 2 {
		t.Fatalf("expected xcodegen and xcodebuild, got %#v", fake.commands)
	}
	if fake.commands[0].dir != filepath.Join(dir, "ios") || fake.commands[0].name != "xcodegen" {
		t.Fatalf("unexpected xcodegen command: %#v", fake.commands[0])
	}
	joinedArgs := strings.Join(fake.commands[1].args, " ")
	if !strings.Contains(joinedArgs, filepath.Join(dir, "ios", "SampleApp.xcodeproj")) ||
		!strings.Contains(joinedArgs, "-scheme SampleApp") {
		t.Fatalf("unexpected xcodebuild args: %q", joinedArgs)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("expected %s to contain %q, got:\n%s", path, want, data)
	}
}
