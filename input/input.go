// Package input provides helpers for prompting the user for interactive input from the terminal, such as confirmations,
// free-form text, and selection from a list of options.
// Prompts honor [WithNonInteractive] and check cancellation before starting.
// They do not cancel an already running prompt.
package input

import (
	"context"

	"github.com/pterm/pterm"
)

// Input prompts the user for a free-form string value and returns the response.
func Input(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !IsInteractive(ctx) {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.Show(prompt)
}
