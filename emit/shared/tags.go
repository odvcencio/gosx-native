package shared

var swiftuiTags = map[string]string{
	"view":   "Group",
	"vstack": "VStack",
	"hstack": "HStack",
	"text":   "Text",
	"button": "Button",
}

// SwiftUITag returns the SwiftUI widget name for a symbolic NIR tag, or "" if
// the tag has no mapping.
func SwiftUITag(nirTag string) string {
	return swiftuiTags[nirTag]
}
