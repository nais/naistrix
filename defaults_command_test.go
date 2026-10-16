package naistrix_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
	"github.com/spf13/cobra"
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

	if _, err := runCommand(configPath, "defaults get expected_key"); err == nil || !strings.Contains(err.Error(), "no choices available") {
		t.Fatalf("expected an error for an empty configuration, got %v", err)
	}
}

func TestDefaultsSetCreatesConfigDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "nested", "config.yaml")
	if _, err := runCommand(path, "defaults set team Dev"); err != nil {
		t.Fatalf("expected missing directories to be created without prompting: %v", err)
	}
	got, err := runCommand(path, "defaults get team")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "team = Dev") {
		t.Errorf("configuration was not saved: %q", got)
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

func TestDefaultsKeyValidation(t *testing.T) {
	for _, command := range []string{"get", "unset"} {
		for _, mode := range []string{"known keys", "unknown key", "mixed keys", "empty file", "missing file"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				contents := "alpha: first\nzeta: last\n"
				if mode == "empty file" {
					contents = "{}\n"
				}
				if mode != "missing file" {
					if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				keys := "alpha zeta"
				switch mode {
				case "unknown key":
					keys = "unknown"
				case "mixed keys":
					keys = "alpha unknown"
				}
				_, err := runCommand(path, "defaults "+command+" "+keys)
				if mode == "known keys" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				want := "invalid value"
				if mode == "empty file" || mode == "missing file" {
					want = "no choices available"
				}
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q error, got %v", want, err)
				}
				if mode != "missing file" {
					got, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if string(got) != contents {
						t.Errorf("failed validation modified the configuration: %q", got)
					}
				}
			})
		}
	}
}

func TestDefaultsInteractiveKeySelection(t *testing.T) {
	for _, command := range []string{"get", "unset"} {
		t.Run(command, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte("zeta: last\nalpha: first\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			selections := 0
			restoreSelect := naistrix.SetChoicePromptForTest(func(prompt string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
				if prompt != "#1 KEY: Select a configuration key" || !slices.Equal(choices, []string{"alpha", "zeta"}) {
					t.Errorf("unexpected prompt %q or choices %v", prompt, choices)
				}
				choice := choices[selections]
				selections++
				return choice, nil
			})
			defer restoreSelect()
			restoreConfirm := naistrix.SetArgumentConfirmationForTest(func(string, ...input.ConfirmOptionFunc) (bool, error) {
				return selections < 2, nil
			})
			defer restoreConfirm()
			got, err := runCommand(path, "defaults "+command)
			if err != nil {
				t.Fatal(err)
			}
			if selections != 2 || !strings.Contains(got, "alpha") || !strings.Contains(got, "zeta") {
				t.Errorf("selections=%d, output=%q", selections, got)
			}
		})
	}
}

func TestDefaultsKeyCompletion(t *testing.T) {
	for _, command := range []string{"get", "unset"} {
		for _, mode := range []string{"all keys", "prefix", "repeatable", "updated configuration", "empty configuration"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				app, stdout, _ := newCompletionApplication(t)
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte("zeta: last\nalpha: first\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args := []string{"defaults", command, "--config", path}
				want := []string{"alpha", "zeta"}
				switch mode {
				case "prefix":
					args = append(args, "a")
					want = []string{"alpha"}
				case "repeatable":
					args = append(args, "alpha", "")
				default:
					args = append(args, "")
				}
				if mode == "updated configuration" || mode == "empty configuration" {
					requestCompletion(t, app, stdout, args...)
					contents := "beta: new\n"
					want = []string{"beta"}
					if mode == "empty configuration" {
						contents = "{}\n"
					}
					if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				got, directive := requestCompletion(t, app, stdout, args...)
				if mode == "empty configuration" {
					if !strings.Contains(strings.Join(got, " "), "no choices available") {
						t.Errorf("expected no choices diagnostic, got %v", got)
					}
				} else if !slices.Equal(got, cobra.AppendActiveHelp(want, "Select a configuration key")) {
					t.Errorf("unexpected completions %v", got)
				}
				if directive != cobra.ShellCompDirectiveNoFileComp {
					t.Errorf("unexpected directive %v", directive)
				}
			})
		}
	}
}
