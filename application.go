// Package naistrix provides an opinionated wrapper around the https://github.com/spf13/cobra library.
package naistrix

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// Application represents a CLI application with a set of commands.
type Application struct {
	// name is the name of the application, used as the root command in the CLI. Can not contain spaces.
	name string

	// title is the title of the application, used as a short description for the help output.
	title string

	// version is the version of the application, used in the help output.
	version string

	// output is the output writer used in the application.
	output *OutputWriter

	// flags are global flags that should be available for all subcommands of the application.
	flags *GlobalFlags

	// commands are the executable commands of the application. To be able to run the application, at least one command
	// must be defined.
	commands []*Command

	// rootCommand is the internal cobra.Command that represents the application (the root command).
	rootCommand *cobra.Command

	// executedCommand is the internal cobra.Command that was executed when running the application with the Run()
	// method.
	executedCommand *cobra.Command
}

// ApplicationOptionFunc is a function that configures an Application.
type ApplicationOptionFunc func(*Application)

// ApplicationWithWriter sets the output destination for the output writer used in the application. This defaults to
// os.Stdout.
func ApplicationWithWriter(w io.Writer) ApplicationOptionFunc {
	return func(a *Application) {
		a.output.writer = w
	}
}

// runOptions holds options for running the application with the Run() method, and is manipulated via RunOptionFunc
// functions.
type runOptions struct {
	// ctx is the context used when running the application with the Run() method. If not set it defaults to
	// context.Background().
	ctx context.Context

	// args are the command line arguments passed to the application when running it with the Run() method. If not set
	// it defaults to os.Args[1:].
	args []string
}

type RunOptionFunc func(*runOptions)

// RunWithContext sets the context for the application. The default context is context.Background().
func RunWithContext(ctx context.Context) RunOptionFunc {
	return func(ro *runOptions) {
		ro.ctx = ctx
	}
}

// RunWithArgs sets the command line arguments for the application. The default is os.Args[1:].
func RunWithArgs(args []string) RunOptionFunc {
	return func(ro *runOptions) {
		ro.args = args
	}
}

// NewApplication creates a new Application with the given name, title and version. Use the available
// ApplicationOptionFunc functions to configure the application to your needs.
func NewApplication(name, title, version string, opts ...ApplicationOptionFunc) (*Application, *GlobalFlags, error) {
	if n := strings.TrimSpace(name); n == "" || strings.Contains(n, " ") {
		return nil, nil, fmt.Errorf("application name must not be empty and must not contain spaces, got: %q", name)
	}

	if t := strings.TrimSpace(title); t == "" {
		return nil, nil, fmt.Errorf("application title must not be empty")
	}

	if !semver.IsValid(version) {
		return nil, nil, fmt.Errorf("application version must be a valid semantic version, got: %q", version)
	}

	flags := &GlobalFlags{}
	app := &Application{
		name:    name,
		title:   title,
		version: version,
		flags:   flags,
		output:  &OutputWriter{level: &flags.VerboseLevel},
	}

	for _, opt := range opts {
		opt(app)
	}

	if app.output.writer == nil {
		app.output.writer = os.Stdout
	}

	cobra.EnableTraverseRunHooks = true

	app.rootCommand = &cobra.Command{
		Use:                app.name,
		Short:              app.title,
		Version:            app.version,
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableSuggestions: true,
	}
	app.rootCommand.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	app.rootCommand.SetOut(app.output.writer)

	if err := setupFlags(app.rootCommand, app.flags, app.rootCommand.PersistentFlags()); err != nil {
		return nil, nil, fmt.Errorf("failed to setup application flags: %w", err)
	}

	return app, app.flags, nil
}

// AddCommand adds one or more commands to the application. The application must have at least one command to be able to
// run.
func (a *Application) AddCommand(cmd *Command, cmds ...*Command) error {
	all := append([]*Command{cmd}, cmds...)
	a.commands = append(a.commands, all...)

	commandsAndAliases := make([]string, 0)
	usageTemplate := a.rootCommand.UsageTemplate()

	for _, c := range all {
		if c.Group != "" && a.rootCommand.ContainsGroup(c.Group) {
			a.rootCommand.AddGroup(&cobra.Group{
				ID:    c.Group,
				Title: c.Group,
			})
		}

		if err := c.init(a.name, a.output, usageTemplate); err != nil {
			return err
		}

		a.rootCommand.AddCommand(c.cobraCmd)

		commandsAndAliases = append(commandsAndAliases, c.Name)
		commandsAndAliases = append(commandsAndAliases, c.Aliases...)
	}

	if d := duplicate(commandsAndAliases); d != "" {
		return fmt.Errorf("the application contains duplicate commands and/or aliases: %q", d)
	}

	return nil
}

// AddGlobalFlags adds global flags to the application. These flags will be available for all subcommands of the
// application. The passed flags must be a pointer to a struct where each field represents a flag.
func (a *Application) AddGlobalFlags(flags any) error {
	if err := setupFlags(a.rootCommand, flags, a.rootCommand.PersistentFlags()); err != nil {
		return fmt.Errorf("unable to add global flags: %w", err)
	}

	return nil
}

// Run executes the application. At least one command must be registered using the AddCommand method.
func (a *Application) Run(opts ...RunOptionFunc) error {
	if len(a.commands) == 0 {
		return fmt.Errorf("the application must have at least one command to be able to run")
	}

	ro := &runOptions{}
	for _, opt := range opts {
		opt(ro)
	}

	if ro.ctx == nil {
		ro.ctx = context.Background()
	}

	if ro.args == nil {
		ro.args = os.Args[1:]
	}

	a.rootCommand.SetArgs(ro.args)

	var err error
	a.executedCommand, err = a.rootCommand.ExecuteContextC(ro.ctx)

	return err
}

// ExecutedCommand returns the name of the command that was executed, along with the parent command names and the
// application name. Only valid commands are included, so if the application was run with an unknown command, only
// known command names up until the unknown one are included. Will return nil if the application has not been run yet.
func (a *Application) ExecutedCommand() []string {
	if a.executedCommand == nil {
		return nil
	}
	return strings.Split(a.executedCommand.CommandPath(), " ")
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
