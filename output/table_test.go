package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nais/naistrix/output"
	"github.com/pterm/pterm"
)

func TestTable_InvalidData(t *testing.T) {
	t.Run("non-slice", func(t *testing.T) {
		var buf bytes.Buffer
		err := output.NewTable(&buf).Render("some data")

		if err == nil {
			t.Fatal("expected error")
		}

		if contains := "non-empty slice"; !strings.Contains(err.Error(), contains) {
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
	t.Run("slice of structs", func(t *testing.T) {
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
	})

	t.Run("slice of string slices", func(t *testing.T) {
		var buf bytes.Buffer
		table := output.NewTable(&buf)

		data := [][]string{
			{"Name", "Age"},
			{"Alice", "30"},
			{"Bob", "25"},
		}

		if err := table.Render(data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestTable_Margins(t *testing.T) {
	pterm.DisableStyling()
	defer pterm.EnableStyling()

	data := []struct {
		Name string `heading:"Col"`
	}{
		{Name: "Val"},
	}

	tests := []struct {
		name          string
		opt           []output.TableOptionFunc
		expectedStart string
		expectedEnd   string
	}{
		{
			name:          "no options, default behaviour",
			opt:           nil,
			expectedStart: "Col\n",
			expectedEnd:   "Val\n",
		},
		{
			name:          "top margin",
			opt:           []output.TableOptionFunc{output.TableWithTopMargin()},
			expectedStart: "\nCol",
			expectedEnd:   "Val\n",
		},
		{
			name:          "bottom margin",
			opt:           []output.TableOptionFunc{output.TableWithBottomMargin()},
			expectedStart: "Col\n",
			expectedEnd:   "al\n\n",
		},
		{
			name:          "both margins",
			opt:           []output.TableOptionFunc{output.TableWithMargins()},
			expectedStart: "\nCol",
			expectedEnd:   "al\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			table := output.NewTable(&buf, tt.opt...)

			if err := table.Render(data); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tbl := buf.String()
			if got := tbl[:4]; got != tt.expectedStart {
				t.Fatalf("expected first character in table to be %q, got: %q", tt.expectedStart, got)
			}

			if got := tbl[len(tbl)-4:]; got != tt.expectedEnd {
				t.Fatalf("expected last character in table to be %q, got: %q", tt.expectedEnd, got)
			}
		})
	}
}
