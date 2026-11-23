package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

type applicationFlags struct {
	ErrorFormat string `name:"error-format" usage:"Error format: text, json, or yaml."`
}

func run(flags *applicationFlags) error {
	app, _, err := naistrix.NewApplication("example", "Example application with structured errors", "v0.0.0")
	if err != nil {
		return err
	}
	if err := app.AddGlobalFlags(flags); err != nil {
		return err
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "fail",
		Title: "Demonstrate a failure without changing any resources.",
		RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
			return naistrix.NewCodedError("permission_denied", "You cannot update this resource.", nil)
		},
	}); err != nil {
		return err
	}
	return app.Run()
}

func main() {
	flags := &applicationFlags{ErrorFormat: string(naistrix.ErrorFormatJSON)}
	if err := run(flags); err != nil {
		format := naistrix.ErrorFormat(flags.ErrorFormat)
		switch format {
		case naistrix.ErrorFormatText, naistrix.ErrorFormatJSON, naistrix.ErrorFormatYAML:
		default:
			_, _ = fmt.Fprintf(os.Stderr, "invalid --error-format %q: choose text, json, or yaml\n", format)
			os.Exit(1)
		}
		if renderErr := naistrix.RenderError(os.Stderr, err, format); renderErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot render error: %v\n", renderErr)
		}
		os.Exit(1)
	}
}
