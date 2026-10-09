package naistrix

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nais/naistrix/input"
)

func enableArgumentPrompts(t *testing.T) {
	t.Helper()
	previous := argumentsInteractive
	argumentsInteractive = func() bool { return true }
	t.Cleanup(func() { argumentsInteractive = previous })
}

func TestCommandRejectsBlankPromptAnswers(t *testing.T) {
	enableArgumentPrompts(t)
	for _, repeatable := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "repeatable"}[repeatable], func(t *testing.T) {
			previousPrompt := promptArgument
			defer func() { promptArgument = previousPrompt }()
			answers := []string{"", " \t ", " hello "}
			if repeatable {
				answers = append(answers, "")
			}
			promptArgument = func(question string) (string, error) {
				wantQuestion := "#1 WORD: Enter a word"
				if repeatable && len(answers) == 1 {
					wantQuestion = "#1 WORD: Enter another value, or press Enter to finish"
				}
				if question != wantQuestion || len(answers) == 0 {
					t.Fatalf("unexpected prompt %q, remaining answers: %v", question, answers)
				}
				answer := answers[0]
				answers = answers[1:]
				return answer, nil
			}
			var output bytes.Buffer
			app, _, err := NewApplication("app", "title", "v0.0.0", ApplicationWithWriter(&output))
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := app.AddCommand(&Command{
				Name: "test", Title: "Test",
				Args: []Argument{{Name: "word", Prompt: "Enter a word", Repeatable: repeatable}},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					got = args.All()
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs([]string{"test"})); err != nil {
				t.Fatal(err)
			}
			if len(answers) != 0 || !reflect.DeepEqual(got, []string{" hello "}) {
				t.Errorf("got %v, remaining answers: %v", got, answers)
			}
			if count := strings.Count(output.String(), "A value is required"); count != 2 {
				t.Errorf("expected 2 warnings, got %d: %s", count, output.String())
			}
		})
	}
}

func TestCommandPromptErrorAfterBlankAnswer(t *testing.T) {
	enableArgumentPrompts(t)
	previousPrompt := promptArgument
	defer func() { promptArgument = previousPrompt }()
	wantError := errors.New("prompt cancelled")
	calls := 0
	promptArgument = func(string) (string, error) {
		calls++
		if calls == 1 {
			return "", nil
		}
		return "", wantError
	}
	app, _, err := NewApplication("app", "title", "v0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddCommand(&Command{
		Name: "test", Title: "Test", Args: []Argument{{Name: "word", Prompt: "Enter a word"}},
		RunFunc: func(_ context.Context, _ *Arguments, _ *OutputWriter) error {
			t.Error("RunFunc called after prompt error")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"test"})); !errors.Is(err, wantError) {
		t.Fatalf("expected wrapped prompt error, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 prompts, got %d", calls)
	}
}

