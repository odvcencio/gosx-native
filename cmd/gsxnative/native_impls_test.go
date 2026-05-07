package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odvcencio/gosx-native/target"
)

func TestParseNativeImplementationsGroupsFunctionTargets(t *testing.T) {
	groups, err := parseNativeImplementations([]byte(`
package nativefixture

//gosx:native swift
func formatTitle(value string) string {
	return value
}

//gosx:native kotlin
fun formatTitle(value: String): String {
	return value
}
`))
	if err != nil {
		t.Fatalf("parse native implementations: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %#v, want one group", groups)
	}
	group := groups[0]
	if group.Name != "formatTitle" || len(group.Implementations) != 2 {
		t.Fatalf("unexpected group: %#v", group)
	}
	if err := validateNativeImplementations(groups, target.IOS); err != nil {
		t.Fatalf("ios validate: %v", err)
	}
	if err := validateNativeImplementations(groups, target.Android); err != nil {
		t.Fatalf("android validate: %v", err)
	}
}

func TestValidateNativeImplementationsRejectsDuplicateTarget(t *testing.T) {
	groups, err := parseNativeImplementations([]byte(`
package nativefixture

//gosx:native swift
//gosx:native ios
func formatTitle(value string) string {
	return value
}
`))
	if err != nil {
		t.Fatalf("parse native implementations: %v", err)
	}
	err = validateNativeImplementations(groups, target.IOS)
	if err == nil {
		t.Fatalf("expected duplicate swift implementation")
	}
	if !strings.Contains(err.Error(), "repeats //gosx:native swift") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}

func TestValidateNativeImplementationsRejectsMissingTarget(t *testing.T) {
	groups, err := parseNativeImplementations([]byte(`
package nativefixture

//gosx:native swift
func formatTitle(value string) string {
	return value
}
`))
	if err != nil {
		t.Fatalf("parse native implementations: %v", err)
	}
	err = validateNativeImplementations(groups, target.Android)
	if err == nil {
		t.Fatalf("expected missing kotlin implementation")
	}
	if !strings.Contains(err.Error(), "missing //gosx:native kotlin for android") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}

func TestValidateNativeImplementationsRejectsUnsupportedTarget(t *testing.T) {
	_, err := parseNativeImplementations([]byte(`
//gosx:native objc
func formatTitle(value string) string {
	return value
}
`))
	if err == nil {
		t.Fatalf("expected unsupported target")
	}
	if !strings.Contains(err.Error(), `unsupported //gosx:native target "objc"`) {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}

func TestRunCheckRejectsNativeImplementationMissingTarget(t *testing.T) {
	source := writeNativeFixture(t, `
package nativefixture

//gosx:native swift
func formatTitle(value string) string {
	return value
}

func NativeFixture() Node {
	return <div>Native</div>
}
`)
	err := runCheck([]string{"android", source})
	if err == nil {
		t.Fatalf("expected missing native implementation")
	}
	if !strings.Contains(err.Error(), "missing //gosx:native kotlin for android") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}

func TestBuildNativeImplementationStopsBeforeNativeTools(t *testing.T) {
	fake := useFakeBuildRunner(t)
	source := writeNativeFixture(t, `
package nativefixture

//gosx:native swift
func formatTitle(value string) string {
	return value
}

func NativeFixture() Node {
	return <div>Native</div>
}
`)
	output := filepath.Join(t.TempDir(), "NativeFixture.kt")
	err := runBuild([]string{
		"android",
		"--source", source,
		"--output", output,
		"--project", t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected missing native implementation")
	}
	if !strings.Contains(err.Error(), "missing //gosx:native kotlin for android") {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("expected no native commands after validation failure, got %#v", fake.commands)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("expected no generated output, stat err: %v", err)
	}
}

func writeNativeFixture(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "native_fixture.gsx")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	return path
}
