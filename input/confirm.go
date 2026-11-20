package input

import (
	"context"

	"github.com/pterm/pterm"
)

// ConfirmOptionFunc is a function used to control options for the [Confirm] function.
type ConfirmOptionFunc func(*confirmOptions)

// ConfirmWithDefaultTrue sets the default result value of the confirmation to true.
func ConfirmWithDefaultTrue() ConfirmOptionFunc {
	return func(opt *confirmOptions) {
		opt.defaultValue = true
	}
}

// confirmOptions holds options for the [Confirm] function.
type confirmOptions struct {
	defaultValue bool
}

// Confirm prompts the user with a yes/no question and returns the response. The question will get a " [y/N]" suffix
// automatically.
func Confirm(prompt string, opts ...ConfirmOptionFunc) (bool, error) {
	return ConfirmContext(context.Background(), prompt, opts...)
}

// ConfirmContext is [Confirm] with invocation-scoped interaction policy and a cancellation check before prompting.
// Disabled interaction returns [ErrNotInteractive], even when the default answer is true.
func ConfirmContext(ctx context.Context, prompt string, opts ...ConfirmOptionFunc) (bool, error) {
	options := &confirmOptions{
		defaultValue: false,
	}

	for _, o := range opts {
		o(options)
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !IsInteractiveContext(ctx) {
		return false, ErrNotInteractive
	}

	return pterm.DefaultInteractiveConfirm.
		WithDefaultValue(options.defaultValue).
		Show(prompt)
}
