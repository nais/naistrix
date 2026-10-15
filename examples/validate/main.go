package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/nais/naistrix"
)

type registerFlags struct {
	Team string `usage:"Team to register the user in."`
}

func teamRequired(flags *registerFlags) naistrix.ValidateFlagsFunc {
	return func(context.Context) error {
		if flags.Team == "" {
			return naistrix.Errorf("team must not be empty")
		}
		return nil
	}
}

func teamNoSpaces(flags *registerFlags) naistrix.ValidateFlagsFunc {
	return func(context.Context) error {
		if strings.ContainsAny(flags.Team, " \t") {
			return naistrix.Errorf("team must not contain whitespace")
		}
		return nil
	}
}

func usernameMinLength(n int) naistrix.ValidateArgsFunc {
	return func(_ context.Context, args *naistrix.Arguments) error {
		if len(args.Get("username")) < n {
			return naistrix.Errorf("username must be at least %d characters long", n)
		}
		return nil
	}
}

func usernameLowercase() naistrix.ValidateArgsFunc {
	return func(_ context.Context, args *naistrix.Arguments) error {
		for _, r := range args.Get("username") {
			if unicode.IsUpper(r) {
				return naistrix.Errorf("username must be lowercase")
			}
		}
		return nil
	}
}

func usernameNoSpaces() naistrix.ValidateArgsFunc {
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

	flags := &registerFlags{Team: "default"}
	err = app.AddCommand(&naistrix.Command{
		Name:  "register",
		Title: "Register a new user",
		Flags: flags,
		ValidateFlagsFunc: naistrix.ValidateFlagsFuncs(
			teamRequired(flags),
			teamNoSpaces(flags),
		),
		Args: []naistrix.Argument{
			{Name: "username"},
		},
		ValidateArgsFunc: naistrix.ValidateArgsFuncs(
			usernameMinLength(3),
			usernameLowercase(),
			usernameNoSpaces(),
		),
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Registered user: %s (team: %s)\n", args.Get("username"), flags.Team)
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
