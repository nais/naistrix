package naistrix

import (
	"context"
	"errors"
	"testing"
)

func TestValidateExactArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("fails with incorrect amount of args", func(t *testing.T) {
		cb := ValidateExactArgs(2)
		args := newArguments([]Argument{{Name: "arg"}}, []string{"arg1"})
		if err := cb(ctx, args); err == nil {
			t.Fatalf("ValidateExactArgs should fail with incorrect amount of args")
		}
	})

	t.Run("passes with correct amount of args", func(t *testing.T) {
		cb := ValidateExactArgs(2)
		args := newArguments([]Argument{{Name: "arg1"}, {Name: "arg2"}}, []string{"arg1", "arg2"})
		if err := cb(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateMinArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("fails with too few args", func(t *testing.T) {
		cb := ValidateMinArgs(2)
		args := newArguments([]Argument{{Name: "arg1"}}, []string{"arg1"})
		if err := cb(ctx, args); err == nil {
			t.Fatalf("ValidateExactArgs should fail with incorrect amount of args")
		}
	})

	t.Run("passes with exact amount of args", func(t *testing.T) {
		cb := ValidateMinArgs(2)
		args := newArguments([]Argument{{Name: "arg1"}, {Name: "arg2"}}, []string{"arg1", "arg2"})
		if err := cb(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("passes with more than min args", func(t *testing.T) {
		cb := ValidateMinArgs(2)
		args := newArguments([]Argument{{Name: "arg1"}, {Name: "arg2"}, {Name: "arg3"}}, []string{"arg1", "arg2", "arg3"})
		if err := cb(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateFuncs(t *testing.T) {
	ctx := context.Background()
	args := newArguments([]Argument{{Name: "arg1"}}, []string{"arg1"})

	pass := func(_ context.Context, _ *Arguments) error { return nil }
	failA := func(_ context.Context, _ *Arguments) error { return errors.New("a failed") }
	failB := func(_ context.Context, _ *Arguments) error { return errors.New("b failed") }

	t.Run("passes with no validators", func(t *testing.T) {
		if err := ValidateFuncs()(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("passes when all validators pass", func(t *testing.T) {
		if err := ValidateFuncs(pass, pass)(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns first error and short-circuits", func(t *testing.T) {
		called := false
		track := func(_ context.Context, _ *Arguments) error {
			called = true
			return nil
		}
		err := ValidateFuncs(pass, failA, track)(ctx, args)
		if err == nil || err.Error() != "a failed" {
			t.Fatalf("expected first error 'a failed', got %v", err)
		}
		if called {
			t.Fatalf("validator after failure should not be called")
		}
	})

	t.Run("skips nil validators", func(t *testing.T) {
		if err := ValidateFuncs(nil, pass, nil)(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err := ValidateFuncs(nil, failB)(ctx, args)
		if err == nil || err.Error() != "b failed" {
			t.Fatalf("expected 'b failed', got %v", err)
		}
	})
}
