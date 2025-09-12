package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/output"
)

// User represents a row in the table. The headings for the table is derived from the struct field names.
// You can override the default heading by using the `heading` struct tag.
// You can hide a column by using the `hidden:"true"` struct tag.
type User struct {
	Name  name `heading:"Full name"` // Overrides default heading "Name"
	Email string
	Age   int    `hidden:"true"` // Hidden column, shown when the output.TableWithShowHiddenColumns() option is used
	data  string // Unexported fields are ignored when rendering the table
}

// name is a custom type used to demonstrate that the fmt.Stringer interface is supported for rendering table cells.
type name struct {
	First string
	Last  string
}

func (n name) String() string {
	return n.First + " " + n.Last
}

func main() {
	users := []User{{
		Name:  name{First: "Jane", Last: "Doe"},
		Email: "jane@example.com",
		Age:   30,
		data:  "some internal data",
	}, {
		Name:  name{First: "John", Last: "Doe"},
		Email: "john@example.com",
		Age:   42,
		data:  "some other internal data",
	}}

	app := &naistrix.Application{
		Name:  "example",
		Title: "Example application",
		SubCommands: []*naistrix.Command{{
			Name:  "show",
			Title: "Show users.",
			RunFunc: func(_ context.Context, out naistrix.Output, _ []string) error {
				return out.Table().Render(users)
			},
		}, {
			Name:  "show-full",
			Title: "Show users with hidden columns.",
			RunFunc: func(_ context.Context, out naistrix.Output, _ []string) error {
				return out.Table(output.TableWithShowHiddenColumns()).Render(users)
			},
		}, {
			Name:  "show-simple",
			Title: "Render a slice of string slices as a table.",
			RunFunc: func(_ context.Context, out naistrix.Output, _ []string) error {
				data := [][]string{
					{"Name", "Email", "Age"}, // first row is used as headers
					{"Alice", "alice@example.com", "30"},
					{"Bob", "bob@example.com", "42"},
				}
				return out.Table().Render(data)
			},
		}},
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
