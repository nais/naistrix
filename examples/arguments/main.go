package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nais/naistrix"
)

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application with command arguments",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "transform",
		Title: "Transform all the words",
		Args: []naistrix.Argument{
			{Name: "func"},
			{Name: "word", Repeatable: true},
		},
		ValidateFunc: func(ctx context.Context, args []string) error {
			switch args[0] {
			case "upper", "lower":
				return nil
			default:
				return naistrix.Errorf("only 'upper' or 'lower' is allowed")
			}
		},
		RunFunc: func(ctx context.Context, out *naistrix.OutputWriter, args []string) error {
			var t func(string) string
			if args[0] == "upper" {
				t = strings.ToUpper
			} else {
				t = strings.ToLower
			}

			out.Printf("Words: ")
			words := make([]any, len(args)-1)
			for i, word := range args[1:] {
				words[i] = t(word)
			}
			out.Println(words...)
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
