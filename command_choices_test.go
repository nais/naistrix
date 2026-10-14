package naistrix_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

func TestCaseInsensitiveArgumentChoices(t *testing.T) {
	for _, tt := range []struct {
		name        string
		choices     []string
		insensitive bool
		repeatable  bool
		values      []string
		want        []string
		wantError   bool
	}{
		{name: "exact match by default", choices: []string{"foo", "bar"}, values: []string{"foo"}, want: []string{"foo"}},
		{name: "case sensitive by default", choices: []string{"foo", "bar"}, values: []string{"FOO"}, wantError: true},
		{name: "uppercase input", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"FOO"}, want: []string{"foo"}},
		{name: "mixed case input", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"bAr"}, want: []string{"bar"}},
		{name: "configured spelling", choices: []string{"Foo", "BAR"}, insensitive: true, values: []string{"foo"}, want: []string{"Foo"}},
		{name: "unicode folding", choices: []string{"S"}, insensitive: true, values: []string{"ſ"}, want: []string{"S"}},
		{name: "empty choice", choices: []string{""}, insensitive: true, values: []string{""}, want: []string{""}},
		{name: "no choices", insensitive: true, values: []string{"MiXeD"}, want: []string{"MiXeD"}},
		{name: "whitespace is not trimmed", choices: []string{"foo"}, insensitive: true, values: []string{" foo "}, wantError: true},
		{name: "configured whitespace is preserved", choices: []string{" Foo "}, insensitive: true, values: []string{" fOO "}, want: []string{" Foo "}},
		{name: "invalid choice", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"BAZ"}, wantError: true},
		{name: "repeatable normalization", choices: []string{"Foo", "Bar"}, insensitive: true, repeatable: true, values: []string{"FOO", "bar", "fOo"}, want: []string{"Foo", "Bar", "Foo"}},
		{name: "repeatable no choices", insensitive: true, repeatable: true, values: []string{"FOO", "bar"}, want: []string{"FOO", "bar"}},
		{name: "repeatable remains case sensitive", choices: []string{"foo", "bar"}, repeatable: true, values: []string{"foo", "BAR"}, wantError: true},
		{name: "invalid repeatable choice", choices: []string{"foo", "bar"}, insensitive: true, repeatable: true, values: []string{"FOO", "BAZ"}, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			validated, called := false, false
			check := func(args *naistrix.Arguments) {
				t.Helper()
				if tt.wantError {
					t.Error("invalid choices reached custom validation or execution")
					return
				}
				if got := args.All(); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("All() = %v, want %v", got, tt.want)
				}
				if tt.repeatable {
					if got := args.GetRepeatable("value"); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("GetRepeatable() = %v, want %v", got, tt.want)
					}
				} else if got := args.Get("value"); got != tt.want[0] {
					t.Errorf("Get() = %q, want %q", got, tt.want[0])
				}
			}
			choices := slices.Clone(tt.choices)
			if err := app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{{Name: "value", Choices: choices, ChoicesCaseInsensitive: tt.insensitive, Repeatable: tt.repeatable}},
				ValidateFunc: func(_ context.Context, args *naistrix.Arguments) error {
					validated = true
					check(args)
					return nil
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					called = true
					check(args)
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")}, tt.values...)
			original := slices.Clone(args)
			err = app.Run(naistrix.RunWithArgs(args))
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "invalid value") || validated || called {
					t.Fatalf("expected invalid choice before validation or execution, got %v, validated %v, called %v", err, validated, called)
				}
			} else if err != nil || !validated || !called {
				t.Fatalf("error %v, validated %v, called %v", err, validated, called)
			}
			if !slices.Equal(args, original) || !slices.Equal(choices, tt.choices) {
				t.Error("normalization changed caller-supplied arguments or configured choices")
			}
		})
	}
}

func TestAmbiguousArgumentChoices(t *testing.T) {
	for _, tt := range []struct {
		name        string
		choices     []string
		insensitive bool
		wantError   bool
	}{
		{name: "distinct casing allowed by default", choices: []string{"foo", "FOO"}},
		{name: "distinct insensitive choices", choices: []string{"foo", "BAR"}, insensitive: true},
		{name: "ambiguous casing", choices: []string{"foo", "FOO"}, insensitive: true, wantError: true},
		{name: "ambiguous unicode", choices: []string{"S", "ſ"}, insensitive: true, wantError: true},
		{name: "exact duplicate", choices: []string{"foo", "foo"}, insensitive: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			err = app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test",
				Args:    []naistrix.Argument{{Name: "value", Choices: tt.choices, ChoicesCaseInsensitive: tt.insensitive}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error { return nil },
			})
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "duplicate choices") {
					t.Fatalf("expected duplicate choices error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCaseInsensitiveChoicePrompt(t *testing.T) {
	choices := []string{"Upper", "Lower"}
	selections := 0
	restore := naistrix.SetChoicePromptForTest(func(_ string, got []string, _ ...input.SelectOptionFunc) (string, error) {
		selections++
		if !slices.Equal(got, choices) {
			t.Errorf("displayed choices %v, want %v", got, choices)
		}
		return got[0], nil
	})
	defer restore()
	app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	validated, called := false, false
	if err := app.AddCommand(&naistrix.Command{
		Name: "test", Title: "Test",
		Args: []naistrix.Argument{{Name: "value", Choices: choices, ChoicesCaseInsensitive: true}},
		ValidateFunc: func(_ context.Context, args *naistrix.Arguments) error {
			validated = true
			if got := args.Get("value"); got != "Upper" {
				t.Errorf("validated value %q, want Upper", got)
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
			called = true
			if got := args.Get("value"); got != "Upper" {
				t.Errorf("executed value %q, want Upper", got)
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")})); err != nil {
		t.Fatal(err)
	}
	if selections != 1 || !validated || !called {
		t.Errorf("selections %d, validated %v, called %v", selections, validated, called)
	}
}

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
