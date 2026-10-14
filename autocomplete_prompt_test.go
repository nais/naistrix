package naistrix_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/spf13/cobra"
)

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
	want := cobra.AppendActiveHelp(nil, "Argument completion failed: cannot render help please retry")
	if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveNoFileComp || calls != 1 {
		t.Fatalf("completion %v, directive %v, calls %d; want %v, NoFileComp, one call", got, directive, calls, want)
	}
	if !strings.Contains(stderr.String(), "cannot render help") {
		t.Errorf("missing prompt failure diagnostic: %q", stderr.String())
	}
}
