package naistrix

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplicationOutputDestinations(t *testing.T) {
	var result, diagnostics bytes.Buffer
	for _, tt := range []struct {
		name        string
		options     []ApplicationOptionFunc
		result      io.Writer
		diagnostics io.Writer
	}{
		{name: "defaults", result: os.Stdout, diagnostics: os.Stderr},
		{
			name: "nil defaults",
			options: []ApplicationOptionFunc{
				ApplicationWithWriter(nil), ApplicationWithDiagnosticWriter(nil),
			},
			result: os.Stdout, diagnostics: os.Stderr,
		},
		{
			name: "custom results preserve stderr", options: []ApplicationOptionFunc{ApplicationWithWriter(&result)},
			result: &result, diagnostics: os.Stderr,
		},
		{
			name: "custom diagnostics preserve stdout", options: []ApplicationOptionFunc{ApplicationWithDiagnosticWriter(&diagnostics)},
			result: os.Stdout, diagnostics: &diagnostics,
		},
		{
			name: "custom destinations",
			options: []ApplicationOptionFunc{
				ApplicationWithWriter(&result), ApplicationWithDiagnosticWriter(&diagnostics),
			},
			result: &result, diagnostics: &diagnostics,
		},
		{
			name: "merged destinations",
			options: []ApplicationOptionFunc{
				ApplicationWithWriter(&result), ApplicationWithDiagnosticWriter(&result),
			},
			result: &result, diagnostics: &result,
		},
		{
			name: "discard both",
			options: []ApplicationOptionFunc{
				ApplicationWithWriter(io.Discard), ApplicationWithDiagnosticWriter(io.Discard),
			},
			result: io.Discard, diagnostics: io.Discard,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := NewApplication("streamtest", "Output streams", "v0.0.0", tt.options...)
			if err != nil {
				t.Fatal(err)
			}
			if app.output.writer != tt.result || app.rootCommand.OutOrStdout() != tt.result {
				t.Error("results and Cobra output must use the result destination")
			}
			if app.output.diagnosticWriter != tt.diagnostics || app.rootCommand.ErrOrStderr() != tt.diagnostics {
				t.Error("diagnostics and Cobra errors must use the diagnostic destination")
			}
		})
	}
}

func TestApplicationCobraDiagnostics(t *testing.T) {
	var result, diagnostics bytes.Buffer
	app, flags, err := NewApplication("streamtest", "Output streams", "v0.0.0",
		ApplicationWithWriter(&result), ApplicationWithDiagnosticWriter(&diagnostics))
	if err != nil {
		t.Fatal(err)
	}
	flags.Config = filepath.Join(t.TempDir(), "config.yaml")
	if err := app.AddCommand(&Command{
		Name: "show", Title: "Show a result",
		RunFunc: func(_ context.Context, _ *Arguments, out *OutputWriter) error {
			out.Println("result")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Cobra children inherit the root's error destination, including completion diagnostics.
	app.rootCommand.Commands()[0].PrintErrln("cobra diagnostic")
	if err := app.Run(RunWithArgs([]string{"show"})); err != nil {
		t.Fatal(err)
	}
	if result.String() != "result\n" || !strings.Contains(diagnostics.String(), "cobra diagnostic") {
		t.Fatalf("incorrect destinations: results=%q, diagnostics=%q", result.String(), diagnostics.String())
	}
}

func TestStandaloneOutputWriterDestinations(t *testing.T) {
	for _, tt := range []struct {
		name    string
		merged  bool
		discard bool
	}{
		{name: "separate destinations"},
		{name: "merged destinations", merged: true},
		{name: "discard diagnostics", discard: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var results, diagnostics bytes.Buffer
			var diagnosticWriter io.Writer = &diagnostics
			if tt.merged {
				diagnosticWriter = &results
			} else if tt.discard {
				diagnosticWriter = io.Discard
			}
			level := OutputVerbosityLevelTrace
			out := NewOutputWriter(&results, diagnosticWriter, &level)
			out.Println("result").Warnln("warning").Traceln("trace")
			if !strings.HasPrefix(results.String(), "result\n") {
				t.Fatalf("missing result: %q", results.String())
			}
			messages := diagnostics.String()
			if tt.merged {
				messages = results.String()
			} else if results.String() != "result\n" {
				t.Errorf("diagnostics leaked into results: %q", results.String())
			}
			if tt.discard {
				if messages != "" {
					t.Errorf("discarded diagnostics reached a buffer: %q", messages)
				}
				return
			}
			for _, message := range []string{"warning", "trace"} {
				if !strings.Contains(messages, message) {
					t.Errorf("missing diagnostic %q: %q", message, messages)
				}
			}
		})
	}
}
