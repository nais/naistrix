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

// SetArgumentConfirmationForTest replaces repeatable argument confirmation for external tests only.
// Tests must restore the hook and must not run in parallel.
func SetArgumentConfirmationForTest(confirm func(string, ...input.ConfirmOptionFunc) (bool, error)) func() {
	previous := confirmArgument
	confirmArgument = confirm
	return func() { confirmArgument = previous }
}
