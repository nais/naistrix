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
