package naistrix

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/nais/naistrix/input"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// defaultArgumentPrompt is the text question used when an argument has no custom Prompt or Choices.
	defaultArgumentPrompt = "Enter a value"

	// defaultArgumentSelectPrompt is the selection question used when an argument has Choices but no custom Prompt.
	defaultArgumentSelectPrompt = "Select a value"

	// repeatableArgumentPrompt is the text question used after collecting the first value of a repeatable argument.
	repeatableArgumentPrompt = "Enter another value, or press Enter to finish"

	// repeatableArgumentConfirm is the confirmation format used after each repeatable choice; %s is the argument name.
	repeatableArgumentConfirm = "Add another %s?"
)

var (
	promptArgument       = input.Input
	selectArgument       = input.Select[string]
	confirmArgument      = input.Confirm
	argumentsInteractive = input.IsInteractive
)

// Argument represents a positional argument for a command. All arguments for a command will be grouped together in a
// string slice, and the arguments will be injected into the command's [RunFunc] (amongst others) in the order they are
// defined.
type Argument struct {
	// Name is the name of the argument, used for help output. This field is required.
	Name string

	// Repeatable can be used for repeatable arguments. Only the last argument for a command can be repeatable.
	Repeatable bool

	// Prompt overrides the default question shown when this argument is missing in an interactive terminal. The prompt
	// will be prefixed argument position and uppercase name, as in "#1 NAME:". Automatic choice completion also uses
	// this text as shell active help, without that prefix. Mutually exclusive with [Argument.PromptFunc].
	Prompt string

	// PromptFunc renders the question for a missing argument and the active help for automatic choice completion. As
	// with [Argument.Prompt], interactive questions are prefixed with the argument position and uppercase name. Shell
	// active help uses the returned text without that prefix.
	//
	// Flags are synchronized before it runs, and args contains only earlier validated and normalized arguments. It is
	// called only when text is needed, at most once per argument per execution or completion request. An empty result
	// uses the default question when prompting and adds no active help during completion. Mutually exclusive with
	// [Argument.Prompt].
	PromptFunc ArgumentPromptFunc

	// Choices lists the valid values for this argument. If empty, any value is allowed. When prompting, these values
	// are presented as a selectable list instead of a text prompt. They also provide shell completion unless the
	// command configures [Command.AutoCompleteFunc] or [Command.AutoCompleteExtensions]. Mutually exclusive with
	// [Argument.ChoicesFunc].
	Choices []string

	// ChoicesFunc resolves allowed values at execution time, after flags are synchronized. Its args contain only
	// earlier positional arguments, already validated and normalized. The function is called at most once per argument
	// per execution, including for values supplied on the command line. An empty result means no valid choices. It also
	// supplies automatic shell completion and may be called again for each completion request. Mutually exclusive with
	// [Argument.Choices].
	ChoicesFunc ArgumentChoicesFunc

	// ChoicesCaseInsensitive accepts [Argument.Choices] regardless of case, using Unicode case folding. Accepted values
	// are normalized to the configured choice before [ValidateFunc] and [RunFunc] receive them. Choices must be
	// distinct under case-insensitive matching. This also applies to [Argument.ChoicesFunc] results and has no effect
	// without either source.
	ChoicesCaseInsensitive bool
}

// ArgumentChoicesFunc resolves allowed values for an argument using the command context and earlier resolved arguments.
// Returned values are used for both interactive selection and validation.
type ArgumentChoicesFunc func(ctx context.Context, args *Arguments) ([]string, error)

// ArgumentPromptFunc renders argument text using the command context and earlier resolved arguments.
type ArgumentPromptFunc func(ctx context.Context, args *Arguments) (string, error)

