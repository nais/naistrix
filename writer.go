package naistrix

import (
	"fmt"
	"io"

	"github.com/nais/naistrix/output"
	"github.com/pterm/pterm"
)

type OutputWriter struct {
	writer io.Writer
	level  *Count
}

// Table creates a new table that can be rendered to the destination.
func (w *OutputWriter) Table(opts ...output.TableOptionFunc) *output.Table {
	return output.NewTable(w.writer, opts...)
}

// JSON creates a new JSON output that can be rendered to the destination.
func (w *OutputWriter) JSON(opts ...output.JSONOptionFunc) *output.JSON {
	return output.NewJSON(w.writer, opts...)
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

//

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
	_, _ = fmt.Fprintln(w.writer, a...)
}

// Printf writes formatted output to the destination. This outputs in all verbosity levels.
func (w *OutputWriter) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.writer, format, a...)
}

// Verboseln writes a line of verbose output to the destination, appending a newline at the end. Spaces are added between
// arguments. This outputs in OutputVerbosityLevelVerbose and higher levels.
func (w *OutputWriter) Verboseln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelVerbose {
		return
	}

	_, _ = fmt.Fprintln(w.writer, a...)
}

// Verbosef writes formatted verbose output to the destination. This outputs in OutputVerbosityLevelVerbose and higher
// levels.
func (w *OutputWriter) Verbosef(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelVerbose {
		return
	}

	_, _ = fmt.Fprintf(w.writer, format, a...)
}

// Debugln writes a line of debug output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in OutputVerbosityLevelDebug and higher levels.
func (w *OutputWriter) Debugln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelDebug {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = ""
	pterm.Debug.WithWriter(w.writer).WithPrefix(pterm.Prefix{}).Println(a...)
}

// Debugf writes formatted debug output to the destination. This outputs in OutputVerbosityLevelDebug and higher levels.
func (w *OutputWriter) Debugf(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelDebug {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = ""
	prefix.Style = nil
	pterm.Debug.WithWriter(w.writer).WithPrefix(pterm.Prefix{}).Printf(format, a...)
}

// Traceln writes a line of trace output to the destination, appending a newline at the end. Spaces are added
// between arguments. This outputs in OutputVerbosityLevelTrace level.
func (w *OutputWriter) Traceln(a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelTrace {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = ""
	pterm.Debug.WithWriter(w.writer).WithPrefix(prefix).Println(a...)
}

// Tracef writes formatted trace output to the destination. This outputs in OutputVerbosityLevelTrace level.
func (w *OutputWriter) Tracef(format string, a ...any) {
	if w == nil || *w.level < OutputVerbosityLevelTrace {
		return
	}

	pterm.EnableDebugMessages()
	defer pterm.DisableDebugMessages()
	prefix := pterm.Debug.Prefix
	prefix.Text = ""
	pterm.Debug.WithWriter(w.writer).WithPrefix(prefix).Printf(format, a...)
}
