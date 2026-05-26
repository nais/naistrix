// Skip when race detector is enabled: https://github.com/atomicgo/keyboard/issues/6
//go:build !race

package input_test

import (
	"testing"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/nais/naistrix/input"
)

func TestInput_ReturnsValue(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress("hi")
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	if result, err := input.Input("Enter something"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if result != "hi" {
		t.Errorf("expected %q, got %q", "hi", result)
	}
}

func TestInput_DefaultValueOnEnter(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	if result, err := input.Input("Enter something"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