// Command represents a command in the CLI application.
type Command struct {
	// Name is the name of the command, this is used to invoke the command in the CLI. This field is required.
	//
	// Example: "list" or "create-user".
	Name string

	// Aliases are alternative names for the command, used to invoke the command with different names on the same
	// command level.
	Aliases []string

	// TopLevelAliases are alternative names for the command, used to invoke the command with different names from the
	// top-level of the CLI. This must only be used for [Command.SubCommands], and the top-level aliases must be unique
	// across the entire CLI.
	TopLevelAliases []string

	// Title is the title of the command, used as a short description for the help output and as a header for the
	// optional [Command.Description] field. This field is required.
	Title string

	// Description is a detailed description of the command, shown in the help output. When set, it will be prefixed
	// with the [Command.Title] field.
	Description string

	// Deprecated marks the command as deprecated. When set, the commands [Command.RunFunc] is never executed, instead
	// a warning is shown to the user, and if a replacement command is specified, the user is prompted to execute the
	// replacement command instead.
	//
	// Commands with one or more [Command.SubCommands] can not be marked as deprecated. Instead, each subcommand must
	// be marked as deprecated individually.
	Deprecated *DeprecatedCommand

	// RunFunc will be executed when the command is run. The [Command.RunFunc] and [Command.SubCommands] fields are
	// mutually exclusive.
	RunFunc RunFunc

	// ValidateFunc will be executed before the command's [Command.RunFunc] is executed.
	ValidateFunc ValidateFunc

	// AutoCompleteFunc sets up a function that will be used to provide auto-completion suggestions for the command.
	// When unset, [Argument.Choices] and [Argument.ChoicesFunc] provide automatic completion.
	AutoCompleteFunc AutoCompleteFunc

	// AutoCompleteExtensions specifies which file extensions to list in autocompletion. This overrides
	// [Command.AutoCompleteFunc].
	AutoCompleteExtensions []string

	// Group places the command in a specific group. This is mainly used for grouping of commands in the help text.
	Group string

	// SubCommands adds subcommands to the command. The [Command.SubCommands] and [Command.RunFunc] fields are mutually
	// exclusive.
	SubCommands []*Command

	// Args are the positional arguments to the command. The arguments will be injected into [Command.RunFunc]. The
	// command will be validated when executed to ensure that the correct amount of arguments is specified.
	Args []Argument

	// DisableArgumentPrompts prevents interactive prompting for missing arguments to this command.
	DisableArgumentPrompts bool

	// Flags sets up flags for the command.
	Flags any

	// StickyFlags sets up flags that is persistent across all subcommands.
	StickyFlags any

	// Examples are examples of how to use the command. The examples are shown in the help output in the added order.
	Examples []Example

	// cobraCmd is the internal cobra.Command for the Command.
	cobraCmd *cobra.Command
}

// Example represents an example of how to use a command. It is used to provide examples in the help output for the
// command.
type Example struct {
	// Description is a description of the example, shown in the help output. It should be a short, concise description
	// of what the example does.
	//
	// Example: "List all members of the team."
	Description string

	// Command is the command string to be used as an example. The command name itself will be automatically prepended
	// to this string, and should not be included in the Command field.
	//
	// Example: "<arg> --flag value" will result in an example that looks like "nais command-name <arg> --flag value"
	Command string
}

// RunFunc is a function that will be executed when the command is run.
//
// The [Arguments] parameter holds the arguments specified by the user, and all output should be generated using the
// [OutputWriter].
type RunFunc func(ctx context.Context, args *Arguments, out *OutputWriter) error

// promptText returns prompt text based on a callback or a statically configured string.
func (a Argument) promptText(ctx context.Context, prefix *Arguments) (string, error) {
	if a.PromptFunc != nil {
		return a.PromptFunc(ctx, prefix)
	}
	return a.Prompt, nil
}

// promptValues collects one or more values for a missing argument at the given 1-based position.
func (a Argument) promptValues(ctx context.Context, out *OutputWriter, position int) ([]string, error) {
	name := strings.ToUpper(a.Name)
	question := a.Prompt
	if question == "" {
		question = defaultArgumentPrompt
		if len(a.Choices) > 0 {
			question = defaultArgumentSelectPrompt
		}
	}
	prefix := fmt.Sprintf("#%d %s: ", position, name)
	prompt := prefix + question
	var values []string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var value string
		var err error
		if len(a.Choices) > 0 {
			value, err = selectArgument(prompt, a.Choices)
		} else {
			value, err = promptArgument(prompt)
		}
		if err != nil {
			return nil, err
		}
		if len(a.Choices) == 0 && strings.TrimSpace(value) == "" {
			if len(values) > 0 {
				return values, nil
			}
			out.Warnln("A value is required. Please try again.")
			continue
		}
		values = append(values, value)
		if !a.Repeatable {
			return values, nil
		}
		if len(a.Choices) > 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			more, err := confirmArgument(fmt.Sprintf(repeatableArgumentConfirm, name))
			if err != nil {
				return nil, err
			}
			if !more {
				return values, nil
			}
		} else {
			prompt = prefix + repeatableArgumentPrompt
		}
	}
}

