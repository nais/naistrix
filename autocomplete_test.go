package naistrix_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/spf13/cobra"
)

func newCompletionApplication(t *testing.T) (*naistrix.Application, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	app, flags, err := naistrix.NewApplication("completiontest", "Completion tests", "v0.0.0", naistrix.ApplicationWithWriter(&stdout))
	if err != nil {
		t.Fatal(err)
	}
	flags.Config = filepath.Join(t.TempDir(), "config.yaml")
	naistrix.SetCompletionErrorWriterForTest(app, &stderr)
	return app, &stdout, &stderr
}

func requestCompletion(t *testing.T, app *naistrix.Application, stdout *bytes.Buffer, args ...string) ([]string, cobra.ShellCompDirective) {
	t.Helper()
	stdout.Reset()
	if err := app.Run(naistrix.RunWithArgs(append([]string{cobra.ShellCompRequestCmd}, args...))); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, ":") {
		t.Fatalf("missing completion directive: %q", stdout.String())
	}
	number, err := strconv.Atoi(strings.TrimPrefix(last, ":"))
	if err != nil {
		t.Fatalf("invalid completion directive: %q", last)
	}
	return lines[:len(lines)-1], cobra.ShellCompDirective(number)
}

func completionMustNotRun(t *testing.T) naistrix.RunFunc {
	t.Helper()
	return func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
		t.Error("completion executed RunFunc")
		return nil
	}
}

func TestAutomaticChoiceCompletion(t *testing.T) {
	for _, tt := range []struct {
		name  string
		args  []naistrix.Argument
		input []string
		want  []string
	}{
		{name: "all choices", args: []naistrix.Argument{{Name: "value", Choices: []string{"foo", "bar"}}}, input: []string{""}, want: []string{"foo", "bar"}},
		{name: "prefix filter", args: []naistrix.Argument{{Name: "value", Choices: []string{"foo", "bar"}}}, input: []string{"b"}, want: []string{"bar"}},
		{name: "case sensitive prefix", args: []naistrix.Argument{{Name: "value", Choices: []string{"Foo"}}}, input: []string{"f"}},
		{name: "case insensitive prefix", args: []naistrix.Argument{{Name: "value", Choices: []string{"Foo"}, ChoicesCaseInsensitive: true}}, input: []string{"f"}, want: []string{"foo"}},
		{name: "uppercase prefix preserved", args: []naistrix.Argument{{Name: "value", Choices: []string{"Alice"}, ChoicesCaseInsensitive: true}}, input: []string{"AL"}, want: []string{"ALice"}},
		{name: "unicode prefix", args: []naistrix.Argument{{Name: "value", Choices: []string{"Kilo"}, ChoicesCaseInsensitive: true}}, input: []string{"K"}, want: []string{"Kilo"}},
		{name: "longer prefix", args: []naistrix.Argument{{Name: "value", Choices: []string{"Foo"}, ChoicesCaseInsensitive: true}}, input: []string{"fooo"}},
		{name: "unrestricted current argument", args: []naistrix.Argument{{Name: "action", Choices: []string{"foo"}}, {Name: "text"}}, input: []string{"foo", ""}},
		{name: "arguments already filled", args: []naistrix.Argument{{Name: "value", Choices: []string{"foo"}}}, input: []string{"foo", ""}},
		{name: "repeatable choices", args: []naistrix.Argument{{Name: "value", Repeatable: true, Choices: []string{"foo", "bar"}}}, input: []string{"foo", ""}, want: []string{"foo", "bar"}},
		{name: "later position", args: []naistrix.Argument{{Name: "text"}, {Name: "value", Choices: []string{"foo", "bar"}}}, input: []string{"anything", "f"}, want: []string{"foo"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			if err := app.AddCommand(&naistrix.Command{
				Name: "cmd", Title: "Command", Args: tt.args, RunFunc: completionMustNotRun(t),
				ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					t.Error("completion executed ValidateArgsFunc")
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, append([]string{"cmd"}, tt.input...)...)
			if !slices.Equal(got, tt.want) || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("got %v, directive %v; want %v, NoFileComp", got, directive, tt.want)
			}
		})
	}
}

