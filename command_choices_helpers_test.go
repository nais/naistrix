package naistrix

import "github.com/nais/naistrix/input"

// SetChoicePromptForTest replaces choice selection and terminal detection for external tests only.
// Tests must restore the hooks and must not run in parallel.
func SetChoicePromptForTest(selectChoice func(string, []string, ...input.SelectOptionFunc) (string, error)) func() {
	previousSelect, previousInteractive := selectArgument, argumentsInteractive
	selectArgument, argumentsInteractive = selectChoice, func() bool { return true }
	return func() {
		selectArgument, argumentsInteractive = previousSelect, previousInteractive
	}
}
