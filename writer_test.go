package naistrix_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/pterm/pterm"
)

func TestOutputWriter_SeparateDiagnostics(t *testing.T) {
	for _, merged := range []bool{false, true} {
		name := "separate"
		if merged {
			name = "merged"
		}
		t.Run(name, func(t *testing.T) {
			var result, diagnostics bytes.Buffer
			var diagnosticWriter io.Writer = &diagnostics
			if merged {
				diagnosticWriter = &result
			}
			options := []naistrix.ApplicationOptionFunc{
				naistrix.ApplicationWithWriter(&result),
				naistrix.ApplicationWithDiagnosticWriter(diagnosticWriter),
			}
			app, flags, err := naistrix.NewApplication("streamtest", "Output streams", "v0.0.0", options...)
			if err != nil {
				t.Fatal(err)
			}
			flags.Config = filepath.Join(t.TempDir(), "config.yaml")
			if err := app.AddCommand(&naistrix.Command{
				Name: "show", Title: "Show a result",
				PreRunHooks: []naistrix.PreRunHook{{Name: "notice", RunFunc: func(_ context.Context, out *naistrix.OutputWriter) error {
					out.Warnln("hook warning")
					return nil
				}}},
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Successln("saved").Successf("saved %d\n", 1).
						Infoln("loaded").Infof("loaded %d\n", 1).
						Warnln("warning").Warnf("warning %d\n", 1).
						Errorln("diagnostic").Errorf("diagnostic %d\n", 1).
						Verboseln("verbose").Verbosef("verbose %d\n", 1).
						Debugln("debug").Debugf("debug %d\n", 1).
						Traceln("trace").Tracef("trace %d\n", 1)
					return out.JSON().Render(map[string]string{"status": "confirmed"})
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(naistrix.RunWithArgs([]string{"show", "-vvv"})); err != nil {
				t.Fatal(err)
			}
			messages := result.String()
			if !merged {
				messages = diagnostics.String()
				var value map[string]string
				decoder := json.NewDecoder(&result)
				if err := decoder.Decode(&value); err != nil || value["status"] != "confirmed" {
					t.Fatalf("invalid result: %v, %v", value, err)
				}
				if err := decoder.Decode(&value); err != io.EOF {
					t.Fatalf("unexpected output after JSON result: %v", err)
				}
			} else if diagnostics.Len() != 0 {
				t.Fatalf("merged output unexpectedly used a separate stream: %q", diagnostics.String())
			}
			for _, message := range []string{"Initializing configuration", "hook warning", "saved", "saved 1", "loaded", "loaded 1", "warning", "warning 1", "diagnostic", "diagnostic 1", "verbose", "verbose 1", "debug", "debug 1", "trace", "trace 1"} {
				if !strings.Contains(messages, message) {
					t.Errorf("missing diagnostic %q in %q", message, messages)
				}
			}
		})
	}
}

func TestOutputWriter_PlainResultsUseTheirOwnWriter(t *testing.T) {
	var first, second, diagnostics bytes.Buffer
	level := naistrix.OutputVerbosityLevelNormal
	out := naistrix.NewOutputWriter(&first, &diagnostics, &level)
	_ = naistrix.NewOutputWriter(&second, &second, &level)
	out.Println("first").Printf("result %d\n", 1)
	out.Warnln("separate diagnostic")
	if first.String() != "first\nresult 1\n" || second.Len() != 0 {
		t.Fatalf("results went to the wrong writer: first=%q, second=%q", first.String(), second.String())
	}
	if !strings.Contains(diagnostics.String(), "separate diagnostic") {
		t.Fatalf("missing diagnostic: %q", diagnostics.String())
	}
}

func TestOutputWriter_ConditionalOutput(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		flags    []string
	}{
		{
			name:     "regular output",
			expected: "normal: n1 n2\n",
			flags:    []string{},
		},
		{
			name:     "verbose output",
			expected: "normal: n1 n2\nverbose: v1 v2\nverbosef: v1\n",
			flags:    []string{"-v"},
		},
		{
			name:     "debug output",
			expected: "normal: n1 n2\nverbose: v1 v2\nverbosef: v1\nDEBUG: debug: d1 d2\nDEBUG: debugf: d1\n",
			flags:    []string{"-vv"},
		},
		{
			name:     "trace output",
			expected: "normal: n1 n2\nverbose: v1 v2\nverbosef: v1\nDEBUG: debug: d1 d2\nDEBUG: debugf: d1\nTRACE: trace: t1 t2\nTRACE: tracef: t1\n",
			flags:    []string{"-vvv"},
		},
	}

	pterm.DisableStyling()
	defer pterm.EnableStyling()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0",
				naistrix.ApplicationWithWriter(&buf), naistrix.ApplicationWithDiagnosticWriter(&buf))
			if err != nil {
				t.Fatalf("unable to create application: %v", err)
			}

			err = app.AddCommand(&naistrix.Command{
				Name:  "test",
				Title: "Test command",
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Println("normal:", "n1", "n2")
					out.Verboseln("verbose:", "v1", "v2")
					out.Verbosef("verbosef: %s\n", "v1")
					out.Debugln("debug:", "d1", "d2")
					out.Debugf("debugf: %s\n", "d1")
					out.Traceln("trace:", "t1", "t2")
					out.Tracef("tracef: %s\n", "t1")

					return nil
				},
			})
			if err != nil {
				t.Fatalf("unable to add command: %v", err)
			}

			if err := app.Run(naistrix.RunWithArgs(append([]string{"test"}, tt.flags...))); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if actual := buf.String(); !strings.Contains(actual, tt.expected) {
				t.Errorf("expected output to contain %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestOutputWriter_OutputStyles(t *testing.T) {
	pterm.DisableStyling()
	defer pterm.EnableStyling()

	var buf bytes.Buffer
	app, _, err := naistrix.NewApplication("app", "title", "v0.0.0",
		naistrix.ApplicationWithWriter(&buf), naistrix.ApplicationWithDiagnosticWriter(&buf))
	if err != nil {
		t.Fatalf("unable to create application: %v", err)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "test",
		Title: "Test command",
		RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.
				Successf("some success\n").
				Successln("more", "success").
				Infof("some info\n").
				Infoln("more", "info").
				Warnf("some warning\n").
				Warnln("more", "warning").
				Errorf("some error\n").
				Errorln("more", "error").
				Println("An <info>informational</info> message.").
				Println("A <warn>warning</warn> message.").
				Println("An <error>error</error> message.").
				Println("Some <info>info</info>, a <warn>warning</warn> and an <error>error</error>.")

			return nil
		},
	})
	if err != nil {
		t.Fatalf("unable to add command: %v", err)
	}

	if err := app.Run(naistrix.RunWithArgs([]string{"test"})); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	expectedSubstrings := []string{
		"SUCCESS: some success",
		"SUCCESS: more success",
		"INFO: some info",
		"INFO: more info",
		"WARNING: some warning",
		"WARNING: more warning",
		"ERROR: some error",
		"ERROR: more error",
		"An informational message.",
		"A warning message.",
		"An error message.",
		"Some info, a warning and an error.",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("expected output to contain %q, got %q", substr, output)
		}
	}
}
