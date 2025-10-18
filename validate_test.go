package naistrix

import (
	"context"
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
