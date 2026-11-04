package naistrix_test

import (
	"bytes"
	"context"
	"errors"
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
					app, stdout, _ = newCompletionApplication(t)
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

func TestDefaultsDocumentation(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		for _, mode := range []string{"list", "empty", "unknown key", "get", "unset"} {
			t.Run(commandName+"/"+mode, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				var stdout bytes.Buffer
				opts := []naistrix.ApplicationOptionFunc{
					naistrix.ApplicationWithWriter(&stdout),
					naistrix.ApplicationWithDefaultsCommandName(commandName),
				}
				if mode != "empty" {
					opts = append(opts,
						naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{Key: "zeta", Title: "<info>Last\nsetting.</info>", Description: "Long documentation.\n\nAnother paragraph."}),
						naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{Key: "alpha", Title: "First setting."}),
					)
				}
				app, _, err := naistrix.NewApplication("test", "Test defaults documentation", "v0.0.0", opts...)
				if err != nil {
					t.Fatal(err)
				}
				addDefaultsTestFlags(t, app)
				args := []string{"--no-colors", "--config", path, commandName}
				switch mode {
				case "unknown key":
					args = append(args, "set", "undocumented", "value")
				case "get", "unset":
					args = append(args, mode, "alpha")
				default:
					args = append(args, "list")
				}
				err = app.Run(naistrix.RunWithArgs(args))
				if mode == "get" || mode == "unset" {
					if err == nil || !strings.Contains(err.Error(), "no choices available") {
						t.Fatalf("documented-only key should not be selectable: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				got := stdout.String()
				switch mode {
				case "list":
					if strings.Contains(got, "Long documentation") || strings.Contains(got, "Another paragraph") {
						t.Errorf("list should not show detailed descriptions: %q", got)
					}
					if strings.Contains(got, "<info>") || strings.Contains(got, "\x1b[") {
						t.Errorf("no-colors output should strip formatting: %q", got)
					}
					for _, want := range []string{"Configured value", "Description", "(not set)", "First setting.", "Last setting.", "Other keys may also be set"} {
						if !strings.Contains(got, want) {
							t.Errorf("missing %q in %q", want, got)
						}
					}
					if strings.Index(got, "alpha") > strings.Index(got, "zeta") {
						t.Errorf("keys not sorted: %q", got)
					}
				case "empty":
					if !strings.Contains(got, "is empty, or it does not yet exist") {
						t.Errorf("unexpected output: %q", got)
					}
				case "unknown key":
					contents, err := os.ReadFile(path)
					if err != nil || !strings.Contains(string(contents), "undocumented: value") {
						t.Fatalf("undocumented key was not persisted: %q, %v", contents, err)
					}
					return
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("documentation should not create a configuration file: %v", err)
				}
			})
		}
	}
}

func TestDefaultsListConfiguredValuesAndDocumentation(t *testing.T) {
	for _, mode := range []string{"documented", "no documentation", "empty file"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			contents := "team: my-team\ncustom: some-value\nempty: \"\"\n"
			if mode == "empty file" {
				contents = "{}\n"
			}
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			opts := []naistrix.ApplicationOptionFunc{naistrix.ApplicationWithWriter(&stdout)}
			if mode != "no documentation" {
				opts = append(opts, naistrix.ApplicationWithDefaultsDocumentation(
					naistrix.DefaultDefinition{Key: "TEAM", Title: "Team to use."},
					naistrix.DefaultDefinition{Key: "output", Title: "Output format."},
				))
			}
			app, _, err := naistrix.NewApplication("test", "Test defaults list", "v0.0.0", opts...)
			if err != nil {
				t.Fatal(err)
			}
			addDefaultsTestFlags(t, app)
			// Environment overrides must not be presented as configured values.
			t.Setenv("TEST_TEAM", "environment-team")
			if err := app.Run(naistrix.RunWithArgs([]string{"--no-colors", "--config", path, "defaults", "list"})); err != nil {
				t.Fatal(err)
			}
			got := stdout.String()
			if strings.Contains(got, "environment-team") {
				t.Errorf("list included an environment override: %q", got)
			}
			var rows [][]string
			for line := range strings.SplitSeq(got, "\n") {
				if !strings.Contains(line, "│") {
					continue
				}
				columns := strings.Split(line, "│")
				for i := range columns {
					columns[i] = strings.TrimSpace(columns[i])
				}
				rows = append(rows, columns)
			}
			want := [][]string{
				{"Key", "Configured value", "Description"},
				{"custom", "some-value", "—"},
				{"empty", `""`, "—"},
				{"output", "(not set)", "Output format."},
				{"team", "my-team", "Team to use."},
			}
			switch mode {
			case "no documentation":
				want = [][]string{{"Key", "Configured value"}, {"custom", "some-value"}, {"empty", `""`}, {"team", "my-team"}}
			case "empty file":
				want = [][]string{{"Key", "Configured value", "Description"}, {"output", "(not set)", "Output format."}, {"team", "(not set)", "Team to use."}}
			}
			if !slices.EqualFunc(rows, want, slices.Equal[[]string]) {
				t.Errorf("unexpected table rows:\ngot  %v\nwant %v\noutput: %s", rows, want, got)
			}
			saved, err := os.ReadFile(path)
			if err != nil || string(saved) != contents {
				t.Errorf("list modified configuration: %q, %v", saved, err)
			}
		})
	}
}

