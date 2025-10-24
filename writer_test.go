package naistrix_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/pterm/pterm"
)

func TestOutputWriter_ConditionalOutput(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		flags    []string
	}{
		{
			name:     "regular output",
			expected: "normal:n1 n2\n",
			flags:    []string{},
		},
		{
			name:     "verbose output",
			expected: "normal:n1 n2\nverbose:v1 v2\n",
			flags:    []string{"-v"},
		},
		{
			name:     "debug output",
			expected: "normal:n1 n2\nverbose:v1 v2\ndebug:d1 d2\n",
			flags:    []string{"-vv"},
		},
		{
			name:     "trace output",
			expected: "normal:n1 n2\nverbose:v1 v2\ndebug:d1 d2\ntrace:t1 t2\n",
			flags:    []string{"-vvv"},
		},
	}

	pterm.RawOutput = true
	defer func() { pterm.RawOutput = false }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0", naistrix.ApplicationWithWriter(&buf))
			if err != nil {
				t.Fatalf("unable to create application: %v", err)
			}

			err = app.AddCommand(&naistrix.Command{
				Name:  "test",
				Title: "Test command",
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Printf("normal:")
					out.Println("n1", "n2")

					out.Verbosef("verbose:")
					out.Verboseln("v1", "v2")

					out.Debugf("debug:")
					out.Debugln("d1", "d2")

					out.Tracef("trace:")
					out.Traceln("t1", "t2")
					return nil
				},
			})
			if err != nil {
				t.Fatalf("unable to add command: %v", err)
			}

			if err := app.Run(naistrix.RunWithArgs(append([]string{"test"}, tt.flags...))); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if actual := buf.String(); tt.expected != actual {
				t.Errorf("expected output %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestOutputWriter_OutputStyles(t *testing.T) {
	pterm.RawOutput = true
	defer func() { pterm.RawOutput = false }()

	var buf bytes.Buffer
	app, _, err := naistrix.NewApplication("app", "title", "v0.0.0", naistrix.ApplicationWithWriter(&buf))
	if err != nil {
		t.Fatalf("unable to create application: %v", err)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "test",
		Title: "Test command",
		RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Infof("some info\n")
			out.Infoln("more", "info")

			out.Warnf("some warning\n")
			out.Warnln("more", "warning")

			out.Errorf("some error\n")
			out.Errorln("more", "error")
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
		"INFO: some info",
		"INFO: more info",
		"WARNING: some warning",
		"WARNING: more warning",
		"ERROR: some error",
		"ERROR: more error",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("expected output to contain %q, got %q", substr, output)
		}
	}
}
