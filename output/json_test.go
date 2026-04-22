package output_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/nais/naistrix"
	"github.com/nais/naistrix/output"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name           string
		jsonOpts       []output.JSONOptionFunc
		dataToRender   any
		expectedOutput string
	}{
		{
			name:           "render data",
			jsonOpts:       nil,
			dataToRender:   "some data",
			expectedOutput: "\"some data\"\n",
		},
		{
			name: "pretty output",
			jsonOpts: []output.JSONOptionFunc{
				output.JSONWithPrettyOutput(),
				output.JSONWithIndentChar(" "),
			},
			dataToRender: map[string]any{"foo": "bar", "baz": 42, "quux": []string{"a", "b"}},
			expectedOutput: heredoc.Doc(`
				{
				 "baz": 42,
				 "foo": "bar",
				 "quux": [
				  "a",
				  "b"
				 ]
				}
			`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0", naistrix.ApplicationWithWriter(&buf))
			if err != nil {
				t.Fatalf("unable to create application: %v", err)
			}

			err = app.AddCommand(&naistrix.Command{
				Name:  "test",
				Title: "Some title",
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					return out.JSON(tt.jsonOpts...).Render(tt.dataToRender)
				},
			})
			if err != nil {
				t.Fatalf("unable to add command: %v", err)
			}

			if err := app.Run(naistrix.RunWithArgs([]string{"test"})); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if actual := buf.String(); actual != tt.expectedOutput {
				fmt.Println(actual)
				t.Fatalf("expected %q, got: %q", tt.expectedOutput, actual)
			}
		})
	}
}
