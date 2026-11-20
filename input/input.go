// Package input provides helpers for prompting the user for interactive input from the terminal, such as confirmations,
// free-form text, and selection from a list of options.
// Context-aware helpers honor [WithNonInteractive] and check cancellation before starting a prompt.
// They do not cancel an already running prompt. Helpers without a context preserve their terminal-only behavior.
package input

import (
	"context"

	"github.com/pterm/pterm"
)

// Input prompts the user for a free-form string value and returns the response.
func Input(prompt string) (string, error) {
	return InputContext(context.Background(), prompt)
}

// InputContext is [Input] with invocation-scoped interaction policy and a cancellation check before prompting.
func InputContext(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !IsInteractiveContext(ctx) {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.Show(prompt)
}
