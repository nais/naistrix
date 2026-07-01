// Package input provides helpers for prompting the user for interactive input from the terminal, such as confirmations,
// free-form text, and selection from a list of options.
package input

import (
	"github.com/pterm/pterm"
)

// Input prompts the user for a free-form string value and returns the response.
func Input(prompt string) (string, error) {
	if !interactive() {
		return "", ErrNotInteractive
	}
	return pterm.DefaultInteractiveTextInput.Show(prompt)
}
