package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

func main() {
	app := &naistrix.Application{
		Name:  "example",
		Title: "Example application",
		SubCommands: []*naistrix.Command{
			{
				Name:  "greet",
				Title: "Greet the user",
				Args:  []naistrix.Argument{{Name: "user_name"}},
				RunFunc: func(_ context.Context, out naistrix.Output, args []string) error {
					out.Println("Hello, " + args[0] + "!")
					return nil
				},
			},
		},
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
