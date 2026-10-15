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
	"github.com/spf13/cobra"
)

func TestFlagValidationBeforeArgumentResolution(t *testing.T) {
	for _, mode := range []string{"supplied", "prompted", "missing flag", "missing flag and argument"} {
		t.Run(mode, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			var events []string
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			missingTeam := errors.New("--team is required")
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", Flags: flags, DisableArgumentPrompts: mode != "prompted",
				ValidateFlagsFunc: func(gotCtx context.Context) error {
					events = append(events, "flags")
					if gotCtx != ctx {
						t.Error("flag validator received a different context")
					}
					if flags.Team == "" {
						return missingTeam
					}
					return nil
				},
				Args: []naistrix.Argument{{
					Name: "member", ChoicesCaseInsensitive: true,
					ChoicesFunc: func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
						events = append(events, "choices")
						if flags.Team != "Dev" || args.Len() != 0 {
							t.Errorf("unexpected flag %q or prefix %v", flags.Team, args.All())
						}
						return []string{"Alice"}, nil
					},
					PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
						events = append(events, "prompt")
						return "Member of " + flags.Team, nil
					},
				}},
				ValidateArgsFunc: func(_ context.Context, args *naistrix.Arguments) error {
					events = append(events, "arguments")
					if args.Get("member") != "Alice" {
						t.Errorf("argument validation received %q", args.Get("member"))
					}
					return nil
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					events = append(events, "run")
					if args.Get("member") != "Alice" {
						t.Errorf("execution received %q", args.Get("member"))
					}
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"cmd"}
			want := []string{"flags", "choices", "arguments", "run"}
			switch mode {
			case "supplied":
				args = append(args, "--team", "Dev", "ALICE")
			case "prompted":
				args = append(args, "--team", "Dev")
				restore := naistrix.SetChoicePromptForTest(func(prompt string, _ []string, _ ...input.SelectOptionFunc) (string, error) {
					events = append(events, "select")
					if prompt != "#1 MEMBER: Member of Dev" {
						t.Errorf("unexpected question %q", prompt)
					}
					return "Alice", nil
				})
				defer restore()
				want = []string{"flags", "choices", "prompt", "select", "arguments", "run"}
			case "missing flag":
				args = append(args, "ALICE")
				want = []string{"flags"}
			case "missing flag and argument":
				want = []string{"flags"}
			}
			err := app.Run(naistrix.RunWithArgs(args), naistrix.RunWithContext(ctx))
			if strings.HasPrefix(mode, "missing flag") {
				if !errors.Is(err, missingTeam) {
					t.Fatalf("expected original flag error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(events, want) {
				t.Errorf("events %v, want %v; output %q", events, want, stdout.String())
			}
		})
	}
}

func TestFlagValidationSources(t *testing.T) {
	for _, source := range []string{"CLI", "configuration", "environment", "default", "global", "sticky"} {
		t.Run(source, func(t *testing.T) {
			app, _, _ := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			args := []string{"cmd"}
			switch source {
			case "configuration":
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte("team: Dev\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", path)
			case "environment":
				t.Setenv("COMPLETIONTEST_TEAM", "Dev")
			case "default":
				flags.Team = "Dev"
			default:
				args = append(args, "--team", "Dev")
			}
			calls, runs := 0, 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command",
				ValidateFlagsFunc: func(context.Context) error {
					calls++
					if flags.Team != "Dev" {
						t.Errorf("flag validation saw unsynchronized team %q", flags.Team)
					}
					return nil
				},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					runs++
					return nil
				},
			}
			parent := cmd
			switch source {
			case "global":
				if err := app.AddGlobalFlags(flags); err != nil {
					t.Fatal(err)
				}
			case "sticky":
				parent = &naistrix.Command{Name: "parent", Title: "Parent", StickyFlags: flags, SubCommands: []*naistrix.Command{cmd}}
				args = append([]string{"parent"}, args...)
			default:
				cmd.Flags = flags
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(naistrix.RunWithArgs(args)); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || runs != 1 {
				t.Errorf("flag validations %d, executions %d; want one each", calls, runs)
			}
		})
	}
}

