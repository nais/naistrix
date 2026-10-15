package naistrix_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/spf13/cobra"
)

func TestArgumentFileCompletion(t *testing.T) {
	for _, tt := range []struct {
		name      string
		args      []naistrix.Argument
		input     []string
		want      []string
		wantFiles bool
		wantHelp  string
	}{
		{
			name: "file only", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml", "json"}}},
			input: []string{""}, want: []string{"json", "yaml"}, wantFiles: true,
		},
		{
			name: "custom file help", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}, Prompt: "Select a manifest"}},
			input: []string{""}, want: []string{"yaml"}, wantFiles: true, wantHelp: "Select a manifest",
		},
		{
			name: "file prefix is filtered by shell", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"dir/conf"}, want: []string{"yaml"}, wantFiles: true,
		},
		{
			name: "choices before file", args: []naistrix.Argument{{Name: "mode", Choices: []string{"apply", "preview"}}, {Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"a"}, want: []string{"apply"},
		},
		{
			name: "file after choices", args: []naistrix.Argument{{Name: "mode", Choices: []string{"apply"}}, {Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"apply", ""}, want: []string{"yaml"}, wantFiles: true,
		},
		{
			name: "different extensions per position", args: []naistrix.Argument{{Name: "source", AutoCompleteExtensions: []string{"json"}}, {Name: "target", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"input.json", ""}, want: []string{"yaml"}, wantFiles: true,
		},
		{
			name: "choices after file", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}}, {Name: "mode", Choices: []string{"apply"}}},
			input: []string{"input.yaml", ""}, want: []string{"apply"},
		},
		{
			name: "unrestricted before file", args: []naistrix.Argument{{Name: "name"}, {Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{""},
		},
		{
			name: "unrestricted after file", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}}, {Name: "name"}},
			input: []string{"input.yaml", ""},
		},
		{
			name: "arguments filled", args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"input.yaml", ""},
		},
		{
			name: "repeatable file", args: []naistrix.Argument{{Name: "file", Repeatable: true, AutoCompleteExtensions: []string{"yaml"}}},
			input: []string{"first.yaml", "second.yaml", ""}, want: []string{"yaml"}, wantFiles: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			extensions := make([][]string, len(tt.args))
			for i, arg := range tt.args {
				extensions[i] = slices.Clone(arg.AutoCompleteExtensions)
			}
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", Args: tt.args, RunFunc: completionMustNotRun(t),
				ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
					t.Error("completion ran argument validation")
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				got, directive := requestCompletion(t, app, stdout, append([]string{"cmd"}, tt.input...)...)
				wantDirective := cobra.ShellCompDirectiveNoFileComp
				want := slices.Clone(tt.want)
				if tt.wantFiles {
					wantDirective = cobra.ShellCompDirectiveFilterFileExt
					formatted := make([]string, len(tt.want))
					for i, ext := range tt.want {
						formatted[i] = "*." + ext
					}
					help := tt.wantHelp
					if help == "" {
						help = "Select a file (" + strings.Join(formatted, " or ") + ")."
					}
					want = cobra.AppendActiveHelp(want, help)
				}
				if !slices.Equal(got, want) || directive != wantDirective {
					t.Fatalf("got %v, directive %v; want %v, directive %v", got, directive, want, wantDirective)
				}
			}
			for i, arg := range cmd.Args {
				if !slices.Equal(arg.AutoCompleteExtensions, extensions[i]) {
					t.Errorf("completion mutated extensions for argument %q", arg.Name)
				}
			}
		})
	}
}

