package main

import (
	"context"
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

func TestBuildUnsupportedScene3DStopsBeforeNativeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "SceneDemo.kt")
	err = runBuild([]string{
		"android",
		"--source", filepath.Join(root, "testdata/corpus/go/scene3d_compute.gsx"),
		"--output", output,
		"--project", t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected Scene3D build failure")
	}
	if !strings.Contains(err.Error(), "Scene3D native backend does not support <ComputeParticles> yet") {
		t.Fatalf("expected Scene3D diagnostic, got %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected no native commands after validation failure, got %#v", fake.commands)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("expected no generated output, stat err: %v", err)
	}
}
