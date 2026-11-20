package input

import (
	"context"

	"github.com/pterm/pterm"
)

// Secret prompts the user for sensitive text, masking typed characters with asterisks.
func Secret(prompt string) (string, error) {
	return SecretContext(context.Background(), prompt)
}

// SecretContext is [Secret] with invocation-scoped interaction policy and a cancellation check before prompting.
func SecretContext(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !IsInteractiveContext(ctx) {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.WithMask("*").Show(prompt)
}
