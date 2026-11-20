package input_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

func TestContextPromptsRefuseDisabledOrCanceledInteraction(t *testing.T) {
	restore := input.SetInteractive(func() bool { return true })
	defer restore()
	readOutput, restoreOutput := input.CapturePromptOutputForTest(t)
	defer restoreOutput()

	base := t.Context()
	disabled := input.WithNonInteractive(base)
	canceled, cancel := context.WithCancel(base)
	cancel()
	prompts := []struct {
		name string
		call func(context.Context) error
	}{
		{name: "input", call: func(ctx context.Context) error {
			_, err := input.InputContext(ctx, "Name")
			return err
		}},
		{name: "secret", call: func(ctx context.Context) error {
			_, err := input.SecretContext(ctx, "Token")
			return err
		}},
		{name: "confirmation with default yes", call: func(ctx context.Context) error {
			confirmed, err := input.ConfirmContext(ctx, "Proceed?", input.ConfirmWithDefaultTrue())
			if confirmed {
				t.Error("disabled or canceled interaction must not imply confirmation")
			}
			return err
		}},
		{name: "selection", call: func(ctx context.Context) error {
			_, err := input.SelectContext(ctx, "Pick one", []string{"one", "two"})
			return err
		}},
	}
	for _, tt := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "disabled with terminal", ctx: disabled, want: input.ErrNotInteractive},
		{name: "already canceled", ctx: canceled, want: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, prompt := range prompts {
				t.Run(prompt.name, func(t *testing.T) {
					if err := prompt.call(tt.ctx); !errors.Is(err, tt.want) {
						t.Fatalf("prompt error = %v, want %v", err, tt.want)
					}
				})
			}
		})
	}
	value, err := input.SelectContext(disabled, "Pick one", []string{"only"}, input.SelectWithAutoSelectSingleOption())
	if err != nil || value != "only" {
		t.Fatalf("non-prompting single selection = %q, %v", value, err)
	}
	if _, err := input.SelectContext(canceled, "Pick one", []string{"only"}, input.SelectWithAutoSelectSingleOption()); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled single selection = %v, want context cancellation", err)
	}
	duringLabels, cancelLabels := context.WithCancel(disabled)
	defer cancelLabels()
	_, err = input.SelectContext(duringLabels, "Pick one", []cancelingChoice{{label: "one", cancel: cancelLabels}, {label: "two"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation during label formatting = %v, want context cancellation", err)
	}
	if !input.IsInteractiveContext(base) || !input.IsInteractive() {
		t.Error("context policy leaked into the parent context or terminal detection")
	}
	if output := readOutput(); output != "" {
		t.Fatalf("disabled or canceled prompts produced UI output: %q", output)
	}
}

type cancelingChoice struct {
	label  string
	cancel context.CancelFunc
}

func (c cancelingChoice) String() string {
	if c.cancel != nil {
		c.cancel()
	}
	return c.label
}

func TestNonInteractiveInvocation(t *testing.T) {
	restore := input.SetInteractive(func() bool { return true })
	defer restore()
	type requestKey struct{}
	base := context.WithValue(t.Context(), requestKey{}, "request")

	tests := []struct {
		name          string
		args          []string
		choices       bool
		confirmation  bool
		contextPolicy bool
		alias         bool
		interactive   bool
		wantRun       bool
	}{
		{name: "missing text argument"},
		{name: "missing choice argument", choices: true},
		{name: "complete invocation", args: []string{"one"}, wantRun: true},
		{name: "hook cannot confirm by default", args: []string{"one"}, confirmation: true},
		{name: "caller context disables prompting", contextPolicy: true},
		{name: "top-level alias honors flag", alias: true},
		{name: "next invocation remains interactive", args: []string{"one"}, interactive: true, wantRun: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookCalls := 0
			app, flags, err := naistrix.NewApplication("app", "title", "v0.0.0",
				naistrix.ApplicationWithWriter(io.Discard),
				naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
					Name: "policy",
					RunFunc: func(ctx context.Context, _ *naistrix.OutputWriter) error {
						hookCalls++
						if input.IsInteractiveContext(ctx) != tt.interactive {
							t.Error("interaction policy was not established before hooks")
							return errors.New("unexpected interaction policy")
						}
						if ctx.Value(requestKey{}) != "request" {
							t.Error("invocation lost its caller context")
						}
						if tt.confirmation {
							_, err := input.ConfirmContext(ctx, "Approve the change?", input.ConfirmWithDefaultTrue())
							return err
						}
						return nil
					},
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			flags.Config = filepath.Join(t.TempDir(), "config.yaml")
			argument := naistrix.Argument{Name: "resource"}
			if tt.choices {
				argument.Choices = []string{"one", "two"}
			}
			called := false
			command := &naistrix.Command{
				Name: "run", Title: "Run", TopLevelAliases: []string{"short"},
				Args: []naistrix.Argument{argument},
				RunFunc: func(ctx context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					called = true
					if input.IsInteractiveContext(ctx) != tt.interactive || args.Get("resource") != "one" {
						t.Error("handler received unexpected policy or arguments")
					}
					return nil
				},
			}
			if err := app.AddCommand(&naistrix.Command{
				Name: "resources", Title: "Resources",
				SubCommands: []*naistrix.Command{command},
			}); err != nil {
				t.Fatal(err)
			}
			path := []string{"resources", "run"}
			if tt.alias {
				path = []string{"short"}
			}
			cliArgs := append(path, tt.args...)
			ctx := base
			if tt.contextPolicy {
				ctx = input.WithNonInteractive(ctx)
			} else if !tt.interactive {
				cliArgs = append(cliArgs, "--non-interactive")
			}
			err = app.Run(naistrix.RunWithContext(ctx), naistrix.RunWithArgs(cliArgs))
			if (err == nil) != tt.wantRun || called != tt.wantRun || hookCalls != 1 {
				t.Fatalf("error %v, handler called %v, hook calls %d; want execution %v", err, called, hookCalls, tt.wantRun)
			}
			if tt.confirmation && !errors.Is(err, input.ErrNotInteractive) {
				t.Fatalf("confirmation error = %v, want ErrNotInteractive", err)
			}
		})
	}
	if !input.IsInteractiveContext(base) {
		t.Error("invocation policy changed the caller's context")
	}
}

func TestDeprecatedReplacementDoesNotPromptWhenNonInteractive(t *testing.T) {
	restore := input.SetInteractive(func() bool { return true })
	defer restore()
	readOutput, restoreOutput := input.CapturePromptOutputForTest(t)
	defer restoreOutput()

	app, flags, err := naistrix.NewApplication("app", "title", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	flags.Config = filepath.Join(t.TempDir(), "config.yaml")
	called := false
	if err := app.AddCommand(
		&naistrix.Command{
			Name: "old", Title: "Old",
			Deprecated: naistrix.DeprecatedWithReplacement([]string{"new"}),
		},
		&naistrix.Command{
			Name: "new", Title: "New",
			RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
				called = true
				return nil
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	err = app.Run(naistrix.RunWithArgs([]string{"old", "--non-interactive"}))
	if !errors.Is(err, input.ErrNotInteractive) || called {
		t.Fatalf("replacement error = %v, replacement executed = %v", err, called)
	}
	if output := readOutput(); output != "" {
		t.Fatalf("replacement produced prompt UI output: %q", output)
	}
}