// cobraExample generates a formatted string of examples suitable for the underlying cobra.Command.
func (c *Command) cobraExample(prefix string) (string, error) {
	if len(c.Examples) == 0 {
		return "", nil
	}

	const indent = "  "

	var sb strings.Builder
	for _, ex := range c.Examples {
		description := strings.TrimSpace(ex.Description)
		if description == "" {
			return "", fmt.Errorf("example for command %q is missing description", c.Name)
		}

		cmd := prefix + " " + strings.TrimSpace(ex.Command)
		sb.WriteString(indent + "# " + description + "\n")
		sb.WriteString(indent + "$ " + cmd + "\n\n")
	}

	return indent + strings.TrimSpace(sb.String()), nil
}

// cobraUse generates the command usage string for the underlying cobra.Command.
func (c *Command) cobraUse() string {
	var cmd strings.Builder
	cmd.WriteString(c.Name)
	for _, arg := range c.Args {
		format := " %[1]s" // ARG
		if arg.Repeatable {
			format += " [%[1]s...]" // ARG [ARG...]
		}
		_, _ = fmt.Fprintf(&cmd, format, strings.ToUpper(arg.Name))
	}

	return cmd.String()
}

// validateArgs validates the positional arguments for the command, and prepends a ValidateFunc to the command that will
// make sure the correct amount of arguments is sent to the command when executed by the end-user.
func (c *Command) validateArgs() error {
	hasRepeatable := false

	for i, arg := range c.Args {
		if arg.Name == "" {
			return fmt.Errorf("argument name (%+v) cannot be empty", arg)
		}
		if arg.Prompt != "" && arg.PromptFunc != nil {
			return fmt.Errorf("argument %q cannot have both Prompt and PromptFunc", arg.Name)
		}
		if arg.ChoicesFunc != nil && arg.Choices != nil {
			return fmt.Errorf("argument %q cannot have both Choices and ChoicesFunc", arg.Name)
		}
		if err := arg.validateChoices(); err != nil {
			return err
		}

		if arg.Repeatable {
			hasRepeatable = true
			if i != len(c.Args)-1 {
				return fmt.Errorf("a repeatable argument (%+v) must be the last argument for the command", arg)
			}
		}
	}

	numArgs := len(c.Args)
	var validationFunc ValidateFunc
	if numArgs > 0 && hasRepeatable {
		validationFunc = ValidateMinArgs(numArgs)
	} else if numArgs > 0 {
		validationFunc = ValidateExactArgs(numArgs)
	}

	if validationFunc != nil {
		existingValidateFunc := c.ValidateFunc
		c.ValidateFunc = func(ctx context.Context, args *Arguments) error {
			if err := validationFunc(ctx, args); err != nil {
				return err
			}
			for _, arg := range c.Args {
				if len(arg.Choices) == 0 {
					continue
				}
				var values []string
				if arg.Repeatable {
					values = args.GetRepeatable(arg.Name)
				} else {
					values = []string{args.Get(arg.Name)}
				}
				for _, value := range values {
					if !slices.Contains(arg.Choices, value) {
						return Errorf("invalid value %q for argument %q: expected one of %s", value, arg.Name, strings.Join(arg.Choices, ", "))
					}
				}
			}

			if existingValidateFunc == nil {
				return nil
			}

			return existingValidateFunc(ctx, args)
		}
	}

	return nil
}

