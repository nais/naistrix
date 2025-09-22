package naistrix_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestCommandValidation(t *testing.T) {
	noop := func(context.Context, *naistrix.OutputWriter, []string) error { return nil }

	tests := []struct {
		name          string
		cmd           *naistrix.Command
		errorContains string
	}{
		{
			name: "command with no name",
			cmd: &naistrix.Command{
				Title:   "Test command",
				RunFunc: noop,
			},
			errorContains: "cannot be empty",
		},
		{
			name: "command with space in name",
			cmd: &naistrix.Command{
				Name:    "test command",
				Title:   "Test command",
				RunFunc: noop,
			},
			errorContains: "contain spaces",
		},
		{
			name: "command with no title",
			cmd: &naistrix.Command{
				Name:    "cmd",
				RunFunc: noop,
			},
			errorContains: "missing a title",
		},
		{
			name: "command with newline in title",
			cmd: &naistrix.Command{
				Name:    "test",
				Title:   "Test command\nwith newline",
				RunFunc: noop,
			},
			errorContains: "contains newline",
		},
		{
			name: "missing RunFunc and SubCommands",
			cmd: &naistrix.Command{
				Name:  "test",
				Title: "Some title",
			},
			errorContains: "either RunFunc or SubCommands must be set",
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
			errorContains: "either RunFunc or SubCommands must be set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			err = app.AddCommand(tt.cmd)
			if err == nil {
				t.Fatalf("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("expected error message to contain %q, got: %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestArgumentUseString(t *testing.T) {
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
			buf := &bytes.Buffer{}
			app, _, err := naistrix.NewApplication(
				"app",
				"title",
				"v0.0.0",
				naistrix.ApplicationWithWriter(buf),
			)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			err = app.AddCommand(&naistrix.Command{
				Name:  "test",
				Title: "Test command",
				Args:  tt.args,
				RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error {
					return nil
				},
			})
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if err := app.Run(naistrix.RunWithArgs([]string{"test", "-h"})); err != nil {
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
	tests := []struct {
		name          string
		args          []naistrix.Argument
		errorContains string
	}{
		{
			name: "missing argument name",
			args: []naistrix.Argument{
				{Repeatable: true},
			},
			errorContains: "cannot be empty",
		},
		{
			name: "repeatable argument must be last",
			args: []naistrix.Argument{
				{Name: "arg1", Repeatable: true},
				{Name: "arg2"},
			},
			errorContains: "must be the last argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			err = app.AddCommand(&naistrix.Command{
				Name:    "test",
				Title:   "Test command",
				RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error { return nil },
				Args:    tt.args,
			})
			if err == nil {
				t.Fatalf("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("expected error message to contain %q, got: %q", tt.errorContains, err.Error())
			}
		})
	}
}
