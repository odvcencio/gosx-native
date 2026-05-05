package diagnostics

import "testing"

func TestFormatHumanReadable(t *testing.T) {
	d := Diagnostic{
		Code:     "E2103",
		Severity: SeverityError,
		Span:     Span{File: "foo.gsx", StartLine: 42, StartCol: 5},
		Message:  "missing per-target implementation",
		Help:     "provide an Android implementation",
	}
	got := d.String()
	want := "error[E2103]: missing per-target implementation\n  --> foo.gsx:42:5\nhelp: provide an Android implementation\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
