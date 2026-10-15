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

func TestInput_All(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		if got := newArguments([]Argument{}, []string{}).All(); len(got) > 0 {
			t.Errorf("expected no arguments, got: %v", got)
		}
	})

	t.Run("regular arguments", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2"}
		args := newArguments([]Argument{{Name: "a1"}, {Name: "a2"}}, cobraArgs)
		if args.Len() != len(cobraArgs) {
			t.Errorf("expected %d arguments, got: %d", len(cobraArgs), args.Len())
		}

		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})

	t.Run("with repeatable", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2", "v3", "v4"}
		args := newArguments([]Argument{{Name: "a1"}, {Name: "a2", Repeatable: true}}, cobraArgs)
		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})

	t.Run("only repeatable", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2", "v3", "v4"}
		args := newArguments([]Argument{{Name: "a1", Repeatable: true}}, cobraArgs)
		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})
}

func TestArgumentsChoiceNormalizationPreservesInput(t *testing.T) {
	values := []string{"FOO", "bar", "baz", "unknown"}
	definitions := []Argument{
		{Name: "action", Choices: []string{"Foo"}, ChoicesCaseInsensitive: true},
		{Name: "values", Repeatable: true, Choices: []string{"Bar", "BAZ"}, ChoicesCaseInsensitive: true},
	}
	args := newArguments(definitions, values)
	if got, want := args.All(), []string{"Foo", "Bar", "BAZ", "unknown"}; !slices.Equal(got, want) {
		t.Errorf("All() = %v, want %v", got, want)
	}
	if want := []string{"FOO", "bar", "baz", "unknown"}; !slices.Equal(values, want) {
		t.Errorf("input values changed to %v, want %v", values, want)
	}
}

func TestArgumentsLen(t *testing.T) {
	for _, tt := range []struct {
		name    string
		defined []Argument
		values  []string
	}{
		{name: "no values"},
		{name: "missing value", defined: []Argument{{Name: "first"}, {Name: "second"}}, values: []string{"one"}},
		{name: "extra value", defined: []Argument{{Name: "first"}}, values: []string{"one", "two"}},
		{name: "repeatable values", defined: []Argument{{Name: "first"}, {Name: "rest", Repeatable: true}}, values: []string{"one", "two", "three"}},
		{name: "only repeatable values", defined: []Argument{{Name: "values", Repeatable: true}}, values: []string{"one", "two"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := newArguments(tt.defined, tt.values)
			if got, want := args.Len(), len(tt.values); got != want {
				t.Errorf("Len() = %d, want %d", got, want)
			}
			if err := ValidateExactArgs(len(tt.values))(t.Context(), args); err != nil {
				t.Errorf("exact count validation failed: %v", err)
			}
		})
	}

	t.Run("extra values fail exact validation", func(t *testing.T) {
		args := newArguments([]Argument{{Name: "first"}}, []string{"one", "two"})
		if err := ValidateExactArgs(1)(t.Context(), args); err == nil {
			t.Error("expected extra values to fail exact count validation")
		}
	})
}

func TestInput_Get(t *testing.T) {
	cobraArgs := []string{"v1", "v2", "v3", "v4"}
	args := newArguments(
		[]Argument{
			{Name: "a1"},
			{Name: "a2", Repeatable: true},
		},
		cobraArgs,
	)

	t.Run("get regular arg", func(t *testing.T) {
		got := args.Get("a1")
		if expected := cobraArgs[0]; got != expected {
			t.Errorf(`expected argument "a1" to be %q, got: %q`, expected, got)
		}
	})

	t.Run("get repeatable arg", func(t *testing.T) {
		got := args.GetRepeatable("a2")
		if expected := cobraArgs[1:]; !slices.Equal(got, expected) {
			t.Errorf(`expected "a2" to be %q, got: %q`, expected, got)
		}
	})

	t.Run("get regular arg as repeatable", func(t *testing.T) {
		defer func() {
			expectedError := `"a1" is not a valid repeatable argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.GetRepeatable("a1")
	})

	t.Run("get repeatable arg as regular", func(t *testing.T) {
		defer func() {
			expectedError := `"a2" is not a valid argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.Get("a2")
	})

	t.Run("get non-existing regular arg", func(t *testing.T) {
		defer func() {
			expectedError := `"foo" is not a valid argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.Get("foo")
	})

	t.Run("get non-existing repeatable arg", func(t *testing.T) {
		defer func() {
			expectedError := `"foo" is not a valid repeatable argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.GetRepeatable("foo")
	})
}

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
				ValidateArgsFunc: func(context.Context, *Arguments) error { t.Error("validation ran after PromptFunc failed"); return nil },
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
