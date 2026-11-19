package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

func run() error {
	app, _, err := naistrix.NewApplication(
		"example", "Example application with structured errors", "v0.0.0",
		naistrix.ApplicationWithUsageOnError(false),
	)
	if err != nil {
		return err
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "fail",
		Title: "Demonstrate a failure without changing any resources.",
		RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
			return &naistrix.CodedError{
				Code:    "permission_denied",
				Message: "You cannot update this resource.",
			}
		},
	}); err != nil {
		return err
	}
	return app.Run()
}

func main() {
	if err := run(); err != nil {
		if renderErr := naistrix.RenderError(os.Stderr, err, naistrix.ErrorFormatJSON); renderErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot render error: %v\n", renderErr)
		}
		os.Exit(1)
	}
}
