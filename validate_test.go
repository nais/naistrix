package naistrix

import (
	"context"
	"errors"
	"slices"
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

func TestValidateArgsFuncs(t *testing.T) {
	ctx := context.Background()
	args := newArguments([]Argument{{Name: "arg1"}}, []string{"arg1"})

	pass := func(_ context.Context, _ *Arguments) error { return nil }
	failA := func(_ context.Context, _ *Arguments) error { return errors.New("a failed") }
	failB := func(_ context.Context, _ *Arguments) error { return errors.New("b failed") }

	t.Run("passes with no validators", func(t *testing.T) {
		if err := ValidateArgsFuncs()(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("passes when all validators pass", func(t *testing.T) {
		if err := ValidateArgsFuncs(pass, pass)(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns first error and short-circuits", func(t *testing.T) {
		called := false
		track := func(_ context.Context, _ *Arguments) error {
			called = true
			return nil
		}
		err := ValidateArgsFuncs(pass, failA, track)(ctx, args)
		if err == nil || err.Error() != "a failed" {
			t.Fatalf("expected first error 'a failed', got %v", err)
		}
		if called {
			t.Fatalf("validator after failure should not be called")
		}
	})

	t.Run("skips nil validators", func(t *testing.T) {
		if err := ValidateArgsFuncs(nil, pass, nil)(ctx, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err := ValidateArgsFuncs(nil, failB)(ctx, args)
		if err == nil || err.Error() != "b failed" {
			t.Fatalf("expected 'b failed', got %v", err)
		}
	})
}

func TestValidateFlagsFuncs(t *testing.T) {
	for _, tt := range []struct {
		name       string
		validators []int // Zero entries represent nil callbacks; other values identify invocation order.
		failAt     int
		wantCalls  []int
	}{
		{name: "no validators"},
		{name: "only nil validators", validators: []int{0, 0}},
		{name: "all validators pass in order", validators: []int{1, 2, 3}, wantCalls: []int{1, 2, 3}},
		{name: "first validator fails", validators: []int{1, 2, 3}, failAt: 1, wantCalls: []int{1}},
		{name: "later validator fails", validators: []int{1, 2, 3}, failAt: 2, wantCalls: []int{1, 2}},
		{name: "skip nil validators", validators: []int{0, 1, 0, 2, 0}, wantCalls: []int{1, 2}},
		{name: "failure after nil validator", validators: []int{0, 1, 2}, failAt: 1, wantCalls: []int{1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			wantError := errors.New("validation failed")
			var calls []int
			var validators []ValidateFlagsFunc
			for _, id := range tt.validators {
				if id == 0 {
					validators = append(validators, nil)
					continue
				}
				validators = append(validators, func(gotCtx context.Context) error {
					if gotCtx != ctx {
						t.Error("validator received a different context")
					}
					calls = append(calls, id)
					if id == tt.failAt {
						return wantError
					}
					return nil
				})
			}
			err := ValidateFlagsFuncs(validators...)(ctx)
			if tt.failAt != 0 {
				if !errors.Is(err, wantError) {
					t.Errorf("expected original validation error, got %v", err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !slices.Equal(calls, tt.wantCalls) {
				t.Errorf("validator calls %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}
