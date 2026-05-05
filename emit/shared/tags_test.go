package shared

import "testing"

func TestSwiftUITagMapping(t *testing.T) {
	cases := map[string]string{
		"vstack": "VStack",
		"hstack": "HStack",
		"text":   "Text",
		"button": "Button",
		"view":   "Group",
	}
	for nir, want := range cases {
		got := SwiftUITag(nir)
		if got != want {
			t.Errorf("SwiftUITag(%q) = %q, want %q", nir, got, want)
		}
	}
}

func TestUnknownTagReportsError(t *testing.T) {
	got := SwiftUITag("nonexistent_tag")
	if got != "" {
		t.Errorf("expected empty for unknown, got %q", got)
	}
}

func TestComposeTagMapping(t *testing.T) {
	cases := map[string]string{
		"vstack": "Column",
		"hstack": "Row",
		"text":   "Text",
		"button": "Button",
	}
	for nir, want := range cases {
		got := ComposeTag(nir)
		if got != want {
			t.Errorf("ComposeTag(%q) = %q, want %q", nir, got, want)
		}
	}
}

func TestUnknownComposeTagReportsError(t *testing.T) {
	got := ComposeTag("nonexistent_tag")
	if got != "" {
		t.Errorf("expected empty for unknown, got %q", got)
	}
}