func TestDefaultsDocumentationValidation(t *testing.T) {
	for _, definitions := range [][]naistrix.DefaultDefinition{
		{{Title: "Missing key."}},
		{{Key: "two words", Title: "Invalid key."}},
		{{Key: "two\u00a0words", Title: "Invalid key."}},
		{{Key: "team"}},
		{{Key: "team", Title: " \n\t"}},
		{{Key: "team", Description: "A description does not replace the required title."}},
		{{Key: "team", Title: "One."}, {Key: "TEAM", Title: "Two."}},
	} {
		if _, _, err := naistrix.NewApplication("test", "Test defaults documentation", "v0.0.0", naistrix.ApplicationWithDefaultsDocumentation(definitions...)); err == nil {
			t.Errorf("expected invalid definitions to fail: %+v", definitions)
		}
	}
}

func TestDefaultsSetDocumentedKeyCompletion(t *testing.T) {
	for _, mode := range []string{"union", "prefix", "missing file", "empty file", "value", "no documentation"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			contents := "alpha: first\ncustom: existing\n"
			if mode == "empty file" {
				contents = "{}\n"
			}
			if mode != "missing file" {
				if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var stdout bytes.Buffer
			opts := []naistrix.ApplicationOptionFunc{naistrix.ApplicationWithWriter(&stdout)}
			if mode != "no documentation" {
				opts = append(opts, naistrix.ApplicationWithDefaultsDocumentation(
					naistrix.DefaultDefinition{Key: "zeta", Title: "<warn>Last\nsetting.</warn>", Description: "Detailed documentation.\n\nNot for completion."},
					naistrix.DefaultDefinition{Key: "ALPHA", Title: "<info>First</info> <error>setting.</error>"},
				))
			}
			app, _, err := naistrix.NewApplication("test", "Test defaults completion", "v0.0.0", opts...)
			if err != nil {
				t.Fatal(err)
			}
			addDefaultsTestFlags(t, app)
			args := []string{"defaults", "set", "--config", path}
			want := []string{"alpha\tFirst setting.", "custom\tCurrently configured; no documentation available.", "zeta\tLast setting."}
			switch mode {
			case "prefix":
				args = append(args, "a")
				want = want[:1]
			case "value":
				args = append(args, "alpha", "")
				want = nil
			default:
				args = append(args, "")
			}
			if mode == "missing file" || mode == "empty file" {
				want = []string{want[0], want[2]}
			}
			if mode == "no documentation" {
				want = []string{"alpha\tCurrently configured; no documentation available.", want[1]}
			}
			if mode != "value" {
				want = cobra.AppendActiveHelp(want, "Choose a suggested key or enter another key")
			}
			got, directive := requestCompletion(t, app, &stdout, args...)
			if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("got %v (%v), want %v (%v)", got, directive, want, cobra.ShellCompDirectiveNoFileComp)
			}
		})
	}
}

