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
	opts, err := parseDevOptions([]string{"--target=ios", "--once", "--interval=25ms", "--source", "src/app.gsx"})
	if err != nil {
		t.Fatalf("parse dev options: %v", err)
	}
	if opts.target != "ios" || !opts.once || opts.interval != 25*time.Millisecond {
		t.Fatalf("unexpected options: %#v", opts)
	}
	if got := strings.Join(opts.buildArgs, " "); got != "--source src/app.gsx" {
		t.Fatalf("unexpected forwarded build args: %q", got)
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
