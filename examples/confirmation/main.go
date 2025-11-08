package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application with user confirmation.",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "confirm",
		Title: "Ask user for confirmation before continuing.",
		RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
			if ok, err := out.Confirm("Are you sure?"); err != nil {
				return fmt.Errorf("unable to get confirmation: %w", err)
			} else if !ok {
				out.Println("We will not continue.")
				return nil
			}

			out.Println("Continuing operation.")
			return nil
		},
	})
	if err != nil {
		fmt.Printf("error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
