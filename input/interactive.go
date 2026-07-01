package input

import (
	"errors"
	"os"

	"golang.org/x/term"
)

// ErrNotInteractive is returned by the prompt functions in this package when there is no interactive terminal available
// to read input from, for example when running in CI or when standard input or output is piped.
var ErrNotInteractive = errors.New("no interactive terminal available")

// interactive reports whether prompts can read from an interactive terminal. Both standard input and standard output
// must be terminals: standard input so a response can be read, and standard output so the user can see the prompt. It is
// a package variable so tests can override the detection.
var interactive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) // #nosec G115
}
