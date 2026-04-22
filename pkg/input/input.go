package input

import (
	"fmt"

	"github.com/pterm/pterm"
)

// Confirm prompts the user with a yes/no question and returns the response. The question will get a " [y/N]" suffix
// automatically.
func Confirm(prompt string) (bool, error) {
	return pterm.DefaultInteractiveConfirm.Show(prompt)
}

// Input prompts the user for a free-form string value and returns the response.
func Input(prompt string) (string, error) {
	return pterm.DefaultInteractiveTextInput.Show(prompt)
}

// Select prompts the user to select one value from the provided options and returns the selected value. The string
// representation of each option are shown in the prompt.
func Select[T any](prompt string, options []T) (T, error) {
	var empty T

	if len(options) == 0 {
		return empty, fmt.Errorf("no options provided for selection")
	}

	labels := make([]string, len(options))
	for i, o := range options {
		labels[i] = fmt.Sprint(o)
	}

	chosen, err := pterm.DefaultInteractiveSelect.
		WithOptions(labels).
		WithFilterInputPlaceholder("[type to filter]").
		Show(prompt)
	if err != nil {
		return empty, err
	}

	for _, o := range options {
		if fmt.Sprint(o) == chosen {
			return o, nil
		}
	}

	return empty, fmt.Errorf("selected value %q not found in options", chosen)
}
