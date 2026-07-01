package input_test

import (
	"errors"
	"os"
	"testing"

	"github.com/nais/naistrix/input"
)

// TestMain forces interactivity on for the simulated-keyboard tests in this package, which do not run against a real
// terminal.
func TestMain(m *testing.M) {
	restore := input.SetInteractive(func() bool { return true })
	code := m.Run()
	restore()
	os.Exit(code)
}

func TestPrompts_NotInteractive(t *testing.T) {
	restore := input.SetInteractive(func() bool { return false })
	defer restore()

	if _, err := input.Input("prompt"); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Input: expected ErrNotInteractive, got %v", err)
	}
	if _, err := input.Confirm("prompt"); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Confirm: expected ErrNotInteractive, got %v", err)
	}
	if _, err := input.Select("prompt", []string{"alpha", "beta"}); !errors.Is(err, input.ErrNotInteractive) {
		t.Errorf("Select: expected ErrNotInteractive, got %v", err)
	}
}

func TestSelect_AutoSelectSingleOptionWithoutTerminal(t *testing.T) {
	restore := input.SetInteractive(func() bool { return false })
	defer restore()

	result, err := input.Select("prompt", []string{"only"}, input.SelectWithAutoSelectSingleOption())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "only" {
		t.Errorf("expected %q, got %q", "only", result)
	}
}
