package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

type LoginFlags struct {
	Secure bool `name:"secure" short:"s" usage:"Use secure login."`
}

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application using command aliases",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	loginFlags := &LoginFlags{
		Secure: true, // Set default value
	}

	err = app.AddCommand(
		&naistrix.Command{
			Name:  "auth",
			Title: "Auth commands",
			SubCommands: []*naistrix.Command{
				{
					Name:            "login",
					TopLevelAliases: []string{"login"},
					Title:           "Perform a login",
					Flags:           loginFlags,
					RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
						if loginFlags.Secure {
							out.Infoln("Using secure login...")
						} else {
							out.Warnln("Using insecure login...")
						}

						return nil
					},
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
