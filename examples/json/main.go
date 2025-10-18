package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/output"
)

type User struct {
	Name  name   `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
	data  string // Unexported fields are ignored
}

type name struct {
	First string `json:"first"`
	Last  string `json:"last"`
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

	app, _, err := naistrix.NewApplication(
		"example",
		"Example application with JSON output",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	flags := &struct {
		Pretty bool `name:"pretty" short:"p" usage:"Output pretty JSON."`
	}{}
	err = app.AddCommand(
		&naistrix.Command{
			Name:  "show",
			Title: "Show users.",
			Flags: flags,
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				var opts []output.JSONOptionFunc
				if flags.Pretty {
					opts = append(opts, output.JSONWithPrettyOutput())
				}
				return out.JSON(opts...).Render(users)
			},
		},
	)
	if err != nil {
		fmt.Printf("error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