func TestDefaultsDescribe(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		for _, mode := range []string{"stored", "not set", "empty value", "title only", "prompt", "unknown key", "no documentation", "malformed file"} {
			t.Run(commandName+"/"+mode, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				contents := "team: my-team\ncustom: undocumented\n"
				switch mode {
				case "empty value":
					contents = "team: \"\"\n"
				case "malformed file":
					contents = "team: [invalid yaml\n"
				}
				if mode != "not set" {
					if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				var stdout bytes.Buffer
				opts := []naistrix.ApplicationOptionFunc{
					naistrix.ApplicationWithWriter(&stdout),
					naistrix.ApplicationWithDefaultsCommandName(commandName),
				}
				if mode != "no documentation" {
					description := "Used for <info>team flags</info>.\n\nAn explicit flag overrides this setting."
					if mode == "title only" {
						description = ""
					}
					opts = append(opts, naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{
						Key: "TEAM", Title: "<warn>Default\nteam.</warn>", Description: description,
					}))
				}
				app, _, err := naistrix.NewApplication("test", "Test defaults describe", "v0.0.0", opts...)
				if err != nil {
					t.Fatal(err)
				}
				addDefaultsTestFlags(t, app)
				t.Setenv("TEST_TEAM", "environment-team")
				key := "TeAm"
				if mode == "unknown key" {
					key = "custom"
				}
				args := []string{"--no-colors", "--config", path, commandName, "describe"}
				if mode == "prompt" {
					restore := naistrix.SetChoicePromptForTest(func(prompt string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
						if prompt != "#1 KEY: Select a documented configuration key" || !slices.Equal(choices, []string{"team"}) {
							t.Errorf("unexpected prompt %q or choices %v", prompt, choices)
						}
						return "team", nil
					})
					defer restore()
				} else {
					args = append(args, key)
				}
				err = app.Run(naistrix.RunWithArgs(args))
				if mode == "unknown key" || mode == "no documentation" || mode == "malformed file" {
					if err == nil {
						t.Fatal("expected describe to fail")
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					value := "my-team"
					switch mode {
					case "not set":
						value = "(not set)"
					case "empty value":
						value = `""`
					}
					want := "Configuration key: team\nConfigured value: " + value + "\n\nDefault team.\n"
					if mode != "title only" {
						want += "\nUsed for team flags.\n\nAn explicit flag overrides this setting.\n"
					}
					if got := stdout.String(); got != want {
						t.Errorf("unexpected description:\ngot  %q\nwant %q", got, want)
					}
				}
				if mode == "not set" {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("describe created a configuration file: %v", err)
					}
				} else if saved, err := os.ReadFile(path); err != nil || string(saved) != contents {
					t.Errorf("describe modified configuration: %q, %v", saved, err)
				}
			})
		}
	}
}

func TestDefaultsDescribeCompletion(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		for _, mode := range []string{"all", "prefix", "no documentation", "value"} {
			t.Run(commandName+"/"+mode, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				// Persisted, undocumented keys must not be suggested by describe.
				if err := os.WriteFile(path, []byte("custom: undocumented\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				var stdout bytes.Buffer
				opts := []naistrix.ApplicationOptionFunc{
					naistrix.ApplicationWithWriter(&stdout),
					naistrix.ApplicationWithDefaultsCommandName(commandName),
				}
				if mode != "no documentation" {
					opts = append(opts, naistrix.ApplicationWithDefaultsDocumentation(
						naistrix.DefaultDefinition{Key: "zeta", Title: "<warn>Last\nsetting.</warn>", Description: "Not for completion."},
						naistrix.DefaultDefinition{Key: "ALPHA", Title: "First setting."},
					))
				}
				app, _, err := naistrix.NewApplication("test", "Test defaults describe completion", "v0.0.0", opts...)
				if err != nil {
					t.Fatal(err)
				}
				addDefaultsTestFlags(t, app)
				args := []string{commandName, "describe", "--config", path}
				want := []string{"alpha\tFirst setting.", "zeta\tLast setting."}
				switch mode {
				case "prefix":
					args = append(args, "A")
					want = want[:1]
				case "value":
					args = append(args, "alpha", "")
					want = nil
				default:
					args = append(args, "")
				}
				if mode == "no documentation" {
					want = nil
				}
				if mode != "value" {
					want = cobra.AppendActiveHelp(want, "Choose a documented configuration key")
				}
				got, directive := requestCompletion(t, app, &stdout, args...)
				if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp {
					t.Errorf("got %v (%v), want %v (%v)", got, directive, want, cobra.ShellCompDirectiveNoFileComp)
				}
			})
		}
	}
}

