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
		"Example application with command groups",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(
		&naistrix.Command{
			Name:  "cmd",
			Title: "Command in Group #1.",
			Group: "Group #1",
			RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("I'm in Group #1")
				return nil
			},
		},
		&naistrix.Command{
			Name:  "cmd2",
			Title: "Another command in Group #1.",
			Group: "Group #1",
			RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("I'm also in Group #1")
				return nil
			},
		},
		&naistrix.Command{
			Name:  "other-cmd",
			Title: "Command in Group #2.",
			Group: "Group #2",
			RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("I'm in Group #2")
				return nil
			},
		},
	)
	if err != nil {
		fmt.Printf("error when adding commands: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
