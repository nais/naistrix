package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/pkg/input"
)

type Team struct {
	Name    string
	Members int
}

func (t Team) String() string {
	return t.Name + " (" + strconv.Itoa(t.Members) + " members)"
}

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application demonstrating input helpers.",
		"v0.0.0",
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(
		&naistrix.Command{
			Name:  "confirm",
			Title: "Ask user for confirmation before continuing.",
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				if ok, err := input.Confirm("Are you sure?"); err != nil {
					return fmt.Errorf("unable to get confirmation: %w", err)
				} else if !ok {
					out.Println("We will not continue.")
					return nil
				}

				out.Println("Continuing operation.")
				return nil
			},
		},
		&naistrix.Command{
			Name:  "input",
			Title: "Prompt the user for a free-form string value.",
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				name, err := input.Input("What is your name?")
				if err != nil {
					return fmt.Errorf("unable to get input: %w", err)
				}

				out.Printf("Hello, %s!\n", name)
				return nil
			},
		},
		&naistrix.Command{
			Name:  "select",
			Title: "Prompt the user to select a value from a list.",
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				color, err := input.Select("Select a team", []Team{
					{Name: "team-1", Members: 4},
					{Name: "team-2", Members: 7},
					{Name: "team-3", Members: 3},
				})
				if err != nil {
					return fmt.Errorf("unable to get selection: %w", err)
				}

				out.Println("You picked:", color)
				return nil
			},
		},
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when adding commands: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when running application: %v\n", err)
		os.Exit(1)
	}
}
