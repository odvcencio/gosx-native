package android

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestEmitCounter(t *testing.T) {
	data, err := os.ReadFile("../../testdata/expected/nir/counter.json")
	if err != nil {
		t.Fatalf("read nir: %v", err)
	}
	var mod nir.Module
	if err := json.Unmarshal(data, &mod); err != nil {
		t.Fatalf("unmarshal nir: %v", err)
	}
	var buf bytes.Buffer
	if err := Emit(&mod, &buf); err != nil {
		t.Fatalf("emit: %v", err)
	}

	expected, err := os.ReadFile("../../testdata/expected/emit/android/Counter.kt")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(buf.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("emit mismatch.\nGot:\n%s\n\nExpected:\n%s", buf.String(), expected)
	}
}

func TestEmitNoPropComponentUsesLegalPropsClass(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name:  "EmptyDemo",
			Props: &nir.Props{},
			Body:  &nir.Text{Value: "ok"},
		}},
	}
	var buf bytes.Buffer
	if err := Emit(mod, &buf); err != nil {
		t.Fatalf("emit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "class EmptyDemoProps") {
		t.Fatalf("expected no-prop props class, got:\n%s", out)
	}
	if strings.Contains(out, "data class EmptyDemoProps(") {
		t.Fatalf("empty Kotlin data classes do not compile, got:\n%s", out)
	}
}
