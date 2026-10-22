package input_test

import (
	"strings"
	"testing"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/nais/naistrix/input"
	"github.com/pterm/pterm"
)

func TestSecret_ReturnsValueAndMasksOutput(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
	}{
		{name: "text", value: "secret-token"},
		{name: "whitespace", value: "  secret token  "},
		{name: "unicode", value: "hemmelig-秘密"},
		{name: "empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			restore := input.SetInteractive(func() bool { return true })
			defer restore()
			readOutput, restoreOutput := input.CapturePromptOutputForTest(t)
			defer restoreOutput()

			go func() {
				_ = keyboard.SimulateKeyPress(tt.value)
				_ = keyboard.SimulateKeyPress(keys.Enter)
			}()

			got, err := input.Secret("API token")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.value {
				t.Errorf("expected %q, got %q", tt.value, got)
			}
			output := readOutput()
			if !strings.Contains(output, "API token") {
				t.Error("prompt was not displayed")
			}
			if tt.value != "" {
				if strings.Contains(output, tt.value) {
					t.Error("secret was printed in plaintext")
				}
				if !strings.Contains(output, "*") {
					t.Error("secret was not masked")
				}
			}
		})
	}
}

func TestSecret_DoesNotChangeDefaultMask(t *testing.T) {
	restore := input.SetInteractive(func() bool { return true })
	defer restore()
	readOutput, restoreOutput := input.CapturePromptOutputForTest(t)
	defer restoreOutput()
	previousMask := pterm.DefaultInteractiveTextInput.Mask

	go func() {
		_ = keyboard.SimulateKeyPress("secret-token")
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()
	if _, err := input.Secret("Token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pterm.DefaultInteractiveTextInput.Mask != previousMask {
		t.Error("Secret changed the default printer's mask")
	}

	go func() {
		_ = keyboard.SimulateKeyPress("visible-name")
		_ = keyboard.SimulateKeyPress(keys.Enter)
	}()
	if got, err := input.Input("Name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if got != "visible-name" {
		t.Errorf("unexpected value: %q", got)
	}
	if !strings.Contains(readOutput(), "visible-name") {
		t.Error("ordinary input was not displayed after Secret")
	}
}
