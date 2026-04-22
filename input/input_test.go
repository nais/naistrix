// Skip when race detector is enabled: https://github.com/atomicgo/keyboard/issues/6
//go:build !race

package input_test

import (
	"strings"
	"testing"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/nais/naistrix/input"
)

func TestConfirm_Yes(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress('y')
	}()
	result, err := input.Confirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Errorf("expected true when pressing 'y'")
	}
}

func TestConfirm_No(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress('n')
	}()
	result, err := input.Confirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false when pressing 'n'")
	}
}

func TestConfirm_DefaultIsNo(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()
	result, err := input.Confirm("Are you sure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Errorf("expected false (default) when pressing Enter")
	}
}

func TestInput_ReturnsValue(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress("hi")
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	result, err := input.Input("Enter something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hi" {
		t.Errorf("expected %q, got %q", "hi", result)
	}
}

func TestInput_DefaultValueOnEnter(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()
	result, err := input.Input("Enter something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSelect_ReturnsSelectedOption(t *testing.T) {
	go func() {
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()

	if result, err := input.Select("Pick one", []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if result == "" {
		t.Error("expected a non-empty selection")
	}
}

func TestSelect_NoOptions(t *testing.T) {
	if _, err := input.Select("Pick one", []string{}); err == nil {
		t.Error("expected error for empty options, got nil")
	} else if contains := "no options provided"; !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
	}
}

func TestSelect_DuplicateOptions(t *testing.T) {
	if _, err := input.Select("Pick one", []string{"alpha", "beta", "alpha"}); err == nil {
		t.Error("expected error for duplicate options, got nil")
	} else if contains := "duplicate label: alpha (index 2)"; !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
	}
}
