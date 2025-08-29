package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/nais/naistrix/output"
)

func TestJSON(t *testing.T) {
	t.Run("render data", func(t *testing.T) {
		var buf bytes.Buffer
		err := output.NewJSON(&buf).Render("some data")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := `"some data"`
		if strings.TrimSpace(buf.String()) != expected {
			t.Fatalf("expected %q, got: %q", expected, buf.String())
		}
	})

	t.Run("prettify", func(t *testing.T) {
		var buf bytes.Buffer
		err := output.
			NewJSON(
				&buf,
				output.JSONWithPrettyOutput(),
				output.JSONWithIndentChar("    "),
			).
			Render([]string{"foo", "bar"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := heredoc.Doc(`
			[
			    "foo",
			    "bar"
			]
		`)
		if buf.String() != expected {
			t.Fatalf("expected %q, got: %q", expected, buf.String())
		}
	})
}
