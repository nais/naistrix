package naistrix_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestCommandValidation(t *testing.T) {
	ctx := context.Background()
	noop := func(context.Context, naistrix.Output, []string) error {
		return nil
	}

	tests := []struct {
		name          string
		cmd           *naistrix.Command
		panicContains string
	}{
		{
			name: "command with no name",
			cmd: &naistrix.Command{
				Title:   "Test command",
				RunFunc: noop,
			},
			panicContains: "cannot be empty",
		},
		{
			name: "command with space in name",
			cmd: &naistrix.Command{
				Name:    "test command",
				Title:   "Test command",
				RunFunc: noop,
			},
			panicContains: "contain spaces",
		},
		{
			name: "command with no title",
			cmd: &naistrix.Command{
				Name:    "cmd",
				RunFunc: noop,
			},
			panicContains: "missing a title",
		},
		{
			name: "command with newline in title",
			cmd: &naistrix.Command{
				Name:    "test",
				Title:   "Test command\nwith newline",
				RunFunc: noop,
			},
			panicContains: "contains newline",
		},
		{
			name: "missing RunFunc and SubCommands",
			cmd: &naistrix.Command{
				Name:  "test",
				Title: "Some title",
			},
			panicContains: "either RunFunc or SubCommands must be set",
		},
		{
			name: "has both RunFunc and SubCommands",
			cmd: &naistrix.Command{
				Name:    "test",
				Title:   "Some title",
				RunFunc: noop,
				SubCommands: []*naistrix.Command{
					{
						Name:    "sub",
						Title:   "Some title",
						RunFunc: noop,
					},
				},
			},
			panicContains: "either RunFunc or SubCommands must be set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &naistrix.Application{
				Name:        "app",
				SubCommands: []*naistrix.Command{tt.cmd},
			}

			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for command with no name, but did not panic")
				} else if msg := r.(string); !strings.Contains(msg, tt.panicContains) {
					t.Fatalf("expected panic message to contain %q, got: %q", tt.panicContains, msg)
				}
			}()

			_ = app.Run(
				naistrix.RunWithContext(ctx),
				naistrix.RunWithOutput(naistrix.Discard()),
				naistrix.RunWithArgs([]string{"-h"}),
			)
		})
	}
}

func TestArgumentUseString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		expectedArgsString string
		args               []naistrix.Argument
	}{
		{
			name:               "no arguments",
			expectedArgsString: "",
		},
		{
			name:               "argument",
			expectedArgsString: "ARG",
			args: []naistrix.Argument{
				{Name: "arg"},
			},
		},
		{
			name:               "repeatable argument",
			expectedArgsString: "ARG [ARG...]",
			args: []naistrix.Argument{
				{Name: "arg", Repeatable: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &naistrix.Application{
				Name: "app",
				SubCommands: []*naistrix.Command{
					{
						Name:  "test",
						Title: "Test command",
						Args:  tt.args,
						RunFunc: func(context.Context, naistrix.Output, []string) error {
							return nil
						},
					},
				},
			}
			buf := &bytes.Buffer{}
			out := naistrix.NewWriter(buf)
			if err := app.Run(naistrix.RunWithContext(ctx), naistrix.RunWithOutput(out), naistrix.RunWithArgs([]string{"test", "-h"})); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			expectedUsage := strings.TrimSpace("Usage:\n  app test "+tt.expectedArgsString) + " [flags]\n"
			if helpText := buf.String(); !strings.Contains(helpText, expectedUsage) {
				t.Fatalf("expected help text to contain %q, got %q", expectedUsage, helpText)
			}
		})
	}
}

func TestCommandArgumentValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		args          []naistrix.Argument
		panicContains string
	}{
		{
			name: "missing argument name",
			args: []naistrix.Argument{
				{Repeatable: true},
			},
			panicContains: "cannot be empty",
		},
		{
			name: "repeatable argument must be last",
			args: []naistrix.Argument{
				{Name: "arg1", Repeatable: true},
				{Name: "arg2"},
			},
			panicContains: "must be the last argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if msg, ok := recover().(string); ok && !strings.Contains(msg, tt.panicContains) {
					t.Fatalf("expected panic message to contain %q, got: %q", tt.panicContains, msg)
				}
			}()
			_ = (&naistrix.Application{
				Name: "app",
				SubCommands: []*naistrix.Command{
					{
						Name:  "test",
						Title: "Test command",
						RunFunc: func(context.Context, naistrix.Output, []string) error {
							return nil
						},
						Args: tt.args,
					},
				},
			}).Run(naistrix.RunWithContext(ctx), naistrix.RunWithOutput(naistrix.Discard()), naistrix.RunWithArgs([]string{"-h"}))
			t.Fatalf("expected panic")
		})
	}
}
