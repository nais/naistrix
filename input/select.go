package input

import (
	"fmt"
	"maps"
	"slices"

	"github.com/pterm/pterm"
)

// SelectOptionFunc is a function used to control options for the [Select] function.
type SelectOptionFunc func(*selectOptions)

// SelectWithAutoSelectSingleOption sets the default result value of the confirmation to true.
func SelectWithAutoSelectSingleOption() SelectOptionFunc {
	return func(opt *selectOptions) {
		opt.autoSelectSingleOption = true
	}
}

// selectOptions holds options for the [Select] function.
type selectOptions struct {
	autoSelectSingleOption bool
}

// Select prompts the user to select one value from the provided options and returns the selected value. The string
// representation of each option are shown in the prompt.
func Select[T any](prompt string, selection []T, opts ...SelectOptionFunc) (T, error) {
	options := &selectOptions{
		autoSelectSingleOption: false,
	}

	for _, o := range opts {
		o(options)
	}

	var empty T

	if len(selection) == 0 {
		return empty, fmt.Errorf("no options provided")
	}

	if len(selection) == 1 && options.autoSelectSingleOption {
		return selection[0], nil
	}

	labels := make(map[string]struct{})
	for i, o := range selection {
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

	for _, o := range selection {
		if fmt.Sprint(o) == chosen {
			return o, nil
		}
	}

	return empty, fmt.Errorf("selected value %q not found in options", chosen)
}
