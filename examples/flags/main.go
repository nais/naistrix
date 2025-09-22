package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

type GlobalFlags struct {
	Quiet bool `name:"quiet" short:"q" usage:"Suppress all output."`
}

type CreateFlags struct {
	*GlobalFlags
	Resources []string `name:"resources" short:"r" usage:"Resource(s) to add to the application. Can be repeated."`
}

type DeleteFlags struct {
	*GlobalFlags
	Force bool `name:"force" short:"f" usage:"Force deletion of application."`
}

func createCommand(globalFlags *GlobalFlags) *naistrix.Command {
	flags := &CreateFlags{GlobalFlags: globalFlags}
	return &naistrix.Command{
		Name:  "create",
		Args:  []naistrix.Argument{{Name: "app_name"}},
		Title: "Create an application",
		Flags: flags,
		RunFunc: func(ctx context.Context, out *naistrix.OutputWriter, args []string) error {
			// when entering this function, the flags variable has been mutated according to the CLI input provided by
			// the user

			out.Println("Created application:", args[0])
			return nil
		},
	}
}

func deleteCommand(globalFlags *GlobalFlags) *naistrix.Command {
	flags := &DeleteFlags{
		GlobalFlags: globalFlags,
		Force:       true, // Set default value
	}
	return &naistrix.Command{
		Name:  "delete",
		Args:  []naistrix.Argument{{Name: "app_name"}},
		Title: "Delete an application",
		Flags: flags,
		RunFunc: func(ctx context.Context, out *naistrix.OutputWriter, args []string) error {
			// when entering this function, the flags variable has been mutated according to the CLI input provided by
			// the user

			out.Println("Deleted application:", args[0])
			return nil
		},
	}
}

func main() {
	app, flags, err := naistrix.NewApplication(
		"example",
		"Example application with flags",
		"v0.0.0",
	)
	if err != nil {
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	_ = flags // Embed this if you need to access base global flags in any of your commands

	extraGlobalFlags := &GlobalFlags{}
	if err := app.AddGlobalFlags(extraGlobalFlags); err != nil {
		fmt.Printf("error when adding global flags: %v\n", err)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "app",
		Title: "Application commands",
		SubCommands: []*naistrix.Command{
			createCommand(extraGlobalFlags),
			deleteCommand(extraGlobalFlags),
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