func TestCommandPromptsForMissingArguments(t *testing.T) {
	enableArgumentPrompts(t)
	for _, tt := range []struct {
		name     string
		provided []string
		answers  []string
	}{
		{name: "all missing", answers: []string{"first", "second", "third"}},
		{name: "some missing", provided: []string{"first"}, answers: []string{"second", "third"}},
		{name: "all provided", provided: []string{"first", "second", "third"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			err = app.AddCommand(&Command{
				Name: "test", Title: "Test",
				Args: []Argument{
					{Name: "one", Prompt: "First question"},
					{Name: "two", Prompt: "Second question"},
					{Name: "three", Prompt: "Third question"},
				},
				ValidateFunc: func(_ context.Context, args *Arguments) error {
					if args.Len() != 3 {
						t.Errorf("validator received %d arguments, want 3", args.Len())
					}
					return nil
				},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					got = args.All()
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			previousPrompt := promptArgument
			defer func() { promptArgument = previousPrompt }()
			questions := []string{"#1 ONE: First question", "#2 TWO: Second question", "#3 THREE: Third question"}[len(tt.provided):]
			promptArgument = func(question string) (string, error) {
				if len(questions) == 0 || question != questions[0] {
					t.Fatalf("unexpected prompt %q, remaining: %v", question, questions)
					return "", errors.New("unexpected prompt")
				}
				questions = questions[1:]
				answer := tt.answers[0]
				tt.answers = tt.answers[1:]
				return answer, nil
			}
			if err := app.Run(RunWithArgs(append([]string{"test"}, tt.provided...))); err != nil {
				t.Fatal(err)
			}
			if len(questions) != 0 {
				t.Errorf("missing prompts: %v", questions)
			}
			if want := []string{"first", "second", "third"}; !reflect.DeepEqual(got, want) {
				t.Errorf("got arguments %v, want %v", got, want)
			}
		})
	}
}

func TestCommandDefaultArgumentPrompts(t *testing.T) {
	enableArgumentPrompts(t)
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "default prompts", true: "disabled prompts"}[disabled], func(t *testing.T) {
			previousPrompt := promptArgument
			defer func() { promptArgument = previousPrompt }()
			questions := []string{"#1 ONE: Enter a value", "#2 TWO: Custom question", "#3 THREE: Enter a value", "#3 THREE: Enter another value, or press Enter to finish"}
			answers := []string{"first", "second", "third", ""}
			promptArgument = func(question string) (string, error) {
				if disabled || len(questions) == 0 || question != questions[0] {
					t.Fatalf("unexpected prompt %q, remaining: %v", question, questions)
				}
				answer := answers[0]
				questions, answers = questions[1:], answers[1:]
				return answer, nil
			}
			app, _, err := NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := app.AddCommand(&Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: disabled,
				Args: []Argument{{Name: "one"}, {Name: "two", Prompt: "Custom question"}, {Name: "three", Repeatable: true}},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					got = args.All()
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(RunWithArgs([]string{"test"}))
			if disabled {
				if err == nil || !strings.Contains(err.Error(), "Expected at least 3 arguments") || got != nil {
					t.Fatalf("expected validation failure without execution, got %v, %v", err, got)
				}
			} else if err != nil || len(questions) != 0 || !reflect.DeepEqual(got, []string{"first", "second", "third"}) {
				t.Fatalf("error %v, remaining questions %v, values %v", err, questions, got)
			}
		})
	}
}

