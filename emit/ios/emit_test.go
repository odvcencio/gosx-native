package ios

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"m31labs.dev/gosx/nir"
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

	expected, err := os.ReadFile("../../testdata/expected/emit/ios/Counter.swift")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(buf.Bytes()), bytes.TrimSpace(expected)) {
		t.Fatalf("emit mismatch.\nGot:\n%s\n\nExpected:\n%s", buf.String(), expected)
	}
}
