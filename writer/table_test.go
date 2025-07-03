package writer_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nais/naistrix/writer"
)

func TestTable_SingleLevel(t *testing.T) {
	var buf bytes.Buffer
	table := writer.NewTable(&buf, writer.WithColumns("First name", "Age"))

	data := [][]any{
		{"Alice", 30},
		{"Bob", 25},
	}

	err := table.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `
╭────────────┬─────╮
│ First name │ Age │
├────────────┼─────┤
│ Alice      │ 30  │
│ Bob        │ 25  │
╰────────────┴─────╯
`
	if diff := cmp.Diff(buf.String(), expected[1:]); diff != "" {
		t.Errorf("unexpected output (-got +want):\n%s", diff)
	}
}

func TestTable_WriteOnce(t *testing.T) {
	var buf bytes.Buffer
	table := writer.NewTable(&buf, writer.WithColumns("First name", "Age"))

	data := [][]any{
		{"Alice", 30},
		{"Bob", 25},
	}

	err := table.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = table.Write(data)
	if !errors.Is(err, writer.ErrWriteOnce) {
		t.Fatalf("expected error: %v, got: %v", writer.ErrWriteOnce, err)
	}
}