// cobraShort generates the short description for the cobra.Command.
func (c *Command) cobraShort() string {
	title := strings.TrimSpace(c.Title)
	if c.Deprecated != nil {
		title = "(DEPRECATED) " + title
	}

	return title
}

// cobraLong generates the long description for the cobra.Command.
func (c *Command) cobraLong() string {
	description := strings.TrimSpace(c.Description)

	if c.Deprecated != nil {
		description = strings.TrimSpace("This command is deprecated and will be removed in a future release\n\n" + description)
	}

	return strings.TrimSpace(c.cobraShort() + "\n\n" + description)
}

// cobraRun wraps the RunFunc of the command into a function that can be used by the underlying cobra.Command.
func (c *Command) cobraRun(out *OutputWriter) func(*cobra.Command, []string) error {
	if len(c.SubCommands) > 0 {
		return func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				// user tried to run a subcommand that does not exist.
				subCommands := "Available commands:\n"
				for _, s := range cmd.Commands() {
					subCommands = subCommands + "  " + s.Name() + "\n"
				}

				return fmt.Errorf(
					strings.TrimSpace(heredoc.Doc(`
						%[1]w

						Usage:
						  %[2]s <command> [flags]

						%[3]s

						Use "%[2]s -h" for more information.
					`)),
					err,
					cmd.CommandPath(),
					strings.TrimSpace(subCommands),
				)
			}

			return cmd.Help()
		}
	}

	if c.Deprecated != nil {
		c.RunFunc = func(ctx context.Context, args *Arguments, out *OutputWriter) error {
			warning := fmt.Sprintf("The command %q is deprecated and will be removed in a future release\n", c.Name)

			var replacement []string
			if c.Deprecated.replacementFunc != nil {
				replacement = c.Deprecated.replacementFunc(ctx, args)
			}

			if len(replacement) > 0 {
				warning += fmt.Sprintf("Please use the following command as a replacement:\n  %s\n",
					strings.Join(replacement, " "))
			} else {
				warning += "Unfortunately, there is no replacement command available"
			}

			out.Warnln(warning)

			if len(replacement) == 0 {
				return ErrDeprecatedCommandWithoutReplacement
			}

			executeReplacement, err := input.Confirm("Do you want to proceed with the replacement command?")
			if err != nil {
				return err
			}

			return &DeprecatedCommandError{
				Replacement:        replacement,
				ExecuteReplacement: executeReplacement,
			}
		}
	}

	return func(cmd *cobra.Command, args []string) error {
		prompt := len(args) < len(c.Args) && !c.DisableArgumentPrompts && argumentsInteractive()
		resolved, err := c.resolveArguments(cmd.Context(), args, out, prompt)
		if err != nil {
			return err
		}

		if err := c.validateInput(cmd, resolved); err != nil {
			return err
		}

		// Silence the usage for errors that might occur in the RunFunc of the command
		cmd.SilenceUsage = true
		return c.RunFunc(cmd.Context(), resolved, out)
	}
}

// validateInput runs argument count validation followed by the command's custom validation.
func (c *Command) validateInput(cmd *cobra.Command, args *Arguments) error {
	if c.ValidateFunc == nil {
		return nil
	}
	if err := c.ValidateFunc(cmd.Context(), args); err != nil {
		if e, ok := errors.AsType[Error](err); ok {
			return e
		}
		return Errorf("input validation failed: %v", err)
	}
	return nil
}

// validate checks that the command is valid.
func (c *Command) validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if strings.Contains(c.Name, " ") {
		return fmt.Errorf("command name %q contain spaces", c.Name)
	}

	if title := strings.TrimSpace(c.Title); title == "" {
		return fmt.Errorf("command %q is missing a title", c.Name)
	} else if strings.Contains(title, "\n") {
		return fmt.Errorf("title for command %q contains newline", c.Name)
	}

	if len(c.SubCommands) > 0 && c.Deprecated != nil {
		return fmt.Errorf("parent command %q can not be marked as deprecated", c.Name)
	}

	if c.RunFunc != nil && len(c.SubCommands) > 0 {
		return fmt.Errorf("command %q cannot have both RunFunc and SubCommands", c.Name)
	}

	if c.RunFunc == nil && len(c.SubCommands) == 0 && c.Deprecated == nil {
		return fmt.Errorf("either RunFunc, SubCommands or Deprecated must be set for command: %v", c.Name)
	}

	return c.validateArgs()
}

