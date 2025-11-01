package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

type User struct {
	Name  name   `yaml:"name"`
	Email string `yaml:"email"`
	Age   int    `yaml:"age"`
	data  string // Unexported fields are ignored
}

type name struct {
	First string `yaml:"first"`
	Last  string `yaml:"last"`
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
		"Example application with YAML output",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(
		&naistrix.Command{
			Name:  "show",
			Title: "Show users.",
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				return out.YAML().Render(users)
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