func TestAutomaticCompletionOverrides(t *testing.T) {
	for _, mode := range []string{"custom function", "file extensions"} {
		t.Run(mode, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			customCalls := 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
				Args: []naistrix.Argument{{Name: "value", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
					t.Error("explicit override called ChoicesFunc")
					return nil, nil
				}}},
				AutoCompleteFunc: func(context.Context, *naistrix.Arguments, string) ([]string, string) {
					customCalls++
					return []string{"custom"}, ""
				},
			}
			if mode == "file extensions" {
				cmd.Args = []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml", "json"}}}
				cmd.Args[0].PromptFunc = func(context.Context, *naistrix.Arguments) (string, error) {
					t.Error("explicit override called PromptFunc")
					return "", nil
				}
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, "cmd", "")
			if !slices.Equal(got, []string{"custom"}) || customCalls != 1 || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("got %v, calls %d, directive %v", got, customCalls, directive)
			}
		})
	}
}

func TestDynamicChoiceCompletionPrefix(t *testing.T) {
	for _, repeatable := range []bool{false, true} {
		t.Run(fmt.Sprintf("repeatable=%v", repeatable), func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			teamCalls, memberCalls := 0, 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
				Args: []naistrix.Argument{
					{Name: "team", ChoicesCaseInsensitive: true, ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
						teamCalls++
						return []string{"Dev", "Ops"}, nil
					}},
					{Name: "member", Repeatable: repeatable, ChoicesCaseInsensitive: true, ChoicesFunc: func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
						memberCalls++
						if args.Len() != 1 || args.Get("team") != "Dev" {
							t.Errorf("unexpected prefix %v", args.All())
						}
						return []string{"Alice", "Bob"}, nil
					}},
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"cmd", "DEV"}
			if repeatable {
				args = append(args, "alice", "BOB")
			}
			args = append(args, "a")
			got, directive := requestCompletion(t, app, stdout, args...)
			if !slices.Equal(got, []string{"alice"}) || directive != cobra.ShellCompDirectiveNoFileComp || teamCalls != 1 || memberCalls != 1 {
				t.Fatalf("got %v, directive %v, calls %d/%d", got, directive, teamCalls, memberCalls)
			}
			if cmd.Args[0].Choices != nil || cmd.Args[1].Choices != nil {
				t.Error("completion retained resolved choices")
			}
		})
	}
}

func TestDynamicCompletionFlags(t *testing.T) {
	for _, source := range []string{"CLI", "config", "environment", "CLI over config", "global", "sticky", "verbose"} {
		t.Run(source, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			args := []string{"cmd"}
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if source == "config" || source == "CLI over config" {
				team := "Dev"
				if source == "CLI over config" {
					team = "Wrong"
				}
				if err := os.WriteFile(configPath, []byte("team: "+team+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", configPath)
			}
			if source == "environment" {
				t.Setenv("COMPLETIONTEST_TEAM", "Dev")
			} else if source != "config" {
				args = append(args, "--team", "Dev")
			}
			if source == "verbose" {
				args = append(args, "-vv")
			}
			calls := 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
				Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
					calls++
					if flags.Team != "Dev" || args.Len() != 0 {
						t.Errorf("unsynchronized flag %q or prefix %v", flags.Team, args.All())
					}
					return []string{"Alice", "Bob"}, nil
				}}},
			}
			parent := cmd
			switch source {
			case "global":
				if err := app.AddGlobalFlags(flags); err != nil {
					t.Fatal(err)
				}
			case "sticky":
				parent = &naistrix.Command{Name: "team", Title: "Team", StickyFlags: flags, SubCommands: []*naistrix.Command{cmd}, ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					t.Error("completion executed parent validation")
					return nil
				}}
				args = append([]string{"team"}, args...)
			default:
				cmd.Flags = flags
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			args = append(args, "A")
			got, directive := requestCompletion(t, app, stdout, args...)
			if !slices.Equal(got, []string{"Alice"}) || directive != cobra.ShellCompDirectiveNoFileComp || calls != 1 {
				t.Fatalf("got %v, directive %v, calls %d", got, directive, calls)
			}
		})
	}
}