func TestCommandMissingArgumentsWithoutTerminal(t *testing.T) {
	previousInteractive := argumentsInteractive
	defer func() { argumentsInteractive = previousInteractive }()
	argumentsInteractive = func() bool { return false }
	previousPrompt := promptArgument
	defer func() { promptArgument = previousPrompt }()
	promptArgument = func(string) (string, error) {
		t.Fatal("prompt attempted without an interactive terminal")
		return "", input.ErrNotInteractive
	}
	previousSelect := selectArgument
	defer func() { selectArgument = previousSelect }()
	selectArgument = func(string, []string, ...input.SelectOptionFunc) (string, error) {
		t.Fatal("selection attempted without an interactive terminal")
		return "", input.ErrNotInteractive
	}

	for _, tt := range []struct {
		name      string
		args      []Argument
		provided  []string
		wantError string
	}{
		{name: "custom prompt", args: []Argument{{Name: "one", Prompt: "First?"}}, wantError: "Expected exactly 1 argument"},
		{name: "choice prompt", args: []Argument{{Name: "one", Choices: []string{"foo", "bar"}}}, wantError: "Expected exactly 1 argument"},
		{name: "unconfigured argument", args: []Argument{{Name: "one"}}, wantError: "Expected exactly 1 argument"},
		{name: "mixed prompts", args: []Argument{{Name: "one"}, {Name: "two", Prompt: "Second?"}}, wantError: "Expected exactly 2 arguments"},
		{name: "provided values need no terminal", args: []Argument{{Name: "one", Prompt: "First?"}}, provided: []string{"value"}},
		{name: "provided repeatable values need no terminal", args: []Argument{{Name: "one", Prompt: "First?", Repeatable: true}}, provided: []string{"first", "second"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			called := false
			if err := app.AddCommand(&Command{
				Name: "test", Title: "Test", Args: tt.args,
				RunFunc: func(_ context.Context, _ *Arguments, _ *OutputWriter) error {
					called = true
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(RunWithArgs(append([]string{"test"}, tt.provided...)))
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if called != (err == nil) {
				t.Errorf("RunFunc called: %v, error: %v", called, err)
			}
		})
	}
}

func TestRepeatableChoiceCollection(t *testing.T) {
	enableArgumentPrompts(t)
	for _, failConfirmation := range []bool{false, true} {
		t.Run(map[bool]string{false: "multiple values", true: "confirmation error"}[failConfirmation], func(t *testing.T) {
			previousSelect, previousConfirm := selectArgument, confirmArgument
			defer func() {
				selectArgument, confirmArgument = previousSelect, previousConfirm
			}()
			wantError := errors.New("confirmation cancelled")
			selections, confirmations := 0, 0
			selectArgument = func(prompt string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
				if prompt != "#1 VALUE: Choose a value" || !reflect.DeepEqual(choices, []string{"foo", "bar"}) || selections >= 2 {
					t.Fatalf("unexpected selection: %q, %v", prompt, choices)
				}
				value := choices[selections]
				selections++
				return value, nil
			}
			confirmArgument = func(prompt string, _ ...input.ConfirmOptionFunc) (bool, error) {
				if prompt != "Add another VALUE?" {
					t.Errorf("unexpected confirmation %q", prompt)
				}
				confirmations++
				if failConfirmation {
					return false, wantError
				}
				return confirmations == 1, nil
			}
			app, _, err := NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := app.AddCommand(&Command{
				Name: "test", Title: "Test",
				Args: []Argument{{Name: "value", Prompt: "Choose a value", Choices: []string{"foo", "bar"}, Repeatable: true}},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					got = args.GetRepeatable("value")
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(RunWithArgs([]string{"test"}))
			if failConfirmation {
				if !errors.Is(err, wantError) || got != nil {
					t.Fatalf("expected wrapped confirmation error and no execution, got %v, %v", err, got)
				}
			} else if err != nil || !reflect.DeepEqual(got, []string{"foo", "bar"}) || selections != 2 || confirmations != 2 {
				t.Fatalf("got %v, error %v, selections %d, confirmations %d", got, err, selections, confirmations)
			}
		})
	}
}

func TestRepeatableTextCollectionError(t *testing.T) {
	previousPrompt := promptArgument
	defer func() { promptArgument = previousPrompt }()
	wantError := errors.New("input cancelled")
	calls := 0
	promptArgument = func(string) (string, error) {
		calls++
		if calls == 1 {
			return "first", nil
		}
		return "", wantError
	}
	var output bytes.Buffer
	values, err := (Argument{Name: "word", Prompt: "Enter a word", Repeatable: true}).promptValues(context.Background(), NewOutputWriter(&output, new(Count)), 1)
	if !errors.Is(err, wantError) || values != nil || calls != 2 {
		t.Fatalf("got values %v, error %v, calls %d", values, err, calls)
	}
}

func TestCommandCollectsRepeatableArguments(t *testing.T) {
	enableArgumentPrompts(t)
	app, _, err := NewApplication("app", "title", "v0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := app.AddCommand(&Command{
		Name: "test", Title: "Test",
		Args: []Argument{{Name: "words", Repeatable: true, Prompt: "Enter a word"}},
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
			got = args.GetRepeatable("words")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	previousPrompt := promptArgument
	defer func() { promptArgument = previousPrompt }()
	answers := []string{"hello", "two words", " \t "}
	promptArgument = func(question string) (string, error) {
		wantQuestion := "#1 WORDS: Enter another value, or press Enter to finish"
		if len(answers) == 3 {
			wantQuestion = "#1 WORDS: Enter a word"
		}
		if question != wantQuestion || len(answers) == 0 {
			t.Fatalf("unexpected prompt: %q", question)
		}
		answer := answers[0]
		answers = answers[1:]
		return answer, nil
	}
	if err := app.Run(RunWithArgs([]string{"test"})); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"hello", "two words"}) || len(answers) != 0 {
		t.Errorf("got %v, remaining answers: %v", got, answers)
	}
}

func TestCommandValidatesPromptedArguments(t *testing.T) {
	enableArgumentPrompts(t)
	previousPrompt := promptArgument
	defer func() { promptArgument = previousPrompt }()
	promptArgument = func(string) (string, error) { return "invalid", nil }

	app, _, err := NewApplication("app", "title", "v0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	if err := app.AddCommand(&Command{
		Name: "test", Title: "Test", Args: []Argument{{Name: "value", Prompt: "Enter a value"}},
		ValidateFunc: func(_ context.Context, args *Arguments) error {
			if args.Get("value") != "valid" {
				return Errorf("invalid value")
			}
			return nil
		},
		RunFunc: func(_ context.Context, _ *Arguments, _ *OutputWriter) error {
			called = true
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"test"})); err == nil || !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("expected validation error, got %v", err)
	}
	if called {
		t.Error("RunFunc called with invalid prompted input")
	}
}

func TestCommandArgumentChoices(t *testing.T) {
	enableArgumentPrompts(t)
	for _, tt := range []struct {
		name       string
		provided   []string
		prompt     string
		repeatable bool
		selectErr  error
		wantError  string
	}{
		{name: "select missing value", prompt: "Choose a value"},
		{name: "select with default prompt"},
		{name: "provided value skips selection", prompt: "Choose a value", provided: []string{"foo"}},
		{name: "valid without prompt", provided: []string{"bar"}},
		{name: "invalid without prompt", provided: []string{"baz"}, wantError: "invalid value"},
		{name: "invalid with prompt", prompt: "Choose a value", provided: []string{"baz"}, wantError: "invalid value"},
		{name: "repeatable selection", prompt: "Choose a value", repeatable: true},
		{name: "valid repeatable values", repeatable: true, provided: []string{"foo", "bar"}},
		{name: "invalid repeatable value", repeatable: true, provided: []string{"foo", "baz"}, wantError: "invalid value"},
		{name: "selection requires terminal", prompt: "Choose a value", selectErr: input.ErrNotInteractive, wantError: "failed to prompt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			previousSelect := selectArgument
			previousPrompt := promptArgument
			previousConfirm := confirmArgument
			defer func() {
				selectArgument = previousSelect
				promptArgument = previousPrompt
				confirmArgument = previousConfirm
			}()
			confirmArgument = func(prompt string, _ ...input.ConfirmOptionFunc) (bool, error) {
				if !tt.repeatable || len(tt.provided) > 0 || prompt != "Add another VALUE?" {
					t.Errorf("unexpected confirmation: %q", prompt)
				}
				return false, nil
			}
			promptArgument = func(string) (string, error) {
				t.Error("used text prompt instead of selection")
				return "", nil
			}
			selections := 0
			selectArgument = func(prompt string, choices []string, _ ...input.SelectOptionFunc) (string, error) {
				selections++
				question := tt.prompt
				if question == "" {
					question = "Select a value"
				}
				if prompt != "#1 VALUE: "+question || !reflect.DeepEqual(choices, []string{"foo", "bar"}) {
					t.Errorf("unexpected selection prompt %q, choices %v", prompt, choices)
				}
				return "bar", tt.selectErr
			}
			app, _, err := NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			validated := false
			if err := app.AddCommand(&Command{
				Name: "test", Title: "Test",
				Args: []Argument{{Name: "value", Prompt: tt.prompt, Choices: []string{"foo", "bar"}, Repeatable: tt.repeatable}},
				ValidateFunc: func(_ context.Context, _ *Arguments) error {
					validated = true
					return nil
				},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					got = args.All()
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(RunWithArgs(append([]string{"test"}, tt.provided...)))
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
				}
				if got != nil || validated {
					t.Error("executed command or custom validator after invalid input")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				want := tt.provided
				if len(want) == 0 {
					want = []string{"bar"}
				}
				if !reflect.DeepEqual(got, want) || !validated {
					t.Errorf("got %v, want %v; custom validation ran: %v", got, want, validated)
				}
			}
			if tt.selectErr != nil && !errors.Is(err, tt.selectErr) {
				t.Errorf("expected wrapped selection error %v, got %v", tt.selectErr, err)
			}
			wantSelections := 0
			if len(tt.provided) == 0 {
				wantSelections = 1
			}
			if selections != wantSelections {
				t.Errorf("got %d selections, want %d", selections, wantSelections)
			}
		})
	}
}
