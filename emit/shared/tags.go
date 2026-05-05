package shared

var swiftuiTags = map[string]string{
	"view":   "Group",
	"vstack": "VStack",
	"hstack": "HStack",
	"text":   "Text",
	"button": "Button",
}

var composeTags = map[string]string{
	"vstack": "Column",
	"hstack": "Row",
	"text":   "Text",
	"button": "Button",
}

// SwiftUITag returns the SwiftUI widget name for a symbolic NIR tag, or "" if
// the tag has no mapping.
func SwiftUITag(nirTag string) string {
	return swiftuiTags[nirTag]
}

// ComposeTag returns the Jetpack Compose widget name for a symbolic NIR tag,
// or "" if the tag has no mapping.
func ComposeTag(nirTag string) string {
	return composeTags[nirTag]
}
