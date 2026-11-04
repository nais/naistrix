// Package naistrix provides an opinionated wrapper around the https://github.com/spf13/cobra library.
package naistrix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/nais/naistrix/internal/color"
	"github.com/nais/naistrix/internal/configfile"
	"github.com/nais/naistrix/internal/terminal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ErrApplicationAlreadyRun is returned when [Application.Run] is called more than once on the same application.
var ErrApplicationAlreadyRun = errors.New("application has already been run")

// Application represents a CLI application with a set of commands.
// An application is single-use: call [Application.Run] at most once, even if it returns an error.
// Applications are not safe for concurrent use.
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

	// started records whether Run has been called, regardless of its result.
	started bool

	// executedCommand is the internal cobra.Command that was executed when running the application with the Run()
	// method.
	executedCommand *cobra.Command

	// defaultsCommandName is the name of the defaults command.
	defaultsCommandName string

	// defaultDefinitions documents configuration keys without restricting which keys may be set.
	defaultDefinitions []DefaultDefinition

	// defaultsCommandSkipPreRunHooks names hooks to skip for the built-in defaults subtree.
	defaultsCommandSkipPreRunHooks []string

	// config is the Viper configuration instance used for managing application configuration.
	config *viper.Viper

	// completionPreparers selects synchronization and validation for the actual target of inherited flag callbacks.
	completionPreparers completionPreparers

	// commandDefinitions maps executable Cobra commands to their public definitions.
	commandDefinitions map[*cobra.Command]*Command

	// preRunHooks run before command validation and argument resolution.
	preRunHooks []PreRunHook
}

// ApplicationOptionFunc is a function that configures an [Application].
type ApplicationOptionFunc func(*Application)

// topLevelAliases is a map of command names to the command it points to
type topLevelAliases map[string]*Command

// ApplicationWithWriter sets the output destination for the [OutputWriter] used in the application. This defaults to
// [os.Stdout].
func ApplicationWithWriter(w io.Writer) ApplicationOptionFunc {
	return func(a *Application) {
		a.writer = w
	}
}

// ApplicationWithDefaultsCommandName sets the name of the "defaults" command.
func ApplicationWithDefaultsCommandName(name string) ApplicationOptionFunc {
	return func(a *Application) {
		a.defaultsCommandName = name
	}
}

// ApplicationWithDefaultsDocumentation appends documentation for configuration keys. Definitions do not set values or
// restrict accepted keys. Titles are shown by defaults list and shell completion; defaults describe and generated docs
// also show detailed descriptions. Each documented key must match a registered flag; Run validates this after all flags
// have been added.
func ApplicationWithDefaultsDocumentation(definitions ...DefaultDefinition) ApplicationOptionFunc {
	return func(a *Application) {
		a.defaultDefinitions = append(a.defaultDefinitions, definitions...)
	}
}

// ApplicationWithDefaultsCommandSkipPreRunHooks appends hook names to skip for the built-in defaults command and all
// its subcommands.
func ApplicationWithDefaultsCommandSkipPreRunHooks(names ...string) ApplicationOptionFunc {
	return func(a *Application) {
		a.defaultsCommandSkipPreRunHooks = append(a.defaultsCommandSkipPreRunHooks, names...)
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

// RunOptionFunc is a function used to control options for the [Application.Run] method.
type RunOptionFunc func(*runOptions)

// RunWithContext sets the context for the application. The default context is [context.Background].
func RunWithContext(ctx context.Context) RunOptionFunc {
	return func(ro *runOptions) {
		ro.ctx = ctx
	}
}

// RunWithArgs sets the command line arguments for the application. The default is [os.Args] without the program name.
func RunWithArgs(args []string) RunOptionFunc {
	return func(ro *runOptions) {
		ro.args = args
	}
}

// NewApplication creates a new [Application] with the given name, title and version. Use the available
// [ApplicationOptionFunc] functions to configure the application to your needs.
// Create a new application for each CLI invocation, including in tests.
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

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	oldUserConfigDir := userConfigDir + "/." + name
	userConfigDir += "/" + name

	// Clean up old userConfigDir/.nais, and move files to new userConfigDir
	configfile.MigrateLegacy(userConfigDir, oldUserConfigDir, "config.yaml")

	v := viper.New()
	v.SetEnvPrefix(strings.ToUpper(name))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	app := &Application{
		name:    name,
		title:   title,
		version: version,
		flags: &GlobalFlags{
			Config: userConfigDir + "/config.yaml",
		},
		config:              v,
		defaultsCommandName: "defaults",
		completionPreparers: make(completionPreparers),
		commandDefinitions:  make(map[*cobra.Command]*Command),
	}

	for _, opt := range opts {
		opt(app)
	}
	if err := validatePreRunHooks(app.preRunHooks); err != nil {
		return nil, nil, err
	}
	if err := validateDefaultDefinitions(app.defaultDefinitions); err != nil {
		return nil, nil, err
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
		DisableSuggestions: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			out := app.output
			if cmd.Name() == cobra.ShellCompRequestCmd {
				out = NewOutputWriter(io.Discard, &app.flags.VerboseLevel)
			}
			if err := app.syncConfigAndFlags(cmd.Flags(), out); err != nil {
				return err
			}

			// Disable styling if NoColors flag is set or if stdout is not a TTY
			if app.flags.NoColors || !terminal.IsTerminal(os.Stdout) {
				pterm.DisableStyling()
			}

			return app.prepareCommand(cmd)
		},
	}
	app.rootCommand.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)
	app.rootCommand.SetOut(app.writer)
	// Cobra bypasses OutputWriter when rendering help. Format the completed output rather than the command metadata,
	// so --no-colors is already parsed and documentation generation still receives the original tags.
	help := app.rootCommand.HelpFunc()
	app.rootCommand.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		var buffer bytes.Buffer
		writer := cmd.OutOrStdout()
		cmd.SetOut(&buffer)
		defer cmd.SetOut(writer)
		help(cmd, args)

		text := buffer.String()
		if app.flags.NoColors || !terminal.IsTerminal(os.Stdout) {
			text = color.StripTags(text)
		} else {
			text = color.Colorize(text)
		}
		if _, err := io.WriteString(writer, text); err != nil {
			cmd.PrintErrln(err)
		}
	})
	app.output = NewOutputWriter(app.writer, &app.flags.VerboseLevel)
	app.completionPreparers[app.rootCommand] = completionPreparation{syncFlags: app.syncConfigAndFlagsForCompletion}

	if err := setupFlags(app.rootCommand, nil, app.flags, app.rootCommand.PersistentFlags(), app.completionPreparers); err != nil {
		return nil, nil, fmt.Errorf("failed to setup application flags: %w", err)
	}

	defaults := defaultsCommand(app.defaultsCommandName, app.config, app.defaultDefinitions)
	defaults.SkipPreRunHooks = app.defaultsCommandSkipPreRunHooks
	if err := app.AddCommand(defaults); err != nil {
		return nil, nil, fmt.Errorf("failed to add defaults command: %w", err)
	}

	return app, app.flags, nil
}

