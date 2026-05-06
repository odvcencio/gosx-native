package main

import (
	"bytes"
	"os"
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

func TestCompileGoSXScene3DPrintsNIR(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "compile", "../../testdata/corpus/go/scene3d.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), `"tag": "scene3d"`) || !strings.Contains(out.String(), `"tag": "mesh"`) {
		t.Fatalf("expected Scene3D NIR tags, got:\n%s", out.String())
	}
}

func TestCheckIOSGoSXCounterPasses(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "check", "ios", "../../testdata/corpus/go/counter.gsx")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\nstderr:\n%s", err, stderr.String())
	}
}

func TestCheckIOSScene3DPassesStaticSurface(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "check", "ios", "../../testdata/corpus/go/scene3d.gsx")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\nstderr:\n%s", err, stderr.String())
	}
}

func TestCheckIOSScene3DComputeReportsUnsupportedTag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "check", "ios", "../../testdata/corpus/go/scene3d_compute.gsx")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected check failure")
	}
	if !strings.Contains(stderr.String(), "Scene3D native backend does not support <ComputeParticles> yet") {
		t.Fatalf("expected Scene3D diagnostic, got:\n%s", stderr.String())
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

func TestEmitIOSScene3DPrintsRuntimeSurface(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/scene3d.gsx")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(out.String(), "GSXScene3DView(scene: GSXScene3DScene(") || !strings.Contains(out.String(), `GSXScene3DNode(id: "hero", tag: "mesh"`) {
		t.Fatalf("expected Scene3D Swift surface, got:\n%s", out.String())
	}
}

func TestEmitAndroidScene3DPrintsRuntimeSurface(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/scene3d.gsx")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(out.String(), "GSXScene3D(scene = GSXScene3DScene(") || !strings.Contains(out.String(), `GSXScene3DNode(id = "hero", tag = "mesh"`) {
		t.Fatalf("expected Scene3D Kotlin surface, got:\n%s", out.String())
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

func TestEmitIOSGoSXPanelMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/panel.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Panel.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Panel Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXGreeterMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/greeter.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Greeter.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Greeter Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXDerivedMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/derived.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Derived.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Derived Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXToggleMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/conditional.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Toggle.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Toggle Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXProfileMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/component_ref.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Profile.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Profile Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXRosterMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/loop.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Roster.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Roster Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXFormControlsMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/form_controls.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/FormControls.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected FormControls Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitIOSGoSXExpressionsMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "ios", "../../testdata/corpus/go/expressions.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/ios/Expressions.swift")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Expressions Swift golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
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

func TestEmitAndroidGoSXPanelMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/panel.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Panel.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Panel Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXGreeterMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/greeter.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Greeter.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Greeter Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXDerivedMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/derived.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Derived.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Derived Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXToggleMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/conditional.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Toggle.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Toggle Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXProfileMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/component_ref.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Profile.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Profile Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXRosterMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/loop.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Roster.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Roster Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXFormControlsMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/form_controls.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/FormControls.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected FormControls Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func TestEmitAndroidGoSXExpressionsMatchesGolden(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "emit", "android", "../../testdata/corpus/go/expressions.gsx")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := readExpected(t, "../../testdata/expected/emit/android/Expressions.kt")
	if !bytes.Equal(bytes.TrimSpace(out.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("expected Expressions Kotlin golden.\nGot:\n%s\n\nExpected:\n%s", out.String(), expected)
	}
}

func readExpected(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read expected %s: %v", path, err)
	}
	return data
}
