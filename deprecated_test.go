package naistrix

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/nais/naistrix/input"
	"github.com/spf13/cobra"
)

func TestDeprecatedCommandReplacement(t *testing.T) {
	for _, tt := range []struct {
		name       string
		deprecated *DeprecatedCommand
		wantError  error
		wantOutput string
	}{
		{
			name: "dynamic replacement uses the supplied argument",
			deprecated: DeprecatedWithReplacementFunc(func(_ context.Context, args *Arguments) []string {
				return []string{"new", args.Get("service")}
			}),
			wantError:  input.ErrNotInteractive,
			wantOutput: "new gateway",
		},
		{
			name:       "no replacement returns the deprecation error without prompting",
			deprecated: DeprecatedWithoutReplacement(),
			wantError:  ErrDeprecatedCommandWithoutReplacement,
			wantOutput: "there is no replacement command available",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			out := &OutputWriter{writer: io.Discard, diagnosticWriter: &output}
			command := &Command{
				Name:       "old",
				Args:       []Argument{{Name: "service"}},
				Deprecated: tt.deprecated,
			}
			cmd := &cobra.Command{}
			cmd.SetContext(input.WithNonInteractive(t.Context()))
			run := command.cobraRun(out, &commandRegistry{})
			if err := run(cmd, []string{"gateway"}); !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}
			if got := output.String(); !strings.Contains(got, tt.wantOutput) {
				t.Errorf("output = %q, want it to contain %q", got, tt.wantOutput)
			}
		})
	}
}
