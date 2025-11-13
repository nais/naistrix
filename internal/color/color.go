package color

import (
	"fmt"
	"regexp"

	"github.com/pterm/pterm"
)

// coloredText is a regular expression that matches custom tags for info, warn, and error formatting inline in a string.
var coloredText = regexp.MustCompile(`<(info|warn|error)>(.*?)</(info|warn|error)>`)

// Colorize applies colorization to a string based on custom tags.
func Colorize(s string) string {
	return coloredText.ReplaceAllStringFunc(s, func(s string) string {
		m := coloredText.FindStringSubmatch(s)
		openTag, content, closeTag := m[1], m[2], m[3]

		if openTag != closeTag {
			return s
		}

		var printer func(...any) string
		switch openTag {
		case "info":
			printer = pterm.FgLightCyan.Sprint
		case "warn":
			printer = pterm.FgYellow.Sprint
		case "error":
			printer = pterm.FgLightRed.Sprint
		default:
			return s
		}

		return printer(content)
	})
}

// ColorizeAny applies colorization to a slice of values. Each value will be converted to a string.
func ColorizeAny(s []any) []any {
	ret := make([]any, len(s))
	for i, str := range s {
		ret[i] = Colorize(fmt.Sprint(str))
	}
	return ret
}

// ColorizeStrings applies colorization to a slice of strings.
func ColorizeStrings(s []string) []string {
	as := make([]any, len(s))
	for i, v := range s {
		as[i] = v
	}

	coloredAny := ColorizeAny(as)
	for i, v := range coloredAny {
		s[i] = v.(string)
	}

	return s
}