func addDefaultsTestFlags(t *testing.T, app *naistrix.Application) {
	t.Helper()
	if err := app.AddGlobalFlags(&struct {
		Alpha  string
		Zeta   string
		Team   string
		Output string
	}{}); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultsDocumentationMatchesRegisteredFlags(t *testing.T) {
	var stdout bytes.Buffer
	app, _, err := naistrix.NewApplication("test", "Test documented flags", "v0.0.0",
		naistrix.ApplicationWithWriter(&stdout),
		naistrix.ApplicationWithDefaultsDocumentation(
			naistrix.DefaultDefinition{Key: "NO-COLORS", Title: "Built-in global flag."},
			naistrix.DefaultDefinition{Key: "TEAM-NAME", Title: "Additional global flag with a name tag."},
			naistrix.DefaultDefinition{Key: "local", Title: "Nested command-local flag."},
			naistrix.DefaultDefinition{Key: "sticky", Title: "Parent sticky flag."},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddGlobalFlags(&struct {
		Team         string `name:"team-name" short:"t"`
		Undocumented string
	}{}); err != nil {
		t.Fatal(err)
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "parent",
		Title: "Parent command",
		StickyFlags: &struct {
			Sticky string
		}{},
		SubCommands: []*naistrix.Command{{
			Name:  "child",
			Title: "Nested command",
			Flags: &struct {
				Local string
			}{},
			RunFunc: completionMustNotRun(t),
		}},
	}); err != nil {
		t.Fatal(err)
	}
	// The selected command need not have the documented flags itself.
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := app.Run(naistrix.RunWithArgs([]string{"--config", path, "defaults", "list"})); err != nil {
		t.Fatalf("expected documented flags registered after NewApplication to be accepted: %v", err)
	}
}

func TestDefaultsDocumentationRejectsUnregisteredFlags(t *testing.T) {
	for _, key := range []string{"tem", "Team", "t"} {
		for _, args := range [][]string{
			{"defaults", "list"},
			{"defaults", "set", "tem", "new-value"},
			{"run"},
			{"--help"},
			{"--version"},
			{cobra.ShellCompRequestCmd, "defaults", "set", ""},
		} {
			t.Run(key+"/"+strings.Join(args, " "), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.yaml")
				contents := "tem: persisted-value\n"
				if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("TEST_TEM", "environment-value")
				var stdout bytes.Buffer
				app, _, err := naistrix.NewApplication("test", "Test documented flags", "v0.0.0",
					naistrix.ApplicationWithWriter(&stdout),
					naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{Key: key, Title: "Invalid documented key."}),
					naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
						Name: "must-not-run",
						RunFunc: func(context.Context, *naistrix.OutputWriter) error {
							t.Error("hook ran before documentation was validated")
							return nil
						},
					}),
				)
				if err != nil {
					t.Fatalf("flag matching should be deferred until Run: %v", err)
				}
				if err := app.AddGlobalFlags(&struct {
					Team string `name:"team-name" short:"t"`
				}{}); err != nil {
					t.Fatal(err)
				}
				if err := app.AddCommand(&naistrix.Command{Name: "run", Title: "Run command", RunFunc: completionMustNotRun(t)}); err != nil {
					t.Fatal(err)
				}
				err = app.Run(naistrix.RunWithArgs(append([]string{"--config", path}, args...)))
				want := "documented configuration key \"" + key + "\" does not match any registered flag"
				if err == nil || err.Error() != want {
					t.Fatalf("expected %q, got %v", want, err)
				}
				if stdout.Len() != 0 {
					t.Errorf("unexpected output before validation: %q", stdout.String())
				}
				saved, err := os.ReadFile(path)
				if err != nil || string(saved) != contents {
					t.Errorf("invalid documentation modified configuration: %q, %v", saved, err)
				}
				if err := app.Run(); !errors.Is(err, naistrix.ErrApplicationAlreadyRun) {
					t.Errorf("failed validation must still consume the application: %v", err)
				}
			})
		}
	}
}
