package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCompileCounterPrintsNIR(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "compile", "../../testdata/corpus/swift/counter.swift.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), `"name": "Counter"`) {
		t.Fatalf("expected Counter in NIR JSON, got:\n%s", out.String())
	}
}

func TestCompileGoSXCounterPrintsNIR(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "compile", "../../testdata/corpus/go/counter.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), `"source_language": "go"`) || !strings.Contains(out.String(), `"name": "Counter"`) {
		t.Fatalf("expected GoSX Counter in NIR JSON, got:\n%s", out.String())
	}
}

func TestEmitIOSCounterPrintsSwift(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/swift/counter.swift.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "struct Counter: GSXComponent") {
		t.Fatalf("expected Counter struct in emitted Swift, got:\n%s", out.String())
	}
}

func TestEmitIOSGoSXCounterPrintsSwift(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/counter.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "struct Counter: GSXComponent") || !strings.Contains(out.String(), "Button(\"+\") { count = count + 1 }") {
		t.Fatalf("expected Counter Swift from GoSX, got:\n%s", out.String())
	}
}

func TestEmitAndroidCounterPrintsKotlin(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/swift/counter.swift.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "fun Counter(props: CounterProps)") {
		t.Fatalf("expected Counter composable in emitted Kotlin, got:\n%s", out.String())
	}
}

func TestEmitAndroidGoSXCounterPrintsKotlin(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/counter.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "fun Counter(props: CounterProps)") || !strings.Contains(out.String(), "count = count + 1") {
		t.Fatalf("expected Counter Kotlin from GoSX, got:\n%s", out.String())
	}
}
