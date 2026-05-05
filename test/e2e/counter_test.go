//go:build smoke && darwin

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCounterEndToEnd(t *testing.T) {
	repoRoot, _ := filepath.Abs("../..")

	out, err := exec.Command("go", "run",
		filepath.Join(repoRoot, "cmd/gsxnative"),
		"emit", "ios",
		filepath.Join(repoRoot, "testdata/corpus/swift/counter.swift.gsx"),
	).Output()
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	tracked := filepath.Join(repoRoot, "examples/counter-ios/CounterDemo/Generated/Counter.swift")
	expected, err := os.ReadFile(tracked)
	if err != nil {
		t.Fatalf("read tracked Generated/Counter.swift: %v", err)
	}
	if !bytes.Equal(out, expected) {
		t.Fatalf("emit drift detected: regenerated Counter.swift differs from %s.\n"+
			"Run `make demo` and commit the regenerated file if the change is intentional.", tracked)
	}

	gen := exec.Command("xcodegen", "generate")
	gen.Dir = filepath.Join(repoRoot, "examples/counter-ios")
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	if err := gen.Run(); err != nil {
		t.Fatalf("xcodegen: %v", err)
	}

	build := exec.Command("xcodebuild",
		"-project", filepath.Join(repoRoot, "examples/counter-ios/CounterDemo.xcodeproj"),
		"-scheme", "CounterDemo",
		"-destination", "platform=iOS Simulator,name="+simName(),
		"build",
	)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("xcodebuild: %v", err)
	}
}

func simName() string {
	if n := os.Getenv("IOS_SIMULATOR_NAME"); n != "" {
		return n
	}
	return "iPhone 16"
}
