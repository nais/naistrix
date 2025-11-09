package naistrix

import (
	"fmt"
	"io"
	"regexp"

	"github.com/nais/naistrix/output"
	"github.com/pterm/pterm"
)

// coloredText is a regular expression that matches custom tags for info, warn, and error formatting inline in a string.
var coloredText = regexp.MustCompile(`<(info|warn|error)>(.*?)</(info|warn|error)>`)

// OutputWriter is used to write output to the user, with support for different verbosity levels and output formats.
type OutputWriter struct {
	writer io.Writer
	level  *Count
}

// NewOutputWriter creates a new output writer.
func NewOutputWriter(writer io.Writer, level *Count) *OutputWriter {
	pterm.SetDefaultOutput(writer)
	return &OutputWriter{
		writer: writer,
		level:  level,
	}
}

// Table creates a new table that can be rendered to the destination.
func (w *OutputWriter) Table(opts ...output.TableOptionFunc) *output.Table {
	return output.NewTable(w.writer, opts...)
}

// JSON creates a new JSON output that can be rendered to the destination.
func (w *OutputWriter) JSON(opts ...output.JSONOptionFunc) *output.JSON {
	return output.NewJSON(w.writer, opts...)
}

// YAML creates a new YAML output that can be rendered to the destination.
func (w *OutputWriter) YAML() *output.YAML {
	return output.NewYAML(w.writer)
}

// Confirm prompts the user with a yes/no question and returns the response. The question will get a " [y/N]" suffix
// automatically.
func (w *OutputWriter) Confirm(format string, a ...any) (bool, error) {
	return pterm.DefaultInteractiveConfirm.Show(fmt.Sprintf(format, a...))
}

// Infoln writes a line of informational output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in all verbosity levels.
func (w *OutputWriter) Infoln(a ...any) {
	pterm.Info.WithWriter(w.writer).Println(a...)
}

// Infof writes formatted informational output to the destination. This outputs in all verbosity levels.
func (w *OutputWriter) Infof(format string, a ...any) {
	pterm.Info.WithWriter(w.writer).Printf(format, a...)
}

// Warnln writes a line of warning output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in all verbosity levels.
func (w *OutputWriter) Warnln(a ...any) {
	pterm.Warning.WithWriter(w.writer).Println(a...)
}

// Warnf writes formatted warning output to the destination. This outputs in all verbosity levels.
func (w *OutputWriter) Warnf(format string, a ...any) {
	pterm.Warning.WithWriter(w.writer).Printf(format, a...)
}

// Errorln writes a line of error output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in all verbosity levels.
func (w *OutputWriter) Errorln(a ...any) {
	pterm.Error.WithWriter(w.writer).Println(a...)
}

// Errorf writes formatted error output to the destination. This outputs in all verbosity levels.
func (w *OutputWriter) Errorf(format string, a ...any) {
	pterm.Error.WithWriter(w.writer).Printf(format, a...)
}

// Println writes a line of output to the destination, appending a newline at the end. Spaces are added between
// arguments. This outputs in all verbosity levels.
func (w *OutputWriter) Println(a ...any) {
	pterm.Println(colorizeMultiple(a)...)
}

// Printf writes formatted output to the destination. This outputs in all verbosity levels.
func (w *OutputWriter) Printf(format string, a ...any) {
	pterm.Printf(colorize(format), a...)
}

// Verboseln writes a line of verbose output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in OutputVerbosityLevelVerbose and higher levels.
func (w *OutputWriter) Verboseln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelVerbose {
		return
	}

	_, _ = fmt.Fprintln(w.writer, colorizeMultiple(a)...)
}

// Verbosef writes formatted verbose output to the destination. This outputs in OutputVerbosityLevelVerbose and higher
// levels.
func (w *OutputWriter) Verbosef(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelVerbose {
		return
	}

	_, _ = fmt.Fprintf(w.writer, colorize(format), a...)
}

// Debugln writes a line of debug output to the destination, appending a newline at the end. Spaces are added between
// arguments. This outputs in OutputVerbosityLevelDebug and higher levels.
func (w *OutputWriter) Debugln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelDebug {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	pterm.Debug.WithWriter(w.writer).Println(colorizeMultiple(a)...)
}

// Debugf writes formatted debug output to the destination. This outputs in OutputVerbosityLevelDebug and higher levels.
func (w *OutputWriter) Debugf(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelDebug {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	pterm.Debug.WithWriter(w.writer).Printf(colorize(format), a...)
}

// Traceln writes a line of trace output to the destination, appending a newline at the end. Spaces are added between
// arguments. This outputs in OutputVerbosityLevelTrace level.
func (w *OutputWriter) Traceln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelTrace {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = " TRACE "
	pterm.Debug.WithWriter(w.writer).WithPrefix(prefix).Println(colorizeMultiple(a)...)
}

// Tracef writes formatted trace output to the destination. This outputs in OutputVerbosityLevelTrace level.
func (w *OutputWriter) Tracef(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelTrace {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = " TRACE "
	pterm.Debug.WithWriter(w.writer).WithPrefix(prefix).Printf(colorize(format), a...)
}

// colorizeMultiple applies colorization to a slice of values. Each value will be converted to a string.
func colorizeMultiple(s []any) []any {
	ret := make([]any, len(s))
	for i, str := range s {
		ret[i] = colorize(fmt.Sprint(str))
	}
	return ret
}

// colorize applies colorization to a string based on custom tags.
func colorize(s string) string {
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
