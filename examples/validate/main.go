package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/nais/naistrix"
)

func usernameMinLength(n int) naistrix.ValidateFunc {
	return func(_ context.Context, args *naistrix.Arguments) error {
		if len(args.Get("username")) < n {
			return naistrix.Errorf("username must be at least %d characters long", n)
		}
		return nil
	}
}

func usernameLowercase() naistrix.ValidateFunc {
	return func(_ context.Context, args *naistrix.Arguments) error {
		for _, r := range args.Get("username") {
			if unicode.IsUpper(r) {
				return naistrix.Errorf("username must be lowercase")
			}
		}
		return nil
	}
}

func usernameNoSpaces() naistrix.ValidateFunc {
	return func(_ context.Context, args *naistrix.Arguments) error {
		if strings.ContainsAny(args.Get("username"), " \t") {
			return naistrix.Errorf("username must not contain whitespace")
		}
		return nil
	}
}

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application composing several validators",
		"v0.0.0",
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "register",
		Title: "Register a new user",
		Args: []naistrix.Argument{
			{Name: "username"},
		},
		ValidateFunc: naistrix.ValidateFuncs(
			usernameMinLength(3),
			usernameLowercase(),
			usernameNoSpaces(),
		),
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Println("Registered user:", args.Get("username"))
			return nil
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when running application: %v\n", err)
		os.Exit(1)
	}
}
