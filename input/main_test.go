package input_test

import (
	"errors"
	"testing"

	"github.com/nais/naistrix/input"
)

func TestPrompts_NotInteractive(t *testing.T) {
	restore := input.SetInteractive(func() bool { return false })
	defer restore()

	if _, err := input.Input(t.Context(), "prompt"); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Input: expected ErrNotInteractive, got %v", err)
	}
	if _, err := input.Secret(t.Context(), "prompt"); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Secret: expected ErrNotInteractive, got %v", err)
	}
	if _, err := input.Confirm(t.Context(), "prompt"); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Confirm: expected ErrNotInteractive, got %v", err)
	}
	if _, err := input.Select(t.Context(), "prompt", []string{"alpha", "beta"}); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Select: expected ErrNotInteractive, got %v", err)
	}
}

func TestIsInteractive(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "non-interactive", want: false},
		{name: "interactive", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := input.SetInteractive(func() bool { return tt.want })
			defer restore()

			if got := input.IsInteractive(t.Context()); got != tt.want {
				t.Errorf("IsInteractive(ctx) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelect_AutoSelectSingleOptionWithoutTerminal(t *testing.T) {
	restore := input.SetInteractive(func() bool { return false })
	defer restore()

	result, err := input.Select(t.Context(), "prompt", []string{"only"}, input.SelectWithAutoSelectSingleOption())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "only" {
		t.Errorf("expected %q, got %q", "only", result)
	}
}
