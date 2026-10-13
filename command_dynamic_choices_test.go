package naistrix_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

func TestDynamicChoicesFromEarlierArguments(t *testing.T) {
	for _, tt := range []struct {
		name          string
		values        []string
		wantError     string
		wantLookups   int
		dynamicTeam   bool
		caseSensitive bool
	}{
		{name: "canonical earlier argument", values: []string{"DEV", "alice"}, wantLookups: 1},
		{name: "earlier dynamic argument", values: []string{"DEV", "alice"}, wantLookups: 1, dynamicTeam: true},
		{name: "invalid earlier argument", values: []string{"unknown", "alice"}, wantError: "invalid value"},
		{name: "invalid member", values: []string{"dev", "unknown"}, wantError: "invalid value", wantLookups: 1},
		{name: "case-sensitive dynamic choices", values: []string{"dev", "alice"}, caseSensitive: true, wantError: "invalid value", wantLookups: 1},
		{name: "missing arguments", values: []string{"dev"}, wantError: "Expected exactly 2 arguments"},
		{name: "extra arguments", values: []string{"dev", "alice", "extra"}, wantError: "Expected exactly 2 arguments"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			team := naistrix.Argument{Name: "team", Choices: []string{"Dev", "Ops"}, ChoicesCaseInsensitive: true}
			if tt.dynamicTeam {
				team.Choices = nil
				team.ChoicesFunc = func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
					if args.Len() != 0 {
						t.Errorf("first argument callback received %v", args.All())
					}
					return []string{"Dev", "Ops"}, nil
				}
			}
			lookups, validations, runs := 0, 0, 0

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			cmd := &naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{team, {
					Name: "member", ChoicesCaseInsensitive: !tt.caseSensitive,
					ChoicesFunc: func(gotCtx context.Context, args *naistrix.Arguments) ([]string, error) {
						lookups++
						if gotCtx != ctx || args.Len() != 1 || !slices.Equal(args.All(), []string{"Dev"}) || args.Get("team") != "Dev" {
							t.Errorf("unexpected callback context or prefix: %v", args.All())
						}
						return []string{"Alice", "Bob"}, nil
					},
				}},
				ValidateFunc: func(_ context.Context, args *naistrix.Arguments) error {
					validations++
					if !slices.Equal(args.All(), []string{"Dev", "Alice"}) {
						t.Errorf("custom validation saw %v", args.All())
					}
					return nil
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					runs++
					if !slices.Equal(args.All(), []string{"Dev", "Alice"}) {
						t.Errorf("RunFunc saw %v", args.All())
					}
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")}, tt.values...)
			original := slices.Clone(args)
			err = app.Run(naistrix.RunWithArgs(args), naistrix.RunWithContext(ctx))
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) || validations != 0 || runs != 0 {
					t.Fatalf("expected %q without custom validation or execution, got %v, validations %d, runs %d", tt.wantError, err, validations, runs)
				}
			} else if err != nil || validations != 1 || runs != 1 {
				t.Fatalf("error %v, validations %d, runs %d", err, validations, runs)
			}
			if lookups != tt.wantLookups || cmd.Args[1].Choices != nil || !slices.Equal(args, original) {
				t.Errorf("lookups %d (want %d), definitions or supplied values changed", lookups, tt.wantLookups)
			}
		})
	}
}

