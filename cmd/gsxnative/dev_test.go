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
	opts, err := parseDevOptions([]string{"--target=ios", "--once", "--build", "--interval=25ms", "--source", "src/app.gsx"})
	if err != nil {
		t.Fatalf("parse dev options: %v", err)
	}
	if opts.target != "ios" || !opts.once || !opts.build || opts.interval != 25*time.Millisecond {
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