func TestParentHelpSkipsValidation(t *testing.T) {
	for _, target := range [][]string{{"parent"}, {"parent", "nested"}} {
		for _, failure := range []string{"flags", "arguments", "choices", "missing arguments"} {
			t.Run(strings.Join(target, "/")+"/"+failure, func(t *testing.T) {
				app, stdout, _ := newCompletionApplication(t)
				var calls []string
				blocked := errors.New("validation must not run for help")
				leaf := &naistrix.Command{
					Name: "leaf", Title: "Leaf", RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
						calls = append(calls, "run")
						return blocked
					},
				}
				nested := &naistrix.Command{Name: "nested", Title: "Nested", SubCommands: []*naistrix.Command{leaf}}
				parent := &naistrix.Command{Name: "parent", Title: "Parent", SubCommands: []*naistrix.Command{nested}}
				for _, cmd := range []*naistrix.Command{parent, nested} {
					cmd.ValidateFlagsFunc = func(context.Context) error {
						calls = append(calls, "flags")
						if failure == "flags" {
							return blocked
						}
						return nil
					}
					cmd.ValidateArgsFunc = func(context.Context, *naistrix.Arguments) error {
						calls = append(calls, "arguments")
						return blocked
					}
					if failure == "choices" || failure == "missing arguments" {
						cmd.Args = []naistrix.Argument{{Name: "required"}}
						if failure == "choices" {
							cmd.Args[0].ChoicesFunc = func(context.Context, *naistrix.Arguments) ([]string, error) {
								calls = append(calls, "choices")
								return nil, blocked
							}
						}
					}
				}
				if err := app.AddCommand(parent); err != nil {
					t.Fatal(err)
				}
				if err := app.Run(naistrix.RunWithArgs(target)); err != nil {
					t.Fatalf("expected help without validation errors, got %v", err)
				}
				if len(calls) != 0 {
					t.Errorf("unexpected callbacks: %v", calls)
				}
				childName := "nested"
				if len(target) > 1 {
					childName = "leaf"
				}
				if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), childName) {
					t.Errorf("expected parent help listing %s, got %q", childName, stdout.String())
				}
			})
		}
	}
}

func TestParentFlagValidation(t *testing.T) {
	for _, mode := range []string{"execution", "completion", "execution failure", "completion failure"} {
		t.Run(mode, func(t *testing.T) {
			app, stdout, stderr := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			var events []string
			blocked := errors.New("--team is required")
			child := &naistrix.Command{
				Name: "child", Title: "Child", DisableArgumentPrompts: true,
				ValidateFlagsFunc: func(context.Context) error { events = append(events, "child flags"); return nil },
				Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
					events = append(events, "choices")
					return []string{"Alice"}, nil
				}}},
				ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					events = append(events, "child arguments")
					return nil
				},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					events = append(events, "run")
					return nil
				},
			}
			parent := &naistrix.Command{
				Name: "parent", Title: "Parent", StickyFlags: flags, SubCommands: []*naistrix.Command{child},
				ValidateFlagsFunc: func(context.Context) error {
					events = append(events, "parent flags")
					if flags.Team == "" {
						return blocked
					}
					if flags.Team != "Dev" {
						t.Errorf("parent saw unsynchronized flags: %q", flags.Team)
					}
					return nil
				},
				ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					events = append(events, "parent arguments")
					return nil
				},
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			args := []string{"parent", "child"}
			if !strings.HasSuffix(mode, "failure") {
				args = append(args, "--team", "Dev")
			}
			var want []string
			if strings.HasPrefix(mode, "completion") {
				got, directive := requestCompletion(t, app, stdout, append(args, "")...)
				want = []string{"parent flags", "child flags", "choices"}
				if strings.HasSuffix(mode, "failure") {
					want = []string{"parent flags"}
					if !slices.Equal(got, cobra.AppendActiveHelp(nil, "Argument completion failed: --team is required")) || !strings.Contains(stderr.String(), blocked.Error()) {
						t.Errorf("unexpected failure completions %v, stderr %q", got, stderr.String())
					}
				} else if !slices.Equal(got, []string{"Alice"}) {
					t.Errorf("unexpected choices %v", got)
				}
				if directive != cobra.ShellCompDirectiveNoFileComp {
					t.Errorf("unexpected directive %v", directive)
				}
			} else {
				err := app.Run(naistrix.RunWithArgs(append(args, "Alice")))
				want = []string{"parent flags", "parent arguments", "child flags", "choices", "child arguments", "run"}
				if strings.HasSuffix(mode, "failure") {
					want = []string{"parent flags"}
					if !errors.Is(err, blocked) {
						t.Errorf("expected parent flag error, got %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
			if !slices.Equal(events, want) {
				t.Errorf("events %v, want %v", events, want)
			}
		})
	}
}