func TestDynamicChoicesErrors(t *testing.T) {
	lookupError := errors.New("service unavailable")
	for _, tt := range []struct {
		name        string
		choices     []string
		insensitive bool
		lookupError error
		want        string
	}{
		{name: "nil choices", want: "no choices available"},
		{name: "empty choices", choices: []string{}, want: "no choices available"},
		{name: "duplicate choices", choices: []string{"Alice", "Alice"}, want: "duplicate choices"},
		{name: "ambiguous choices", choices: []string{"Alice", "ALICE"}, insensitive: true, want: "duplicate choices"},
		{name: "ambiguous unicode", choices: []string{"S", "ſ"}, insensitive: true, want: "duplicate choices"},
		{name: "lookup error", lookupError: lookupError, want: "failed to resolve choices"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			if err := app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{{Name: "member", ChoicesCaseInsensitive: tt.insensitive, ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
					return tt.choices, tt.lookupError
				}}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					t.Error("RunFunc executed after lookup failure")
					return nil
				},
			}); err != nil {
				t.Fatalf("provider should not be evaluated during registration: %v", err)
			}
			err = app.Run(naistrix.RunWithArgs([]string{"test", "Alice", "--config", filepath.Join(t.TempDir(), "config.yaml")}))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if tt.lookupError != nil && !errors.Is(err, tt.lookupError) {
				t.Errorf("lookup error not preserved: %v", err)
			}
		})
	}
}

func TestStaticAndDynamicChoicesAreMutuallyExclusive(t *testing.T) {
	for _, choices := range [][]string{{"Alice"}, {}} {
		app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0")
		if err != nil {
			t.Fatal(err)
		}
		err = app.AddCommand(&naistrix.Command{
			Name: "test", Title: "Test",
			Args: []naistrix.Argument{{Name: "member", Choices: choices, ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
				t.Error("provider called during registration")
				return nil, nil
			}}},
			RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error { return nil },
		})
		if err == nil || !strings.Contains(err.Error(), "both Choices and ChoicesFunc") {
			t.Fatalf("expected mutually exclusive choices error, got %v", err)
		}
	}
}