func TestCompletionErrors(t *testing.T) {
	for _, mode := range []string{"provider error", "multiline provider error", "empty choices", "duplicate choices", "invalid earlier argument", "bad configuration"} {
		t.Run(mode, func(t *testing.T) {
			app, stdout, stderr := newCompletionApplication(t)
			providerCalls := 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
				Args: []naistrix.Argument{{Name: "value", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
					providerCalls++
					if mode == "empty choices" {
						return nil, nil
					}
					if mode == "duplicate choices" {
						return []string{"Foo", "Foo"}, nil
					}
					if mode == "multiline provider error" {
						return nil, errors.New("service unavailable\nplease log in\tbefore retrying")
					}
					return nil, errors.New("service unavailable")
				}}},
			}
			args := []string{"cmd"}
			if mode == "invalid earlier argument" {
				cmd.Args = append([]naistrix.Argument{{Name: "team", Choices: []string{"Dev"}}}, cmd.Args...)
				args = append(args, "unknown")
			}
			if mode == "bad configuration" {
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte("team: [\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", path)
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, append(args, "")...)
			diagnostic, _, _ := strings.Cut(stderr.String(), "\nCompletion ended with directive:")
			message := strings.Join(strings.Fields(diagnostic), " ")
			want := cobra.AppendActiveHelp(nil, message)
			if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp || !strings.Contains(message, "Completion failed:") {
				t.Fatalf("got %v, directive %v, stderr %q", got, directive, stderr.String())
			}
			wantCalls := 1
			if mode == "invalid earlier argument" || mode == "bad configuration" {
				wantCalls = 0
			}
			if providerCalls != wantCalls {
				t.Errorf("got %d provider calls, want %d", providerCalls, wantCalls)
			}
		})
	}
}

func TestCompletionResolvesChoicesForEachRequest(t *testing.T) {
	app, stdout, _ := newCompletionApplication(t)
	calls := 0
	cmd := &naistrix.Command{
		Name: "cmd", Title: "Command", Aliases: []string{"alias"}, RunFunc: completionMustNotRun(t),
		Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
			calls++
			return []string{fmt.Sprintf("member%d", calls)}, nil
		}}},
	}
	if err := app.AddCommand(cmd); err != nil {
		t.Fatal(err)
	}
	for i, command := range []string{"cmd", "alias"} {
		got, directive := requestCompletion(t, app, stdout, command, "")
		if !slices.Equal(got, []string{fmt.Sprintf("member%d", i+1)}) || directive != cobra.ShellCompDirectiveNoFileComp {
			t.Fatalf("got %v, directive %v", got, directive)
		}
	}
	if calls != 2 || cmd.Args[0].Choices != nil {
		t.Errorf("calls %d, retained choices %v", calls, cmd.Args[0].Choices)
	}
}

func TestCompletionCancellation(t *testing.T) {
	app, stdout, stderr := newCompletionApplication(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
		Args: []naistrix.Argument{{Name: "member", ChoicesFunc: func(gotCtx context.Context, _ *naistrix.Arguments) ([]string, error) {
			if gotCtx != ctx {
				t.Error("callback received a different context")
			}
			cancel()
			return []string{"Alice"}, nil
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{cobra.ShellCompRequestCmd, "cmd", ""}), naistrix.RunWithContext(ctx)); err != nil {
		t.Fatal(err)
	}
	help := cobra.AppendActiveHelp(nil, "Completion failed: context canceled")
	want := fmt.Sprintf("%s\n:%d\n", strings.Join(help, "\n"), cobra.ShellCompDirectiveNoFileComp)
	if stdout.String() != want || !strings.Contains(stderr.String(), "context canceled") {
		t.Fatalf("stdout %q, stderr %q", stdout.String(), stderr.String())
	}
}

func TestAutomaticChoiceCompletionPromptHelp(t *testing.T) {
	for _, tt := range []struct {
		name   string
		args   []naistrix.Argument
		input  []string
		want   []string
		prompt string
	}{
		{
			name:  "static choices",
			args:  []naistrix.Argument{{Name: "role", Prompt: "Role to assign", Choices: []string{"member", "owner"}}},
			input: []string{""}, want: []string{"member", "owner"}, prompt: "Role to assign",
		},
		{
			name: "dynamic choices",
			args: []naistrix.Argument{{Name: "member", Prompt: "Email address of the team member", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
				return []string{"user@example.com"}, nil
			}}},
			input: []string{""}, want: []string{"user@example.com"}, prompt: "Email address of the team member",
		},
		{
			name: "current argument prompt",
			args: []naistrix.Argument{
				{Name: "role", Prompt: "Role to assign", Choices: []string{"owner"}},
				{Name: "member", Prompt: "Email address of the team member", Choices: []string{"user@example.com"}},
			},
			input: []string{"owner", ""}, want: []string{"user@example.com"}, prompt: "Email address of the team member",
		},
		{
			name:  "repeatable argument",
			args:  []naistrix.Argument{{Name: "member", Prompt: "Email address of the team member", Repeatable: true, Choices: []string{"user@example.com"}}},
			input: []string{"user@example.com", ""}, want: []string{"user@example.com"}, prompt: "Email address of the team member",
		},
		{
			name:  "no prefix matches still shows help",
			args:  []naistrix.Argument{{Name: "role", Prompt: "Role to assign", Choices: []string{"member", "owner"}}},
			input: []string{"unknown"}, prompt: "Role to assign",
		},
		{
			name:  "no prompt",
			args:  []naistrix.Argument{{Name: "role", Choices: []string{"member", "owner"}}},
			input: []string{""}, want: []string{"member", "owner"},
		},
		{
			name:  "all arguments supplied",
			args:  []naistrix.Argument{{Name: "role", Prompt: "Role to assign", Choices: []string{"owner"}}},
			input: []string{"owner", ""},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			if err := app.AddCommand(&naistrix.Command{
				Name: "cmd", Title: "Command", Args: tt.args, RunFunc: completionMustNotRun(t),
			}); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, append([]string{"cmd"}, tt.input...)...)
			want := slices.Clone(tt.want)
			if tt.prompt != "" {
				want = cobra.AppendActiveHelp(want, tt.prompt)
			}
			if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("completion = %v, directive %v; want %v, NoFileComp", got, directive, want)
			}
		})
	}
}

