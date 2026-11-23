package input

import (
	"context"
	"errors"
	"os"

	"github.com/nais/naistrix/internal/terminal"
)

// ErrNotInteractive is returned when a prompt cannot use an interactive terminal,
// including when interaction is explicitly disabled through [WithNonInteractive].
var ErrNotInteractive = errors.New("no interactive terminal available")

// interactive reports whether prompts can read from an interactive terminal. Both standard input and standard output
// must be terminals: standard input so a response can be read, and standard output so the user can see the prompt. It is
// a package variable so tests can override the detection.
var interactive = func() bool {
	return terminal.IsTerminal(os.Stdin) && terminal.IsTerminal(os.Stdout)
}

type nonInteractiveKey struct{}

// WithNonInteractive returns a context that disables prompts, even with a terminal.
// It applies only to this context and its descendants and never implies confirmation or consent.
func WithNonInteractive(ctx context.Context) context.Context {
	return context.WithValue(ctx, nonInteractiveKey{}, true)
}

// IsInteractive reports whether ctx permits interaction and standard input and output are terminals.
// Pass [context.Background] if you only want to check terminal availability.
func IsInteractive(ctx context.Context) bool {
	disabled, _ := ctx.Value(nonInteractiveKey{}).(bool)
	return !disabled && interactive()
}
