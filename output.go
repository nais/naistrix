package naistrix

import (
	"fmt"
	"io"
	"os"
)

// Output is an interface that defines methods for writing output to a destination.
type Output interface {
	io.Writer

	// Println writes a line of output to the destination, appending a newline at the end. Spaces are added between
	// arguments.
	Println(a ...any)

	// Printf writes formatted output to the destination.
	Printf(format string, a ...any)
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
