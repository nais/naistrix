package input_test

import (
	"testing"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/nais/naistrix/input"
)

func TestConfirm_Yes(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress('y')
	}()

	if result, err := input.Confirm("Are you sure?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !result {
		t.Errorf("expected true when pressing 'y'")
	}
}

func TestConfirm_No(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress('n')
	}()

	if result, err := input.Confirm("Are you sure?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if result {
		t.Errorf("expected false when pressing 'n'")
	}
}

func TestConfirm_DefaultIsNo(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	if result, err := input.Confirm("Are you sure?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if result {
		t.Errorf("expected false (default) when pressing Enter")
	}
}

func TestConfirm_OverrideDefault(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	if result, err := input.Confirm("Are you sure?", input.ConfirmWithDefaultTrue()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !result {
		t.Errorf("expected true when pressing Enter")
	}
}
