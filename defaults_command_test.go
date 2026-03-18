package naistrix_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	if got, err := runCommand(configPath, "defaults list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if contains := "is empty, or it does not yet exist"; !strings.Contains(got, contains) {
		t.Fatalf("expected output to contain %q, got %q", contains, got)
	}

	if got, err := runCommand(configPath, "defaults set expected_key expected_value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if contains := "Set expected_key = expected_value"; !strings.Contains(got, contains) {
		t.Fatalf("expected output to contain %q, got %q", contains, got)
	}

	if got, err := runCommand(configPath, "defaults get expected_key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if contains := "expected_key = expected_value"; !strings.Contains(got, contains) {
		t.Fatalf("expected output to contain %q, got %q", contains, got)
	}

	if got, err := runCommand(configPath, "defaults unset expected_key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if contains := "Unset expected_key (value: expected_value)"; !strings.Contains(got, contains) {
		t.Fatalf("expected output to contain %q, got %q", contains, got)
	}

	if got, err := runCommand(configPath, "defaults get expected_key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if contains := "No such configuration key: expected_key"; !strings.Contains(got, contains) {
		t.Fatalf("expected output to contain %q, got %q", contains, got)
	}
}

func runCommand(configPath, args string) (string, error) {
	argSlice := []string{"--no-colors", "--config", configPath}
	argSlice = append(argSlice, strings.Split(args, " ")...)

	var outputBuffer bytes.Buffer
	app, _, err := naistrix.NewApplication(
		"test",
		"test application",
		"v0.6.9",
		naistrix.ApplicationWithWriter(&outputBuffer),
	)
	if err != nil {
		return "", err
	}

	err = app.Run(naistrix.RunWithArgs(argSlice))
	return outputBuffer.String(), err
}
