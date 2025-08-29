package naistrix

import (
	"fmt"
	"io"
	"os"

	"github.com/nais/naistrix/output"
)

// Output is an interface that defines methods for writing output to a destination.
type Output interface {
	io.Writer

	// Println writes a line of output to the destination, appending a newline at the end. Spaces are added between
	// arguments.
	Println(a ...any)

	// Printf writes formatted output to the destination.
	Printf(format string, a ...any)

	// Table creates a new table that can be rendered to the destination.
	Table(opts ...output.TableOptionFunc) *output.Table

	// JSON creates a new JSON output that can be rendered to the destination.
	JSON(opts ...output.JSONOptionFunc) *output.JSON
}

type writer struct {
	w io.Writer
}

func (w *writer) Println(a ...any) {
	_, _ = fmt.Fprintln(w.w, a...)
}

func (w *writer) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.w, format, a...)
}

func (w *writer) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w *writer) Table(opts ...output.TableOptionFunc) *output.Table {
	return output.NewTable(w, opts...)
}

func (w *writer) JSON(opts ...output.JSONOptionFunc) *output.JSON {
	return output.NewJSON(w, opts...)
}

func NewWriter(w io.Writer) Output {
	return &writer{w: w}
}

// Stdout returns an Output that writes to standard output.
func Stdout() Output {
	return NewWriter(os.Stdout)
}

// Discard returns an Output that discards all messages.
func Discard() Output {
	return NewWriter(io.Discard)
}
