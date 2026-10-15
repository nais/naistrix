package naistrix

import (
	"context"
)

// ValidateFlagsFunc validates flags captured by the callback after they are synchronized, before argument choices or
// prompts are resolved. It also runs before automatic choice completion and should be side-effect-free and avoid
// output. Return an error to stop execution or completion.
type ValidateFlagsFunc func(ctx context.Context) error

// ValidateArgsFunc validates resolved positional arguments before the command's [RunFunc] is executed. The args passed
// to this function contain the complete, normalized values, including values collected through interactive prompting.
// The function is not executed during shell completion.
type ValidateArgsFunc func(ctx context.Context, args *Arguments) error

// ValidateExactArgs checks that the user has provided an exact amount of arguments to the command.
func ValidateExactArgs(n int) ValidateArgsFunc {
	return func(_ context.Context, args *Arguments) error {
		if got := args.Len(); got != n {
			return Errorf("Expected exactly %d argument%s, got %d", n, plural(n), got)
		}

		return nil
	}
}

// ValidateMinArgs checks that the user has provided a minimum amount of arguments to the command.
func ValidateMinArgs(n int) ValidateArgsFunc {
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

// ValidateArgsFuncs returns a [ValidateArgsFunc] that runs the provided validators in order and returns the first error
// encountered, short-circuiting on failure. Nil entries in fns are skipped. This is useful for composing several
// smaller, focused validators into a single one with logical AND semantics.
func ValidateArgsFuncs(fns ...ValidateArgsFunc) ValidateArgsFunc {
	return func(ctx context.Context, args *Arguments) error {
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			if err := fn(ctx, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// ValidateFlagsFuncs returns a [ValidateFlagsFunc] that runs the provided validators in order and returns the first
// error encountered, short-circuiting on failure. Nil entries in fns are skipped. This is useful for composing several
// smaller, focused validators into a single one with logical AND semantics.
func ValidateFlagsFuncs(fns ...ValidateFlagsFunc) ValidateFlagsFunc {
	return func(ctx context.Context) error {
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			if err := fn(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}
