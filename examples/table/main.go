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
	Name  string `heading:"Full name"` // Overrides default heading "Name"
	Email string
	Age   int    `hidden:"true"` // Hidden column, shown when the output.TableWithShowHiddenColumns() option is used
	data  string // Unexported fields are ignored
}

func main() {
	users := []User{{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
		data:  "some internal data",
	}, {
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   42,
		data:  "some other internal data",
	}}

	app := &naistrix.Application{
		Name:  "example",
		Title: "Example application",
		SubCommands: []*naistrix.Command{{
			Name:  "list",
			Title: "List users.",
			RunFunc: func(_ context.Context, out naistrix.Output, _ []string) error {
				return out.Table().Render(users)
			},
		}, {
			Name:  "list-full",
			Title: "List users with hidden columns.",
			RunFunc: func(_ context.Context, out naistrix.Output, _ []string) error {
				return out.Table(output.TableWithShowHiddenColumns()).Render(users)
			},
		}},
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
