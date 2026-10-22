package input

import "github.com/pterm/pterm"

// Secret prompts the user for sensitive text, masking typed characters with asterisks.
func Secret(prompt string) (string, error) {
	if !interactive() {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.WithMask("*").Show(prompt)
}
