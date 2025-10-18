package naistrix

import (
	"context"
)

// ValidateFunc is a function that will be executed before the command's RunFunc is executed.
//
// The args passed to this function contains the arguments passed to the command by the end-user.
type ValidateFunc func(ctx context.Context, args *Arguments) error

// ValidateExactArgs checks that the user has provided an exact amount of arguments to the command.
func ValidateExactArgs(n int) ValidateFunc {
	return func(_ context.Context, args *Arguments) error {
		if got := args.Len(); got != n {
			return Errorf("Expected exactly %d argument%s, got %d", n, plural(n), got)
		}

		return nil
	}
}

// ValidateMinArgs checks that the user has provided a minimum amount of arguments to the command.
func ValidateMinArgs(n int) ValidateFunc {
	return func(_ context.Context, args *Arguments) error {
		if got := args.Len(); got < n {
			return Errorf("Expected at least %d argument%s, got %d", n, plural(n), got)
		}

		return nil
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