// init validates and initializes the cobra.Command.
func (c *Command) init(cmd string, out *OutputWriter, usageTemplate string, config *viper.Viper, syncInheritedConfigAndFlags syncConfigAndFlagsFunc) error {
	if err := c.validate(); err != nil {
		return err
	}

	cmd = cmd + " " + c.Name

	example, err := c.cobraExample(cmd)
	if err != nil {
		return fmt.Errorf("failed to generate examples for command %q: %w", c.Name, err)
	}
	syncConfigAndFlags := func(cmd *cobra.Command) error {
		if err := syncInheritedConfigAndFlags(cmd); err != nil {
			return err
		}
		return c.syncFlags(cmd, config)
	}

	c.cobraCmd = &cobra.Command{
		Hidden:            c.Deprecated != nil,
		Example:           example,
		Aliases:           c.Aliases,
		Use:               c.cobraUse(),
		Short:             c.cobraShort(),
		Long:              c.cobraLong(),
		GroupID:           c.Group,
		RunE:              c.cobraRun(out),
		ValidArgsFunction: c.autocomplete(syncConfigAndFlags),
		PersistentPreRunE: func(co *cobra.Command, args []string) error {
			if err := c.syncFlags(co, config); err != nil {
				return err
			}

			// Keep parent validation in the persistent hook so it also runs when a child command is executed.
			// Leaf commands validate in cobraRun after prompting for missing arguments and before RunFunc.
			if len(c.SubCommands) > 0 {
				resolved, err := c.resolveArguments(co.Context(), args, out, false)
				if err != nil {
					return err
				}
				return c.validateInput(co, resolved)
			}
			return nil
		},
	}

	if c.RunFunc == nil {
		// The internal cobraCmd will always be runnable since we are hijacking the RunE function to make sure an error
		// is returned if an unknown subcommand is invoked. Because of this the usage template will always treat the
		// command as runnable.
		c.cobraCmd.SetUsageTemplate(strings.ReplaceAll(usageTemplate, "{{if .Runnable}}", "{{if false}}"))
	} else {
		// We must set the usage template so that subcommands does not use the usage template of the parent command,
		// causing child commands to be rendered as "not runnable" even though they are.
		c.cobraCmd.SetUsageTemplate(usageTemplate)
	}

	if err := setupFlags(c.cobraCmd, c.Args, c.Flags, c.cobraCmd.Flags()); err != nil {
		return fmt.Errorf("failed to setup flags: %w", err)
	}

	if err := setupFlags(c.cobraCmd, c.Args, c.StickyFlags, c.cobraCmd.PersistentFlags()); err != nil {
		return fmt.Errorf("failed to setup persistent flags: %w", err)
	}

	commandsAndAliases := make([]string, 0)
	for _, sub := range c.SubCommands {
		if err := sub.init(cmd, out, usageTemplate, config, syncConfigAndFlags); err != nil {
			return err
		}
		c.cobraCmd.AddCommand(sub.cobraCmd)

		commandsAndAliases = append(commandsAndAliases, sub.Name)
		commandsAndAliases = append(commandsAndAliases, sub.Aliases...)
	}

	if d := duplicate(commandsAndAliases); d != "" {
		return fmt.Errorf("command %q contains duplicate commands and/or aliases: %q", cmd, d)
	}

	return nil
}

// syncFlags is shared by execution and generated completion, which does not execute command hooks.
func (c *Command) syncFlags(cmd *cobra.Command, config *viper.Viper) error {
	if err := syncViperToFlags(c.Flags, config, cmd.Flags()); err != nil {
		return fmt.Errorf("failed to sync command flags: %w", err)
	}
	if err := syncViperToFlags(c.StickyFlags, config, cmd.Flags()); err != nil {
		return fmt.Errorf("failed to sync sticky flags: %w", err)
	}
	return nil
}
