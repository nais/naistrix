package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

type Flags struct {
	Verbose naistrix.Count `name:"verbose" short:"v" usage:"Verbosity level. Can be repeated."`
}

type CreateFlags struct {
	*Flags
	Resources []string `name:"resources" short:"r" usage:"Resource(s) to add to the application. Can be repeated."`
}

type DeleteFlags struct {
	*Flags
	Force bool `name:"force" short:"f" usage:"Force deletion of application."`
}

func createCommand(parentFlags *Flags) *naistrix.Command {
	flags := &CreateFlags{Flags: parentFlags}
	return &naistrix.Command{
		Name:  "create",
		Args:  []naistrix.Argument{{Name: "app_name"}},
		Title: "Create an application",
		Flags: flags,
		RunFunc: func(ctx context.Context, out naistrix.Output, args []string) error {
			// ...

			fmt.Println(flags.Resources)
			out.Println("Created application:", args[0])
			return nil
		},
	}
}

func deleteCommand(parentFlags *Flags) *naistrix.Command {
	flags := &DeleteFlags{
		Flags: parentFlags,
		Force: true, // Set default value
	}
	return &naistrix.Command{
		Name:  "delete",
		Args:  []naistrix.Argument{{Name: "app_name"}},
		Title: "Delete an application",
		Flags: flags,
		RunFunc: func(ctx context.Context, out naistrix.Output, args []string) error {
			// ...

			out.Println("Deleted application:", args[0])
			return nil
		},
	}
}

func main() {
	flags := &Flags{}
	app := &naistrix.Application{
		Name:  "example",
		Title: "Example application with flags",
		SubCommands: []*naistrix.Command{
			{
				Name:  "app",
				Title: "Application commands",
				SubCommands: []*naistrix.Command{
					createCommand(flags),
					deleteCommand(flags),
				},
			},
		},
		StickyFlags: flags,
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
