package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nais/naistrix/output"
)

func TestTable_InvalidData(t *testing.T) {
	t.Run("non-slice", func(t *testing.T) {
		var buf bytes.Buffer
		err := output.NewTable(&buf).Render("some data")

		if err == nil {
			t.Fatal("expected error")
		}

		if contains := "must be a slice"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error to contain %q, got: %v", contains, err)
		}
	})

	t.Run("no visible struct fields", func(t *testing.T) {
		var buf bytes.Buffer

		data := []struct {
			name string
			age  int
		}{
			{name: "Alice", age: 30},
			{name: "Bob", age: 25},
		}

		err := output.NewTable(&buf).Render(data)

		if err == nil {
			t.Fatal("expected error")
		}

		if contains := "no visible fields"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error to contain %q, got: %v", contains, err)
		}
	})
}

func TestTable_ValidData(t *testing.T) {
	var buf bytes.Buffer
	table := output.NewTable(&buf, output.TableWithShowHiddenColumns())

	data := []struct {
		Name string `heading:"Full Name" hidden:"false"`
		Age  int    `hidden:"true"`
	}{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	if err := table.Render(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
