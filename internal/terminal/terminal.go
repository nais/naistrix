package terminal

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether file is connected to a terminal.
func IsTerminal(file *os.File) bool {
	return term.IsTerminal(int(file.Fd())) // #nosec G115
}
