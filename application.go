// Package naistrix provides an opinionated wrapper around the https://github.com/spf13/cobra library.
package naistrix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Application represents a CLI application with a set of commands.
type Application struct {
	// name is the name of the application, used as the root command in the CLI. Can not contain spaces.
	name string

	// title is the title of the application, used as a short description for the help output.
	title string

	// version is the version of the application, used in the help output.
	version string

	// writer is the output destination for the OutputWriter used in the application. Defaults to os.Stdout.
	writer io.Writer

	// output is the output writer used in the application.
	output *OutputWriter

	// flags are global flags that should be available for all subcommands of the application.
	flags *GlobalFlags

	// additionalGlobalFlags are additional global flags that should be available for all subcommands of the application.
	additionalGlobalFlags []any

	// commands are the executable commands of the application. To be able to run the application, at least one command
	// must be defined.
	commands []*Command

	// rootCommand is the internal cobra.Command that represents the application (the root command).
	rootCommand *cobra.Command

	// executedCommand is the internal cobra.Command that was executed when running the application with the Run()
	// method.
	executedCommand *cobra.Command

	// config is the Viper configuration instance used for managing application configuration.
	config *viper.Viper
}

// ApplicationOptionFunc is a function that configures an Application.
type ApplicationOptionFunc func(*Application)

// ApplicationWithWriter sets the output destination for the OutputWriter used in the application. This defaults to
// os.Stdout.
func ApplicationWithWriter(w io.Writer) ApplicationOptionFunc {
	return func(a *Application) {
		a.writer = w
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

	if v := strings.TrimSpace(version); v == "" {
		return nil, nil, fmt.Errorf("application version must not be empty")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	v := viper.New()
	v.SetEnvPrefix(strings.ToUpper(name))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	app := &Application{
		name:    name,
		title:   title,
		version: version,
		flags: &GlobalFlags{
			Config: configDir + "/." + name + "/config.yaml",
		},
		config: v,
	}

	for _, opt := range opts {
		opt(app)
	}

	if app.writer == nil {
		app.writer = os.Stdout
	}

	cobra.EnableTraverseRunHooks = true

	app.rootCommand = &cobra.Command{
		Use:                app.name,
		Short:              app.title,
		Version:            app.version,
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableSuggestions: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.initializeConfig(cmd.Flags()); err != nil {
				return fmt.Errorf("failed to initialize configuration: %w", err)
			}

			if err := syncViperToFlags(app.flags, app.config); err != nil {
				return fmt.Errorf("failed to sync sticky flags: %w", err)
			}

			for _, f := range app.additionalGlobalFlags {
				if err := syncViperToFlags(f, app.config); err != nil {
					return fmt.Errorf("failed to sync additional sticky flags: %w", err)
				}
			}

			if app.flags.NoColors {
				pterm.DisableStyling()
			}

			return nil
		},
	}
	app.rootCommand.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	app.rootCommand.SetOut(app.writer)
	app.output = NewOutputWriter(app.writer, &app.flags.VerboseLevel)

	if err := setupFlags(app.rootCommand, nil, app.flags, app.rootCommand.PersistentFlags()); err != nil {
		return nil, nil, fmt.Errorf("failed to setup application flags: %w", err)
	}

	if err := app.AddCommand(configCommand(app.config)); err != nil {
		return nil, nil, fmt.Errorf("failed to add config command: %w", err)
	}

	return app, app.flags, nil
}

// AddCommand adds one or more commands to the application.
func (a *Application) AddCommand(cmd *Command, cmds ...*Command) error {
	all := append([]*Command{cmd}, cmds...)
	a.commands = append(a.commands, all...)

	commandsAndAliases := make([]string, 0)
	usageTemplate := a.rootCommand.UsageTemplate()

	for _, c := range all {
		if c.Group != "" && !a.rootCommand.ContainsGroup(c.Group) {
			a.rootCommand.AddGroup(&cobra.Group{
				ID:    c.Group,
				Title: c.Group,
			})
		}

		if err := c.init(a.name, a.output, usageTemplate, a.config); err != nil {
			return fmt.Errorf("failed to initialize command %q: %w", c.Name, err)
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
	if err := setupFlags(a.rootCommand, nil, flags, a.rootCommand.PersistentFlags()); err != nil {
		return fmt.Errorf("failed to setup global flags: %w", err)
	}

	a.additionalGlobalFlags = append(a.additionalGlobalFlags, flags)

	return nil
}

// Run executes the application.
func (a *Application) Run(opts ...RunOptionFunc) error {
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

	var err error
	for {
		a.rootCommand.SetArgs(ro.args)
		a.executedCommand, err = a.rootCommand.ExecuteContextC(ro.ctx)
		if err == nil {
			return nil
		}

		var deprecatedErr *DeprecatedCommandError
		if errors.As(err, &deprecatedErr) {
			if len(deprecatedErr.Replacement) > 0 && deprecatedErr.ExecuteReplacement {
				ro.args = deprecatedErr.Replacement
				continue
			}

			// prepend the application name to the replacement command for the error message returned to the user
			deprecatedErr.Replacement = append([]string{a.name}, deprecatedErr.Replacement...)
		}

		return err
	}
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

// initializeConfig initializes the configuration for the application using Viper. It reads the configuration file
// specified by the global --config flag.
func (a *Application) initializeConfig(flags *pflag.FlagSet) error {
	p, err := resolveHomeDir(a.flags.Config)
	if err != nil {
		return fmt.Errorf("failed to resolve home directory in config file path: %w", err)
	}

	a.flags.Config = p
	a.config.SetConfigFile(a.flags.Config)
	a.output.Debugf("Initializing configuration using file %q\n", a.flags.Config)

	if err := a.config.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
		a.output.Debugln("The specified configuration file does not exist")
	}

	if err := a.config.BindPFlags(flags); err != nil {
		return fmt.Errorf("failed to bind flags to configuration: %w", err)
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

// resolveHomeDir resolves the home directory in the given path if it starts with "~/".
func resolveHomeDir(path string) (string, error) {
	if len(path) > 1 && path[:2] == "~/" {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		path = filepath.Join(u.HomeDir, path[2:])
	}
	return path, nil
}