func TestFileCompletionPrerequisitesAndPrompt(t *testing.T) {
	for _, failure := range []string{"none", "parent flags", "flags", "earlier choices", "prompt"} {
		t.Run(failure, func(t *testing.T) {
			app, stdout, stderr := newCompletionApplication(t)
			flags := &struct{ Team string }{}
			var events []string
			blocked := errors.New("completion blocked")
			validator := func(name string) naistrix.ValidateFlagsFunc {
				return func(context.Context) error {
					events = append(events, name)
					if flags.Team != "Dev" {
						t.Errorf("unsynchronized flags: %q", flags.Team)
					}
					if failure == name {
						return blocked
					}
					return nil
				}
			}
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t), ValidateFlagsFunc: validator("flags"),
				Args: []naistrix.Argument{
					{Name: "mode", ChoicesCaseInsensitive: true, ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
						events = append(events, "earlier choices")
						if failure == "earlier choices" {
							return nil, blocked
						}
						return []string{"Apply"}, nil
					}},
					{Name: "file", AutoCompleteExtensions: []string{"yaml"}, PromptFunc: func(_ context.Context, args *naistrix.Arguments) (string, error) {
						events = append(events, "prompt")
						if args.Len() != 1 || args.Get("mode") != "Apply" {
							t.Errorf("unexpected earlier arguments: %v", args.All())
						}
						if failure == "prompt" {
							return "", blocked
						}
						return "Select an Apply manifest", nil
					}},
				},
			}
			if err := app.AddCommand(&naistrix.Command{
				Name: "parent", Title: "Parent", StickyFlags: flags, SubCommands: []*naistrix.Command{cmd},
				ValidateFlagsFunc: validator("parent flags"),
			}); err != nil {
				t.Fatal(err)
			}
			got, directive := requestCompletion(t, app, stdout, "parent", "cmd", "--team", "Dev", "APPLY", "")
			wantEvents := []string{"parent flags", "flags", "earlier choices", "prompt"}
			if failure == "none" {
				want := cobra.AppendActiveHelp([]string{"yaml"}, "Select an Apply manifest")
				if !slices.Equal(got, want) || directive != cobra.ShellCompDirectiveFilterFileExt {
					t.Errorf("got %v, directive %v", got, directive)
				}
			} else {
				wantEvents = wantEvents[:slices.Index(wantEvents, failure)+1]
				if directive != cobra.ShellCompDirectiveNoFileComp || !strings.Contains(strings.Join(got, " "), blocked.Error()) || !strings.Contains(stderr.String(), blocked.Error()) {
					t.Errorf("expected completion error, got %v, directive %v, stderr %q", got, directive, stderr.String())
				}
			}
			if !slices.Equal(events, wantEvents) {
				t.Errorf("events %v, want %v", events, wantEvents)
			}
		})
	}
}

func TestFileCompletionDefinitionConflicts(t *testing.T) {
	for _, arg := range []naistrix.Argument{
		{Name: "file", Choices: []string{"one"}, AutoCompleteExtensions: []string{"yaml"}},
		{Name: "file", ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) { return nil, nil }, AutoCompleteExtensions: []string{"yaml"}},
	} {
		app, _, _ := newCompletionApplication(t)
		err := app.AddCommand(&naistrix.Command{Name: "cmd", Title: "Command", Args: []naistrix.Argument{arg}, RunFunc: completionMustNotRun(t)})
		if err == nil || !strings.Contains(err.Error(), "cannot have both AutoCompleteExtensions") {
			t.Fatalf("expected conflicting completion definition error, got %v", err)
		}
	}
}

func TestFileCompletionRejectsInvalidEarlierValue(t *testing.T) {
	app, stdout, stderr := newCompletionApplication(t)
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", RunFunc: completionMustNotRun(t),
		Args: []naistrix.Argument{
			{Name: "mode", Choices: []string{"apply"}},
			{Name: "file", AutoCompleteExtensions: []string{"yaml"}, PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
				t.Error("file prompt ran with invalid earlier arguments")
				return "", nil
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, directive := requestCompletion(t, app, stdout, "cmd", "invalid", "")
	if directive != cobra.ShellCompDirectiveNoFileComp || !strings.Contains(strings.Join(got, " "), "invalid value") || !strings.Contains(stderr.String(), "invalid value") {
		t.Fatalf("expected invalid earlier argument diagnostic, got %v, directive %v, stderr %q", got, directive, stderr.String())
	}
}

func TestFileCompletionDoesNotValidatePaths(t *testing.T) {
	app, _, _ := newCompletionApplication(t)
	called := false
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command", Args: []naistrix.Argument{{Name: "file", AutoCompleteExtensions: []string{"yaml"}}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
			called = true
			if args.Get("file") != "nonexistent.txt" {
				t.Errorf("unexpected path %q", args.Get("file"))
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"cmd", "nonexistent.txt"})); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("command was not executed")
	}
}
