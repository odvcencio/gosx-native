package diagnostics

import (
	"fmt"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Span struct {
	File      string
	StartLine int
	StartCol  int
}

type Diagnostic struct {
	Code     string
	Severity Severity
	Span     Span
	Message  string
	Help     string
}

func (d Diagnostic) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s[%s]: %s\n", d.Severity, d.Code, d.Message)
	fmt.Fprintf(&sb, "  --> %s:%d:%d\n", d.Span.File, d.Span.StartLine, d.Span.StartCol)
	if d.Help != "" {
		fmt.Fprintf(&sb, "help: %s\n", d.Help)
	}
	return sb.String()
}
