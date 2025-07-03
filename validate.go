package naistrix

import (
	"context"
)

// ValidateFunc is a function that will be executed before the command's RunFunc is executed.
//
// The args passed to this function is the arguments passed to the command by the end-user.
type ValidateFunc func(ctx context.Context, args []string) error

// ValidateExactArgs checks that the user has provided an exact amount of arguments to the command.
func ValidateExactArgs(n int) ValidateFunc {
	return func(_ context.Context, args []string) error {
		if len(args) != n {
			return Errorf("Expected exactly %d arguments, got %d", n, len(args))
		}

		return nil
	}
}

// ValidateMinArgs checks that the user has provided a minimum amount of arguments to the command.
func ValidateMinArgs(n int) ValidateFunc {
	return func(_ context.Context, args []string) error {
		if len(args) < n {
			return Errorf("Expected at least %d arguments, got %d", n, len(args))
		}

		return nil
	}
}
