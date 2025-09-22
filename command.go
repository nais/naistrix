package naistrix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

// Argument represents a positional argument for a command. All arguments for a command will be grouped together in a
// string slice, and the arguments will be injected into the command's RunFunc (amongst others) in the order they are
// defined.
type Argument struct {
	// Name is the name of the argument, used for help output. This field is required.
	Name string

	// Repeatable can be used for repeatable arguments. Only the last argument for a command can be repeatable.
	Repeatable bool
}

// Command represents a command in the CLI application.
type Command struct {
	// Name is the name of the command, this is used to invoke the command in the CLI. This field is required.
	//
	// Example: "list" or "create-user".
	Name string

	// Aliases are alternative names for the command, used to invoke the command in the CLI.
	Aliases []string

	// Title is the title of the command, used as a short description for the help output and as a header for the
	// optional Description field. This field is required.
	Title string

	// Description is a detailed description of the command, shown in the help output. When set, it will be prefixed
	// with the Title field.
	Description string

	// RunFunc will be executed when the command is run. The RunFunc and SubCommands fields are mutually exclusive.
	RunFunc RunFunc

	// ValidateFunc will be executed before the command's RunFunc is executed.
	ValidateFunc ValidateFunc

	// AutoCompleteFunc sets up a function that will be used to provide auto-completion suggestions for the command.
	AutoCompleteFunc AutoCompleteFunc

	// AutoCompleteExtensions specifies which file extensions to list in autocompletion. This overrides
	// AutoCompleteFunc.
	AutoCompleteExtensions []string

	// Group places the command in a specific group. This is mainly used for grouping of commands in the help text.
	Group string

	// SubCommands adds subcommands to the command. The SubCommands and RunFunc fields are mutually exclusive.
	SubCommands []*Command

	// Args are the positional arguments to the command. The arguments will be injected into RunFunc. The command will
	// be validated when executed to ensure that the correct amount of arguments is specified.
	Args []Argument

	// Flags sets up flags for the command.
	Flags any

	// StickyFlags sets up flags that is persistent across all subcommands.
	StickyFlags any

	// Examples are examples of how to use the command. The examples are shown in the help output in the added order.
	Examples []Example

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
// The args passed to this function is the arguments passed to the command by the end-user.
type RunFunc func(ctx context.Context, out *OutputWriter, args []string) error

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
	cmd := c.Name
	for _, arg := range c.Args {
		format := " %[1]s" // ARG
		if arg.Repeatable {
			format += " [%[1]s...]" // ARG [ARG...]
		}
		cmd += fmt.Sprintf(format, strings.ToUpper(arg.Name))
	}

	return cmd
}

// validateArgs validates the positional arguments for the command, and prepends a ValidateFunc to the command that will
// make sure the correct amount of arguments is sent to the command when executed by the end-user.
func (c *Command) validateArgs() error {
	hasRepeatable := false

	for i, arg := range c.Args {
		if arg.Name == "" {
			return fmt.Errorf("argument name (%+v) cannot be empty", arg)
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
		c.ValidateFunc = func(ctx context.Context, args []string) error {
			if err := validationFunc(ctx, args); err != nil {
				return err
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
	if !strings.HasSuffix(title, ".") {
		title = title + "."
	}

	return title
}

// cobraLong generates the long description for the cobra.Command.
func (c *Command) cobraLong(short string) string {
	description := strings.TrimSpace(c.Description)
	if description == "" {
		return short
	}

	return strings.TrimRight(short, ".") + "\n\n" + description
}

// cobraRun wraps the RunFunc of the command into a function that can be used by the underlying cobra.Command.
func (c *Command) cobraRun(out *OutputWriter) func(*cobra.Command, []string) error {
	if c.RunFunc == nil {
		return func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				subCommands := "Available commands:\n"
				for _, s := range cmd.Commands() {
					subCommands = subCommands + "  " + s.Name() + "\n"
				}

				return fmt.Errorf(
					strings.TrimSpace(heredoc.Doc(`
						%w

						Usage:
						  %s <command> [flags]

						%s

						Use "%s -h" for more information.
					`)),
					err,
					cmd.CommandPath(),
					strings.TrimSpace(subCommands),
					cmd.CommandPath(),
				)
			}

			return cmd.Help()
		}
	}

	return func(cmd *cobra.Command, args []string) error {
		return c.RunFunc(cmd.Context(), out, args)
	}
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

	if (c.RunFunc == nil && len(c.SubCommands) == 0) || (c.RunFunc != nil && len(c.SubCommands) > 0) {
		return fmt.Errorf("either RunFunc or SubCommands must be set for command: %v", c.Name)
	}

	return c.validateArgs()
}

// init validates and initializes the cobra.Command.
func (c *Command) init(cmd string, out *OutputWriter, usageTemplate string) error {
	if err := c.validate(); err != nil {
		return err
	}

	cmd = cmd + " " + c.Name
	short := c.cobraShort()

	example, err := c.cobraExample(cmd)
	if err != nil {
		return fmt.Errorf("failed to generate examples for command %q: %w", c.Name, err)
	}

	c.cobraCmd = &cobra.Command{
		Example:           example,
		Aliases:           c.Aliases,
		Use:               c.cobraUse(),
		Short:             short,
		Long:              c.cobraLong(short),
		GroupID:           c.Group,
		RunE:              c.cobraRun(out),
		ValidArgsFunction: autocomplete(c.AutoCompleteFunc, c.AutoCompleteExtensions),
		PersistentPreRunE: func(co *cobra.Command, args []string) error {
			if c.ValidateFunc == nil {
				return nil
			}

			if err := c.ValidateFunc(co.Context(), args); err != nil {
				var e Error
				if errors.As(err, &e) {
					return e
				}
				return Errorf("input validation failed: %v", err)
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

	if err := setupFlags(c.cobraCmd, c.Flags, c.cobraCmd.Flags()); err != nil {
		return fmt.Errorf("failed to setup flags: %w", err)
	}

	if err := setupFlags(c.cobraCmd, c.StickyFlags, c.cobraCmd.PersistentFlags()); err != nil {
		return fmt.Errorf("failed to setup persistent flags: %w", err)
	}

	commandsAndAliases := make([]string, 0)
	for _, sub := range c.SubCommands {
		if err := sub.init(cmd, out, usageTemplate); err != nil {
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