func TestDynamicCompletionPrompt(t *testing.T) {
	for _, tt := range []struct {
		name, dynamic, want   string
		repeatable, noMatches bool
	}{
		{name: "dynamic text", dynamic: "Member of Dev with role owner", want: "Member of Dev with role owner"},
		{name: "no custom text"},
		{name: "repeatable prefix", dynamic: "Members of Dev", repeatable: true, want: "Members of Dev"},
		{name: "no matches still shows help", dynamic: "Members of Dev", noMatches: true, want: "Members of Dev"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			calls := 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", Flags: flags, RunFunc: completionMustNotRun(t),
				Args: []naistrix.Argument{
					{Name: "role", Choices: []string{"owner"}, ChoicesCaseInsensitive: true, PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
						t.Error("completion must not render an earlier argument's prompt")
						return "Earlier argument", nil
					}},
					{
						Name: "member", Repeatable: tt.repeatable,
						PromptFunc: func(_ context.Context, args *naistrix.Arguments) (string, error) {
							calls++
							if flags.Team != "Dev" || !slices.Equal(args.All(), []string{"owner"}) {
								t.Errorf("unsynchronized team %q or unresolved prefix %v", flags.Team, args.All())
							}
							return tt.dynamic, nil
						},
						ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
							return []string{"alice", "bob"}, nil
						},
					},
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"cmd", "--team", "Dev", "OWNER"}
			if tt.repeatable {
				args = append(args, "alice", "bob")
			}
			prefix := ""
			want := []string{"alice", "bob"}
			if tt.noMatches {
				prefix, want = "unknown", nil
			}
			got, directive := requestCompletion(t, app, stdout, append(args, prefix)...)
			if tt.want != "" {
				want = cobra.AppendActiveHelp(want, tt.want)
			}
			if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp || calls != 1 || cmd.Args[1].Prompt != "" {
				t.Errorf("got %v, directive %v, calls %d, retained prompt %q; want %v, NoFileComp, one call, no retained text", got, directive, calls, cmd.Args[1].Prompt, want)
			}
		})
	}
}

func TestCompletionPromptRefreshesForEachRequest(t *testing.T) {
	app, stdout, _ := newCompletionApplication(t)
	flags := &struct{ Team string }{}
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", Flags: flags, RunFunc: completionMustNotRun(t),
		Args: []naistrix.Argument{{Name: "member", Choices: []string{"alice"}, PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
			return fmt.Sprintf("Member of %s", flags.Team), nil
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, team := range []string{"Dev", "Ops"} {
		got, _ := requestCompletion(t, app, stdout, "cmd", "--team", team, "")
		if want := cobra.AppendActiveHelp([]string{"alice"}, "Member of "+team); !slices.Equal(got, want) {
			t.Errorf("completion = %v, want %v", got, want)
		}
	}
}

func TestCompletionPromptError(t *testing.T) {
	app, stdout, stderr := newCompletionApplication(t)
	calls := 0
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
		Args: []naistrix.Argument{{Name: "member", Choices: []string{"alice", "bob"}, PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
			calls++
			return "ignored text", errors.New("cannot render help\nplease retry")
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	got, directive := requestCompletion(t, app, stdout, "cmd", "")
	want := cobra.AppendActiveHelp(nil, "Completion failed: cannot render help please retry")
	if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp || calls != 1 {
		t.Fatalf("completion %v, directive %v, calls %d; want %v, NoFileComp, one call", got, directive, calls, want)
	}
	if !strings.Contains(stderr.String(), "cannot render help") {
		t.Errorf("missing prompt failure diagnostic: %q", stderr.String())
	}
}
