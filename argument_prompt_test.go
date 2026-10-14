package naistrix

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix/input"
)

func TestDynamicArgumentPrompt(t *testing.T) {
	enableArgumentPrompts(t)
	for _, tt := range []struct {
		name, dynamic, want           string
		choices, repeatable, supplied bool
	}{
		{name: "text prompt", dynamic: "Member of Dev with role owner", want: "Member of Dev with role owner"},
		{name: "choice prompt", dynamic: "Member of Dev with role owner", choices: true, want: "Member of Dev with role owner"},
		{name: "default text fallback", want: "Enter a value"},
		{name: "default choice fallback", choices: true, want: "Select a value"},
		{name: "repeatable choices", dynamic: "Select members of Dev", choices: true, repeatable: true, want: "Select members of Dev"},
		{name: "supplied values skip callback", supplied: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			previousText, previousSelect, previousConfirm := promptArgument, selectArgument, confirmArgument
			t.Cleanup(func() {
				promptArgument, selectArgument, confirmArgument = previousText, previousSelect, previousConfirm
			})
			selections := 0
			answer := func(question string) (string, error) {
				selections++
				if tt.supplied || question != "#2 MEMBER: "+tt.want {
					t.Errorf("unexpected question %q", question)
				}
				return "alice", nil
			}
			promptArgument = answer
			selectArgument = func(question string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
				if !slices.Equal(choices, []string{"alice", "bob"}) {
					t.Errorf("choices = %v, want alice and bob", choices)
				}
				return answer(question)
			}
			confirmArgument = func(string, ...input.ConfirmOptionFunc) (bool, error) {
				return selections == 1, nil
			}
			app, global, err := NewApplication("prompttest", "Test", "v0.0.0", ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			global.Config = filepath.Join(t.TempDir(), "config.yaml")
			flags := &struct{ Team string }{}
			ctx := t.Context()
			calls := 0
			member := Argument{
				Name: "member", Repeatable: tt.repeatable,
				PromptFunc: func(gotCtx context.Context, args *Arguments) (string, error) {
					calls++
					if gotCtx != ctx || flags.Team != "Dev" || !slices.Equal(args.All(), []string{"owner"}) {
						t.Errorf("unexpected context, team %q, or arguments %v", flags.Team, args.All())
					}
					return tt.dynamic, nil
				},
			}
			if tt.choices {
				member.Choices = []string{"alice", "bob"}
			}
			cmd := &Command{
				Name: "cmd", Title: "Command", Flags: flags,
				Args: []Argument{{Name: "role", Choices: []string{"owner"}, ChoicesCaseInsensitive: true}, member},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					want := []string{"owner", "alice"}
					if tt.repeatable {
						want = append(want, "alice")
					}
					if !slices.Equal(args.All(), want) {
						t.Errorf("arguments = %v, want %v", args.All(), want)
					}
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			args := []string{"cmd", "--team", "Dev", "OWNER"}
			if tt.supplied {
				args = append(args, "alice")
			}
			if err := app.Run(RunWithArgs(args), RunWithContext(ctx)); err != nil {
				t.Fatal(err)
			}
			wantCalls := 1
			if tt.supplied {
				wantCalls = 0
			}
			if calls != wantCalls || cmd.Args[1].Prompt != "" {
				t.Errorf("prompt calls = %d, retained prompt = %q; want %d and no retained text", calls, cmd.Args[1].Prompt, wantCalls)
			}
		})
	}
}

func TestPromptAndPromptFuncAreMutuallyExclusive(t *testing.T) {
	for _, tt := range []struct {
		name               string
		prompt             string
		dynamic, wantError bool
	}{
		{name: "neither source"},
		{name: "static source", prompt: "Static text"},
		{name: "dynamic source", dynamic: true},
		{name: "both sources", prompt: "Static text", dynamic: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := NewApplication("prompttest", "Test", "v0.0.0", ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			arg := Argument{Name: "value", Prompt: tt.prompt}
			if tt.dynamic {
				arg.PromptFunc = func(context.Context, *Arguments) (string, error) {
					t.Error("registration must not invoke PromptFunc")
					return "Dynamic text", nil
				}
			}
			err = app.AddCommand(&Command{Name: "cmd", Title: "Command", Args: []Argument{arg}, RunFunc: func(context.Context, *Arguments, *OutputWriter) error { return nil }})
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "both Prompt and PromptFunc") {
					t.Errorf("error = %v, want mutually exclusive prompt sources error", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDynamicArgumentPromptError(t *testing.T) {
	enableArgumentPrompts(t)
	for _, kind := range []string{"text", "choices"} {
		t.Run(kind, func(t *testing.T) {
			previousText, previousSelect := promptArgument, selectArgument
			defer func() { promptArgument, selectArgument = previousText, previousSelect }()
			promptArgument = func(string) (string, error) {
				t.Fatal("text prompt opened after PromptFunc failed")
				return "", nil
			}
			selectArgument = func(string, []string, ...input.SelectOptionFunc) (string, error) {
				t.Fatal("selection opened after PromptFunc failed")
				return "", nil
			}
			wantError := errors.New("cannot render question")
			calls := 0
			arg := Argument{Name: "member", PromptFunc: func(context.Context, *Arguments) (string, error) {
				calls++
				return "ignored text", wantError
			}}
			if kind == "choices" {
				arg.Choices = []string{"alice", "bob"}
			}
			app, global, err := NewApplication("prompttest", "Test", "v0.0.0", ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			global.Config = filepath.Join(t.TempDir(), "config.yaml")
			if err := app.AddCommand(&Command{
				Name: "cmd", Title: "Command", Args: []Argument{arg},
				ValidateFunc: func(context.Context, *Arguments) error { t.Error("validation ran after PromptFunc failed"); return nil },
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
					t.Error("RunFunc ran after PromptFunc failed")
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(RunWithArgs([]string{"cmd"}))
			if !errors.Is(err, wantError) || !strings.Contains(err.Error(), `argument "member"`) || calls != 1 {
				t.Fatalf("expected wrapped prompt error and one call, got error %v, calls %d", err, calls)
			}
		})
	}
}
