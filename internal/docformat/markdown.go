package docformat

import (
	"html"
	"regexp"
	"strings"

	"github.com/nais/naistrix/internal/color"
)

// Match formatting pairs first, then any other angle-bracket tags, such as <team> or <env>.
var tags = regexp.MustCompile(color.TaggedText.String() + `|<[^<>]+>`)

// Markdown converts matching terminal formatting tags to Markdown and escapes other tags so placeholders
// remain visible rather than being interpreted as HTML. Like terminal color formatting, nested tags are not supported.
func Markdown(s string) string {
	return tags.ReplaceAllStringFunc(s, func(match string) string {
		m := color.TaggedText.FindStringSubmatch(match)
		if m == nil || m[1] != m[3] {
			return html.EscapeString(match)
		}
		if m[2] == "" {
			return ""
		}
		if m[1] == "info" {
			return inlineCode(m[2])
		}
		return "**" + html.EscapeString(m[2]) + "**"
	})
}

func inlineCode(content string) string {
	// Use a delimiter longer than any backtick run in the content.
	delimiter := "`"
	for strings.Contains(content, delimiter) {
		delimiter += "`"
	}
	if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
		content = " " + content + " "
	}
	return delimiter + content + delimiter
}
