// Package naistrix provides an opinionated wrapper around the https://github.com/spf13/cobra library.
package naistrix

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Application represents a CLI application with a set of commands.
type Application struct {
	// Name is the name of the application, used as the root command in the CLI. Can not contain spaces.
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

	// ctx is the context used when running the application with the Run() method. If not set it defaults to
	// context.Background().
	ctx context.Context

	// out is the output destination used when running the application with the Run() method. If not set it defaults to
	// Stdout().
	out Output

	// args are the command line arguments passed to the application when running it with the Run() method. If not set
	// it defaults to os.Args[1:].
	args []string

	// executedCommand is the internal cobra.Command that was executed when running the application with the Run()
	// method.
	executedCommand *cobra.Command
}

// RunOptionFunc is a function that can be used to set options for the application when running it.
type RunOptionFunc func(*Application)

// RunWithContext sets the context for the application. The default context is context.Background().
func RunWithContext(ctx context.Context) RunOptionFunc {
	return func(a *Application) {
		a.ctx = ctx
	}
}

// RunWithOutput sets the output destination for the application. The default output is Stdout().
func RunWithOutput(out Output) RunOptionFunc {
	return func(a *Application) {
		a.out = out
	}
}

// RunWithArgs sets the command line arguments for the application. The default is os.Args[1:].
func RunWithArgs(args []string) RunOptionFunc {
	return func(a *Application) {
		a.args = args
	}
}

// Run executes the application. Validation of the application along with the validation of the commands is performed
// before executing the command's RunFunc. The method returns the names of the executed command and its parent commands
// as a slice of strings, or an error if the command execution fails.
func (a *Application) Run(opts ...RunOptionFunc) error {
	if err := a.validate(); err != nil {
		panic(err.Error())
	}

	for _, opt := range opts {
		opt(a)
	}

	if a.ctx == nil {
		a.ctx = context.Background()
	}

	if a.out == nil {
		a.out = Stdout()
	}

	if a.args == nil {
		a.args = os.Args[1:]
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
	a.cobraCmd.SetArgs(a.args)
	a.cobraCmd.SetOut(a.out)
	a.cobraCmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)

	setupFlags(a.cobraCmd, a.StickyFlags, a.cobraCmd.PersistentFlags())

	for group := range allGroups(a.SubCommands) {
		a.cobraCmd.AddGroup(&cobra.Group{
			ID:    group,
			Title: group,
		})
	}

	commandsAndAliases := make([]string, 0)
	usageTemplate := a.cobraCmd.UsageTemplate()
	for _, sub := range a.SubCommands {
		sub.init(a.Name, a.out, usageTemplate)
		a.cobraCmd.AddCommand(sub.cobraCmd)

		commandsAndAliases = append(commandsAndAliases, sub.Name)
		commandsAndAliases = append(commandsAndAliases, sub.Aliases...)
	}

	if d := duplicate(commandsAndAliases); d != "" {
		panic(fmt.Sprintf("the application contains duplicate commands and/or aliases: %q", d))
	}

	executedCommand, err := a.cobraCmd.ExecuteContextC(a.ctx)
	a.executedCommand = executedCommand
	return err
}

// ExecutedCommand returns the name of the command that was executed, along with the parent command names and the
// application name. Only valid commands are included, so if the application was run with an unknown command, only
// known command names up until the unknown one are included.
func (a *Application) ExecutedCommand() []string {
	if a.executedCommand == nil {
		return nil
	}
	return strings.Split(a.executedCommand.CommandPath(), " ")
}

// validate checks that the application is valid. This is called when trying to Run() the application.
func (a *Application) validate() error {
	if name := strings.TrimSpace(a.Name); name == "" || strings.Contains(name, " ") {
		return fmt.Errorf("application name must not be empty and must not contain spaces, got: %q", a.Name)
	}

	if len(a.SubCommands) == 0 {
		return fmt.Errorf("the application must have at least one command to be able to run")
	}

	return nil
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