// AddCommand adds one or more commands to the application.
func (a *Application) AddCommand(cmd *Command, cmds ...*Command) error {
	all := append([]*Command{cmd}, cmds...)
	for _, c := range all {
		if err := validateCommandPreRunHooks(c, a.name, a.preRunHooks); err != nil {
			return err
		}
	}
	a.commands = append(a.commands, all...)

	namesAndAliases := make([]string, 0)
	for _, c := range a.commands {
		namesAndAliases = append(namesAndAliases, c.Name)
		namesAndAliases = append(namesAndAliases, c.Aliases...)
	}

	if d := duplicate(namesAndAliases); d != "" {
		return fmt.Errorf("the application contains duplicate commands and/or aliases: %q", d)
	}

	usageTemplate := a.rootCommand.UsageTemplate()
	aliases := make(topLevelAliases)

	for _, c := range all {
		if c.Group != "" && !a.rootCommand.ContainsGroup(c.Group) {
			a.rootCommand.AddGroup(&cobra.Group{
				ID:    c.Group,
				Title: c.Group,
			})
		}

		if c.TopLevelAliases != nil {
			return fmt.Errorf("command %q is not allowed to have top-level aliases", c.Name)
		}

		if err := c.init(a.name, a.output, usageTemplate, a.config, a.syncConfigAndFlagsForCompletion, nil, a.completionPreparers); err != nil {
			return fmt.Errorf("failed to initialize command %q: %w", c.Name, err)
		}

		a.rootCommand.AddCommand(c.cobraCmd)
		a.registerCommandDefinitions(c)

		if err := collectTopLevelAliases(c, aliases); err != nil {
			return fmt.Errorf("failed to collect top-level aliases: %w", err)
		}
	}

	if len(aliases) > 0 {
		if err := a.registerTopLevelAliases(aliases); err != nil {
			return fmt.Errorf("failed to register top-level aliases: %w", err)
		}
	}

	return nil
}

// Output returns the [OutputWriter] used in the application.
func (a *Application) Output() *OutputWriter {
	return a.output
}

// registerTopLevelAliases registers the provided top-level aliases as regular commands in the application. If there are
// duplicate aliases an error will be returned.
func (a *Application) registerTopLevelAliases(aliases topLevelAliases) error {
	namesAndAliases := slices.Collect(maps.Keys(aliases))
	for _, c := range a.commands {
		namesAndAliases = append(namesAndAliases, c.Name)
		namesAndAliases = append(namesAndAliases, c.Aliases...)
	}

	if d := duplicate(namesAndAliases); d != "" {
		return fmt.Errorf("the application contains duplicate commands and/or aliases: %q", d)
	}

	for alias, cmd := range aliases {
		a.rootCommand.AddCommand(a.createAliasCommand(alias, cmd))
	}

	return nil
}