func TestFlagValidationDuringCompletion(t *testing.T) {
	for _, tt := range []struct {
		name     string
		supplied bool
	}{
		{name: "missing flag"},
		{name: "valid flag", supplied: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, stderr := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			var events []string
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", Flags: flags,
				ValidateFlagsFunc: func(context.Context) error {
					events = append(events, "flags")
					if flags.Team == "" {
						return errors.New("--team is required")
					}
					return nil
				},
				Args: []naistrix.Argument{{
					Name: "member",
					ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
						events = append(events, "choices")
						return []string{"Alice"}, nil
					},
					PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
						events = append(events, "prompt")
						return "Member of " + flags.Team, nil
					},
				}},
				ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					t.Error("completion ran ValidateArgsFunc")
					return nil
				},
				RunFunc: completionMustNotRun(t),
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"cmd"}
			if tt.supplied {
				args = append(args, "--team", "Dev")
			}
			got, directive := requestCompletion(t, app, stdout, append(args, "")...)
			wantEvents := []string{"flags"}
			want := cobra.AppendActiveHelp(nil, "Argument completion failed: --team is required")
			if tt.supplied {
				wantEvents = append(wantEvents, "choices", "prompt")
				want = cobra.AppendActiveHelp([]string{"Alice"}, "Member of Dev")
			} else if !strings.Contains(stderr.String(), "--team is required") {
				t.Errorf("missing flag diagnostic: %q", stderr.String())
			}
			if !slices.Equal(events, wantEvents) || !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("events %v, suggestions %v, directive %v; want %v, %v, NoFileComp", events, got, directive, wantEvents, want)
			}
		})
	}
}

func TestFlagValidationCanRejectCommandsWithoutArguments(t *testing.T) {
	app, _, _ := newCompletionApplication(t)
	wantError := naistrix.Errorf("required flag missing")
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command",
		ValidateFlagsFunc: func(context.Context) error { return wantError },
		RunFunc:           completionMustNotRun(t),
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"cmd"})); !errors.Is(err, wantError) {
		t.Fatalf("expected original validation error, got %v", err)
	}
}

func TestFlagValidationPreservesExplicitCompletionOverrides(t *testing.T) {
	for _, mode := range []string{"function", "extensions"} {
		t.Run(mode, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
				ValidateFlagsFunc: func(context.Context) error {
					t.Error("automatic flag validator ran for an explicit completion override")
					return nil
				},
				Args:             []naistrix.Argument{{Name: "value", Choices: []string{"Alice"}}},
				AutoCompleteFunc: func(context.Context, *naistrix.Arguments, string) ([]string, string) { return []string{"custom"}, "" },
			}
			if mode == "extensions" {
				cmd.AutoCompleteExtensions = []string{"yaml"}
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, "cmd", "")
			if mode == "function" {
				if !slices.Equal(got, []string{"custom"}) || directive != cobra.ShellCompDirectiveNoFileComp {
					t.Errorf("suggestions %v, directive %v", got, directive)
				}
			} else if !slices.Contains(got, "yaml") || directive != cobra.ShellCompDirectiveFilterFileExt {
				t.Errorf("suggestions %v, directive %v", got, directive)
			}
		})
	}
}

// Ensure applications using only the old validator still have no early argument validation requirement.
func TestArgumentValidatorRemainsAfterResolution(t *testing.T) {
	app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", Args: []naistrix.Argument{{Name: "value", Choices: []string{"Alice"}, ChoicesCaseInsensitive: true}},
		ValidateArgsFunc: func(_ context.Context, args *naistrix.Arguments) error {
			calls++
			if args.Get("value") != "Alice" {
				t.Error("validator ran before normalization")
			}
			return nil
		},
		RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"cmd", "ALICE", "--config", filepath.Join(t.TempDir(), "config.yaml")})); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("got %d validations, want 1", calls)
	}
}
