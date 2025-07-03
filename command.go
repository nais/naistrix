package naistrix

import (
	"context"
	"fmt"
	"strings"

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

	// SubCommands adds sub-commands to the command. The SubCommands and RunFunc fields are mutually exclusive.
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
type RunFunc func(ctx context.Context, out Output, args []string) error

// cobraExample generates a formatted string of examples suitable for the underlying cobra.Command.
func (c *Command) cobraExample(prefix string) string {
	if len(c.Examples) == 0 {
		return ""
	}

	const indent = "  "

	var sb strings.Builder
	for _, ex := range c.Examples {
		description := strings.TrimSpace(ex.Description)
		if description == "" {
			panic(fmt.Sprintf("example for command %q is missing description", c.Name))
		}

		cmd := prefix + " " + strings.TrimSpace(ex.Command)
		sb.WriteString(indent + "# " + description + "\n")
		sb.WriteString(indent + "$ " + cmd + "\n\n")
	}

	return indent + strings.TrimSpace(sb.String())
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
func (c *Command) validateArgs() {
	hasRepeatable := false

	for i, arg := range c.Args {
		if arg.Name == "" {
			panic(fmt.Sprintf("argument name (%+v) cannot be empty", arg))
		}

		if arg.Repeatable {
			hasRepeatable = true
			if i != len(c.Args)-1 {
				panic(fmt.Sprintf("a repeatable argument (%+v) must be the last argument for the command", arg))
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
}

// cobraShort generates the short description for the cobra.Command.
func (c *Command) cobraShort() string {
	title := strings.TrimSpace(c.Title)

	if title == "" {
		panic(fmt.Sprintf("command %q is missing a title", c.Name))
	}

	if strings.Contains(title, "\n") {
		panic(fmt.Sprintf("the title for command %q contains newline", c.Name))
	}

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
func (c *Command) cobraRun(out Output) func(*cobra.Command, []string) error {
	if c.RunFunc == nil {
		return nil
	}

	return func(co *cobra.Command, args []string) error {
		return c.RunFunc(co.Context(), out, args)
	}
}

// init validates and initializes the cobra.Command.
func (c *Command) init(cmd string, out Output) {
	if strings.TrimSpace(c.Name) == "" {
		panic("command name cannot be empty")
	}

	if strings.Contains(c.Name, " ") {
		panic(fmt.Sprintf("command name cannot contain spaces: %v", c.Name))
	}

	c.validateArgs()

	cmd = cmd + " " + c.Name
	short := c.cobraShort()
	c.cobraCmd = &cobra.Command{
		Example:           c.cobraExample(cmd),
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
				return fmt.Errorf("validation failed: %w", err)
			}
			return nil
		},
	}

	setupFlags(c.cobraCmd, c.Flags, c.cobraCmd.Flags())
	setupFlags(c.cobraCmd, c.StickyFlags, c.cobraCmd.PersistentFlags())

	commandsAndAliases := make([]string, 0)
	for _, sub := range c.SubCommands {
		sub.init(cmd, out)
		c.cobraCmd.AddCommand(sub.cobraCmd)

		commandsAndAliases = append(commandsAndAliases, sub.Name)
		commandsAndAliases = append(commandsAndAliases, sub.Aliases...)
	}

	if d := duplicate(commandsAndAliases); d != "" {
		panic(fmt.Sprintf("command %q contains duplicate commands and/or aliases: %q", cmd, d))
	}
}
