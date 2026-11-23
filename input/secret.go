package input

import (
	"context"

	"github.com/pterm/pterm"
)

// Secret prompts the user for sensitive text, masking typed characters with asterisks.
func Secret(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !IsInteractive(ctx) {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.WithMask("*").Show(prompt)
}
