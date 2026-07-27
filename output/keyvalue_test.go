package output_test

import (
	"bytes"
	"testing"

	"github.com/nais/naistrix/output"
)

func TestKeyValue(t *testing.T) {
	tests := []struct {
		name           string
		dataToRender   map[string]any
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "render single key-value pair",
			dataToRender:   map[string]any{"key": "value"},
			expectedOutput: "key=value\n",
		},
		{
			name:           "render multiple key-value pairs sorted",
			dataToRender:   map[string]any{"foo": "bar", "baz": "qux"},
			expectedOutput: "baz=qux\nfoo=bar\n",
		},
		{
			name:          "empty map returns error",
			dataToRender:  map[string]any{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := output.NewKeyValue(&buf)
			err := c.Render(tt.dataToRender)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if actual := buf.String(); actual != tt.expectedOutput {
				t.Fatalf("expected %q, got %q", tt.expectedOutput, actual)
			}
		})
	}
}
