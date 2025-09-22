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
		"Example application",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "run",
		Title: "Run a command",
		RunFunc: func(_ context.Context, out *naistrix.OutputWriter, args []string) error {
			// Messages with labels / colors

			out.Infoln("An informational message.")
			out.Warnln("A warning message.")
			out.Errorln("An error message.")

			// Output based on verbosity levels

			out.Verboseln("A verbose message, only shown when the application is run with -v or more.")
			out.Debugln("A debug message, only shown when the application is run with -vv or more.")
			out.Traceln("A trace message, only shown when the application is run with -vvv or more.")

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
