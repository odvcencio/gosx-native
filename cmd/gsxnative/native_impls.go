package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/odvcencio/gosx-native/target"
)

type nativeImplementation struct {
	Target string
	Line   int
	Inline bool
}

type nativeImplementationGroup struct {
	Name            string
	Line            int
	Implementations []nativeImplementation
}

var nativeFunctionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^func\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
	regexp.MustCompile(`^func\s+\([^)]*\)\s*([A-Za-z_][A-Za-z0-9_]*)\b`),
	regexp.MustCompile(`^fun\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
}

const nativeDirectivePrefix = "//gosx:native"

func validateNativeImplementationsFile(path string, tgt target.Target) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	groups, err := parseNativeImplementations(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := validateNativeImplementations(groups, tgt); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func parseNativeImplementations(src []byte) ([]nativeImplementationGroup, error) {
	lines := strings.Split(string(src), "\n")
	var groups []nativeImplementationGroup
	groupIndexes := make(map[string]int)
	var pending []nativeImplementation
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if isNativeImplementationDirective(line) {
			impl, err := parseNativeImplementationDirective(line, lineNo)
			if err != nil {
				return nil, err
			}
			if impl.Inline {
				addNativeImplementationGroup(&groups, groupIndexes, nativeImplementationGroup{
					Name:            fmt.Sprintf("inline@%d", lineNo),
					Line:            lineNo,
					Implementations: []nativeImplementation{impl},
				})
				continue
			}
			pending = append(pending, impl)
			continue
		}
		if len(pending) == 0 {
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}
		if name, ok := nativeFunctionName(line); ok {
			addNativeImplementationGroup(&groups, groupIndexes, nativeImplementationGroup{
				Name:            name,
				Line:            pending[0].Line,
				Implementations: append([]nativeImplementation(nil), pending...),
			})
			pending = nil
			continue
		}
		return nil, fmt.Errorf("line %d: //gosx:native must precede a function declaration or use inline block form", pending[0].Line)
	}
	if len(pending) > 0 {
		return nil, fmt.Errorf("line %d: //gosx:native must precede a function declaration or use inline block form", pending[0].Line)
	}
	return groups, nil
}

func parseNativeImplementationDirective(line string, lineNo int) (nativeImplementation, error) {
	body := strings.TrimSpace(strings.TrimPrefix(line, nativeDirectivePrefix))
	if body == "" {
		return nativeImplementation{}, fmt.Errorf("line %d: //gosx:native missing target", lineNo)
	}
	targetPart := body
	inline := false
	if before, after, ok := strings.Cut(body, "{"); ok {
		targetPart = strings.TrimSpace(before)
		inline = strings.Contains(after, "}")
		if !inline {
			return nativeImplementation{}, fmt.Errorf("line %d: inline //gosx:native block must close on the same line", lineNo)
		}
	}
	pieces := strings.Fields(targetPart)
	if len(pieces) == 0 {
		return nativeImplementation{}, fmt.Errorf("line %d: //gosx:native missing target", lineNo)
	}
	if len(pieces) > 1 {
		return nativeImplementation{}, fmt.Errorf("line %d: //gosx:native accepts one target before the optional inline block", lineNo)
	}
	rawTarget := pieces[0]
	if name, value, ok := strings.Cut(rawTarget, "="); ok {
		if strings.ToLower(strings.TrimSpace(name)) != "target" {
			return nativeImplementation{}, fmt.Errorf("line %d: //gosx:native field %q is unsupported", lineNo, name)
		}
		rawTarget = value
	}
	normalized, err := normalizeNativeImplementationTarget(rawTarget)
	if err != nil {
		return nativeImplementation{}, fmt.Errorf("line %d: %w", lineNo, err)
	}
	return nativeImplementation{Target: normalized, Line: lineNo, Inline: inline}, nil
}

func validateNativeImplementations(groups []nativeImplementationGroup, tgt target.Target) error {
	var diagnostics []string
	for _, group := range groups {
		seen := make(map[string]int, len(group.Implementations))
		for _, impl := range group.Implementations {
			if firstLine := seen[impl.Target]; firstLine != 0 {
				diagnostics = append(diagnostics, fmt.Sprintf(
					"line %d: native implementation %s repeats //gosx:native %s first declared on line %d",
					impl.Line,
					group.Name,
					impl.Target,
					firstLine,
				))
				continue
			}
			seen[impl.Target] = impl.Line
		}
		if nativeGroupSupportsTarget(group, tgt) {
			continue
		}
		diagnostics = append(diagnostics, fmt.Sprintf(
			"line %d: native implementation %s missing //gosx:native %s for %s (available: %s)",
			group.Line,
			group.Name,
			requiredNativeImplementationTarget(tgt),
			tgt,
			nativeImplementationTargets(group),
		))
	}
	if len(diagnostics) > 0 {
		return errors.New(strings.Join(diagnostics, "\n"))
	}
	return nil
}

func addNativeImplementationGroup(groups *[]nativeImplementationGroup, indexes map[string]int, group nativeImplementationGroup) {
	if index, ok := indexes[group.Name]; ok {
		existing := &(*groups)[index]
		if group.Line < existing.Line {
			existing.Line = group.Line
		}
		existing.Implementations = append(existing.Implementations, group.Implementations...)
		return
	}
	indexes[group.Name] = len(*groups)
	*groups = append(*groups, group)
}

func nativeGroupSupportsTarget(group nativeImplementationGroup, tgt target.Target) bool {
	for _, impl := range group.Implementations {
		if nativeImplementationSupportsTarget(impl.Target, tgt) {
			return true
		}
	}
	return false
}

func nativeImplementationSupportsTarget(implTarget string, tgt target.Target) bool {
	switch implTarget {
	case "go":
		return true
	case "swift":
		return tgt == target.IOS
	case "kotlin":
		return tgt == target.Android
	default:
		return false
	}
}

func requiredNativeImplementationTarget(tgt target.Target) string {
	switch tgt {
	case target.IOS:
		return "swift"
	case target.Android:
		return "kotlin"
	default:
		return string(tgt)
	}
}

func nativeImplementationTargets(group nativeImplementationGroup) string {
	seen := make(map[string]bool, len(group.Implementations))
	var targets []string
	for _, impl := range group.Implementations {
		if seen[impl.Target] {
			continue
		}
		seen[impl.Target] = true
		targets = append(targets, impl.Target)
	}
	sort.Strings(targets)
	return strings.Join(targets, ", ")
}

func normalizeNativeImplementationTarget(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "ios", "swift":
		return "swift", nil
	case "android", "kotlin":
		return "kotlin", nil
	case "go", "portable":
		return "go", nil
	default:
		return "", fmt.Errorf("unsupported //gosx:native target %q (supported: swift, kotlin, go)", raw)
	}
}

func isNativeImplementationDirective(line string) bool {
	if line == nativeDirectivePrefix {
		return true
	}
	return strings.HasPrefix(line, nativeDirectivePrefix+" ") || strings.HasPrefix(line, nativeDirectivePrefix+"\t")
}

func nativeFunctionName(line string) (string, bool) {
	for _, pattern := range nativeFunctionPatterns {
		if match := pattern.FindStringSubmatch(line); len(match) == 2 {
			return match[1], true
		}
	}
	return "", false
}