// createAliasCommand creates a cobra command that acts as a top-level alias to another command.
func (a *Application) createAliasCommand(alias string, c *Command) *cobra.Command {
	short := fmt.Sprintf("Alias for %s", c.cobraCmd.CommandPath())
	return &cobra.Command{
		Use:                alias,
		Short:              short,
		Long:               short + "\n\nRefer to the original command for more details.",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := strings.Split(c.cobraCmd.CommandPath(), " ")[1:] // skip the CLI name
			a.rootCommand.SetArgs(append(path, args...))
			_, err := a.rootCommand.ExecuteContextC(cmd.Context())
			return err
		},
	}
}

// AddGlobalFlags adds global flags to the application. These flags will be available for all subcommands of the
// application. The passed flags must be a pointer to a struct where each field represents a flag.
func (a *Application) AddGlobalFlags(flags any) error {
	if err := setupFlags(a.rootCommand, nil, flags, a.rootCommand.PersistentFlags(), a.completionPreparers); err != nil {
		return fmt.Errorf("failed to setup global flags: %w", err)
	}

	a.additionalGlobalFlags = append(a.additionalGlobalFlags, flags)

	return nil
}

// Run executes the application. It may be called only once per application, even if it returns an error.
// Subsequent calls return [ErrApplicationAlreadyRun] without executing commands or applying options.
func (a *Application) Run(opts ...RunOptionFunc) error {
	if a.started {
		return ErrApplicationAlreadyRun
	}
	a.started = true
	if err := a.validateDocumentedFlags(); err != nil {
		return err
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

	var err error
	for {
		a.rootCommand.SetArgs(ro.args)
		a.executedCommand, err = a.rootCommand.ExecuteContextC(ro.ctx)
		if err == nil {
			return nil
		}

		if deprecatedErr, ok := errors.AsType[*DeprecatedCommandError](err); ok {
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

// validateDocumentedFlags runs after registration so documentation can refer to flags added after NewApplication.
func (a *Application) validateDocumentedFlags() error {
	if len(a.defaultDefinitions) == 0 {
		return nil
	}
	flags := make(map[string]bool)
	collectFlag := func(flag *pflag.Flag) {
		flags[strings.ToLower(flag.Name)] = true
	}
	var collect func(*cobra.Command)
	collect = func(command *cobra.Command) {
		command.Flags().VisitAll(collectFlag)
		command.PersistentFlags().VisitAll(collectFlag)

		command.Flags()
		for _, child := range command.Commands() {
			collect(child)
		}
	}
	collect(a.rootCommand)
	for _, definition := range a.defaultDefinitions {
		if !flags[strings.ToLower(definition.Key)] {
			return fmt.Errorf("documented configuration key %q does not match any registered flag", definition.Key)
		}
	}
	return nil
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
func (a *Application) initializeConfig(flags *pflag.FlagSet, out *OutputWriter) error {
	p, err := configfile.ResolveHomeDir(a.flags.Config)
	if err != nil {
		return fmt.Errorf("failed to resolve home directory in config file path: %w", err)
	}

	a.flags.Config = p
	a.config.SetConfigFile(a.flags.Config)
	out.Debugf("Initializing configuration using file %q\n", a.flags.Config)

	if err := a.config.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
		out.Debugln("The specified configuration file does not exist")
	}

	if err := a.config.BindPFlags(flags); err != nil {
		return fmt.Errorf("failed to bind flags to configuration: %w", err)
	}

	return nil
}

// syncConfigAndFlagsForCompletion reloads configuration and global flags after Cobra parses the target command's flags.
// It deliberately avoids execution hooks and ordinary command output.
func (a *Application) syncConfigAndFlagsForCompletion(cmd *cobra.Command) error {
	return a.syncConfigAndFlags(cmd.Flags(), NewOutputWriter(io.Discard, &a.flags.VerboseLevel))
}

// syncConfigAndFlags loads configuration and synchronizes global flags for execution or completion.
func (a *Application) syncConfigAndFlags(parsedFlags *pflag.FlagSet, out *OutputWriter) error {
	if err := a.initializeConfig(parsedFlags, out); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	if err := syncViperToFlags(a.flags, a.config, parsedFlags); err != nil {
		return fmt.Errorf("failed to sync sticky flags: %w", err)
	}

	for _, flags := range a.additionalGlobalFlags {
		if err := syncViperToFlags(flags, a.config, parsedFlags); err != nil {
			return fmt.Errorf("failed to sync additional sticky flags: %w", err)
		}
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

// collectTopLevelAliases walks through the command and its children and stores top-level aliases in the supplied
// aliases map. Whenever a duplicate alias is found, an error is returned.
func collectTopLevelAliases(cmd *Command, aliases topLevelAliases) error {
	for _, alias := range cmd.TopLevelAliases {
		if c, exists := aliases[alias]; exists {
			return fmt.Errorf("command %q wants to register a top-level alias %q that is already registered by %q", cmd.Name, alias, c.Name)
		}

		aliases[alias] = cmd
	}

	for _, sub := range cmd.SubCommands {
		if err := collectTopLevelAliases(sub, aliases); err != nil {
			return err
		}
	}

	return nil
}