func TestDynamicChoicesFromFlags(t *testing.T) {
	for _, source := range []string{"CLI", "config", "environment", "global", "sticky"} {
		t.Run(source, func(t *testing.T) {
			flags := &struct{ Team string }{}
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			args := []string{"delete-member", "Alice", "--config", configPath}
			switch source {
			case "config":
				if err := os.WriteFile(configPath, []byte("team: Dev\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "environment":
				t.Setenv("TEST_TEAM", "Dev")
			default:
				args = append(args, "--team", "Dev")
			}
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			lookups := 0
			called := false
			cmd := &naistrix.Command{
				Name: "delete-member", Title: "Delete a member", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(_ context.Context, prefix *naistrix.Arguments) ([]string, error) {
					lookups++
					if flags.Team != "Dev" || prefix.Len() != 0 {
						t.Errorf("unsynchronized team %q, prefix %v", flags.Team, prefix.All())
					}
					return []string{"Alice"}, nil
				}}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error { called = true; return nil },
			}
			parent := cmd
			switch source {
			case "global":
				if err := app.AddGlobalFlags(flags); err != nil {
					t.Fatal(err)
				}
			case "sticky":
				parent = &naistrix.Command{Name: "team", Title: "Team", StickyFlags: flags, SubCommands: []*naistrix.Command{cmd}}
				args = append([]string{"team"}, args...)
			default:
				cmd.Flags = flags
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(naistrix.RunWithArgs(args)); err != nil {
				t.Fatal(err)
			}
			if lookups != 1 || !called {
				t.Errorf("lookups %d, called %v", lookups, called)
			}
		})
	}
}

func TestRepeatableDynamicChoices(t *testing.T) {
	for _, mode := range []string{"supplied", "prompted"} {
		t.Run(mode, func(t *testing.T) {
			sharedChoices := []string{"Alice", "Bob"}
			lookups, selections, confirmations := 0, 0, 0
			if mode == "prompted" {
				restore := naistrix.SetChoicePromptForTest(func(prompt string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
					selections++
					if selections == 1 {
						if !strings.HasPrefix(prompt, "#1 TEAM:") || !slices.Equal(choices, []string{"Dev"}) {
							t.Fatalf("unexpected team prompt %q, choices %v", prompt, choices)
						}
						return "Dev", nil
					}
					if !strings.HasPrefix(prompt, "#2 MEMBER:") || !slices.Equal(choices, []string{"Alice", "Bob"}) || selections > 3 {
						t.Fatalf("unexpected member prompt %q, choices %v", prompt, choices)
					}
					// The returned provider slice must have been copied before the prompt is opened.
					sharedChoices[0] = "changed"
					return choices[selections-2], nil
				})
				defer restore()
				restoreConfirm := naistrix.SetArgumentConfirmationForTest(func(prompt string, _ ...input.ConfirmOptionFunc) (bool, error) {
					confirmations++
					if prompt != "Add another MEMBER?" {
						t.Errorf("unexpected confirmation %q", prompt)
					}
					return confirmations == 1, nil
				})
				defer restoreConfirm()
			}
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			cmd := &naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: mode == "supplied",
				Args: []naistrix.Argument{
					{Name: "team", Choices: []string{"Dev"}, ChoicesCaseInsensitive: true},
					{Name: "member", Repeatable: true, ChoicesCaseInsensitive: true, ChoicesFunc: func(_ context.Context, prefix *naistrix.Arguments) ([]string, error) {
						lookups++
						if !slices.Equal(prefix.All(), []string{"Dev"}) {
							t.Errorf("unresolved prefix %v", prefix.All())
						}
						return sharedChoices, nil
					}},
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					got = args.All()
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")}
			if mode == "supplied" {
				args = append(args, "DEV", "alice", "BOB")
			}
			if err := app.Run(naistrix.RunWithArgs(args)); err != nil {
				t.Fatal(err)
			}
			if lookups != 1 || !slices.Equal(got, []string{"Dev", "Alice", "Bob"}) || cmd.Args[1].Choices != nil {
				t.Errorf("lookups %d, values %v, retained choices %v", lookups, got, cmd.Args[1].Choices)
			}
			if mode == "prompted" && (selections != 3 || confirmations != 2) {
				t.Errorf("selections %d, confirmations %d", selections, confirmations)
			}
		})
	}
}

func TestDynamicChoicesResolvedForEachExecution(t *testing.T) {
	app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	lookups := 0
	var got []string
	cmd := &naistrix.Command{
		Name: "test", Title: "Test", DisableArgumentPrompts: true,
		Args: []naistrix.Argument{
			{Name: "team", Choices: []string{"Dev", "Ops"}},
			{Name: "member", ChoicesFunc: func(_ context.Context, prefix *naistrix.Arguments) ([]string, error) {
				lookups++
				return map[string][]string{"Dev": {"Alice"}, "Ops": {"Bob"}}[prefix.Get("team")], nil
			}},
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
			got = args.All()
			return nil
		},
	}
	if err := app.AddCommand(cmd); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	for _, values := range [][]string{{"Dev", "Alice"}, {"Ops", "Bob"}} {
		args := append([]string{"test", "--config", configPath}, values...)
		if err := app.Run(naistrix.RunWithArgs(args)); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, values) || cmd.Args[1].Choices != nil {
			t.Errorf("got %v, wanted %v, retained choices %v", got, values, cmd.Args[1].Choices)
		}
	}
	if lookups != 2 {
		t.Errorf("got %d lookups, want 2", lookups)
	}
}

func TestDynamicChoicesCancellation(t *testing.T) {
	for _, when := range []string{"before lookup", "during lookup", "callback returns after cancellation"} {
		t.Run(when, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if when == "before lookup" {
				cancel()
			}
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			lookups := 0
			if err := app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(ctx context.Context, _ *naistrix.Arguments) ([]string, error) {
					lookups++
					cancel()
					if when == "callback returns after cancellation" {
						return []string{"Alice"}, nil
					}
					return nil, ctx.Err()
				}}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					t.Error("RunFunc executed after cancellation")
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(naistrix.RunWithArgs([]string{"test", "Alice", "--config", filepath.Join(t.TempDir(), "config.yaml")}), naistrix.RunWithContext(ctx))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}
			wantLookups := 0
			if when != "before lookup" {
				wantLookups = 1
			}
			if lookups != wantLookups {
				t.Errorf("got %d lookups, want %d", lookups, wantLookups)
			}
		})
	}
}
