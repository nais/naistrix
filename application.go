// Package naistrix provides an opinionated wrapper around the https://github.com/spf13/cobra library.
package naistrix

import (
	"context"
	"fmt"
	"iter"
	"maps"

	"github.com/spf13/cobra"
)

// Application represents a CLI application with a set of commands.
type Application struct {
	// Name is the name of the application, used as the root command in the CLI.
	Name string

	// Title is the title of the application, used as a short description for the help output.
	Title string

	// Version is the version of the application, used in the help output.
	Version string

	// StickyFlags are flags that should be available for all subcommands of the application.
	StickyFlags any

	// SubCommands are the executable commands of the application. To be able to run the application, at least one
	// command must be defined.
	SubCommands []*Command

	// cobraCmd is the internal cobra.Command that represents the application (the root command).
	cobraCmd *cobra.Command
}

// Run executes the application with the provided context, output writer, and command-line arguments. Validation of the
// application along with the validation of the commands is performed before executing the command's RunFunc. The value
// of args should in most cases be os.Args[1:], but can be overridden for testing purposes. The method returns the names
// of the executed command and its parent commands as a slice of strings, or an error if the command execution fails.
func (a *Application) Run(ctx context.Context, out Output, args []string) ([]string, error) {
	if len(a.SubCommands) == 0 {
		panic("the application must have at least one command to be able to run")
	}

	cobra.EnableTraverseRunHooks = true

	a.cobraCmd = &cobra.Command{
		Use:                a.Name,
		Short:              a.Title,
		Version:            a.Version,
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableSuggestions: true,
	}
	a.cobraCmd.SetArgs(args)
	a.cobraCmd.SetOut(out)

	setupFlags(a.cobraCmd, a.StickyFlags, a.cobraCmd.PersistentFlags())

	for group := range allGroups(a.SubCommands) {
		a.cobraCmd.AddGroup(&cobra.Group{
			ID:    group,
			Title: group,
		})
	}

	commandsAndAliases := make([]string, 0)
	for _, sub := range a.SubCommands {
		sub.init(a.Name, out)
		a.cobraCmd.AddCommand(sub.cobraCmd)

		commandsAndAliases = append(commandsAndAliases, sub.Name)
		commandsAndAliases = append(commandsAndAliases, sub.Aliases...)
	}

	if d := duplicate(commandsAndAliases); d != "" {
		panic(fmt.Sprintf("the application contains duplicate commands and/or aliases: %q", d))
	}

	executedCommand, err := a.cobraCmd.ExecuteContextC(ctx)
	return commandNames(executedCommand), err
}

// duplicate returns the first duplicate value found in the provided slice, or an empty string if no duplicates are
// found.
func duplicate(values []string) string {
	seen := make(map[string]struct{})
	for _, v := range values {
		if _, exists := seen[v]; exists {
			return v
		}
		seen[v] = struct{}{}
	}
	return ""
}

// allGroups returns a sequence of all unique command groups from the provided commands and their subcommands.
func allGroups(cmds []*Command) iter.Seq[string] {
	var rec func(cmds []*Command, groups map[string]struct{})
	rec = func(cmds []*Command, groups map[string]struct{}) {
		for _, cmd := range cmds {
			if cmd.Group != "" {
				groups[cmd.Group] = struct{}{}
			}
			rec(cmd.SubCommands, groups)
		}
	}

	groups := make(map[string]struct{})
	rec(cmds, groups)

	return maps.Keys(groups)
}

// commandNames returns the names of the command and all its parent commands as a slice of strings.
func commandNames(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}

	return append(commandNames(cmd.Parent()), cmd.Name())
}
