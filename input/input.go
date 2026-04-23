package input

import (
	"fmt"
	"maps"
	"slices"

	"github.com/pterm/pterm"
)

// ConfirmOptionFunc is a function used to control options for the Confirm function.
type ConfirmOptionFunc func(*confirmOptions)

// ConfirmWithDefaultTrue sets the default result value of the confirmation to true.
func ConfirmWithDefaultTrue() ConfirmOptionFunc {
	return func(opt *confirmOptions) {
		opt.defaultValue = true
	}
}

// confirmOptions holds options for the Confirm function.
type confirmOptions struct {
	defaultValue bool
}

// Confirm prompts the user with a yes/no question and returns the response. The question will get a " [y/N]" suffix
// automatically.
func Confirm(prompt string, opts ...ConfirmOptionFunc) (bool, error) {
	options := &confirmOptions{
		defaultValue: false,
	}

	for _, o := range opts {
		o(options)
	}

	return pterm.DefaultInteractiveConfirm.
		WithDefaultValue(options.defaultValue).
		Show(prompt)
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
		return empty, fmt.Errorf("no options provided")
	}

	labels := make(map[string]struct{})
	for i, o := range options {
		lbl := fmt.Sprint(o)
		if _, exists := labels[lbl]; exists {
			return empty, fmt.Errorf("duplicate label: %s (index %d)", lbl, i)
		}
		labels[lbl] = struct{}{}
	}

	chosen, err := pterm.DefaultInteractiveSelect.
		WithOptions(slices.Collect(maps.Keys(labels))).
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
