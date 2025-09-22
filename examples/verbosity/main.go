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
			out.Println("Some regular message, always shown.")
			out.Verboseln("Some verbose message, shown with -v or more.")
			out.Debugln("Some debug message, shown with -vv or more.")
			out.Traceln("Some trace message, shown with -vvv or more.")
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
