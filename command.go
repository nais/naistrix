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

	// SkipPreRunHooks names application hooks to skip for this command and all its descendants. Other hooks still run
	// in registration order. See [ApplicationWithPreRunHooks].
	SkipPreRunHooks []string

	// ValidateFlagsFunc runs after flags are synchronized and application pre-run hooks have run, before resolving
	// argument choices or prompts. It also runs before positional argument completion, but not flag-value completion.
	// Parent validators run for their descendants too.
	ValidateFlagsFunc ValidateFlagsFunc

	// ValidateArgsFunc runs after argument resolution and before [Command.RunFunc]. It is not called during completion.
	ValidateArgsFunc ValidateArgsFunc

	// AutoCompleteFunc sets up a function that will be used to provide auto-completion suggestions for the command.
	// When unset, arguments provide automatic choice or file-extension completion.
	AutoCompleteFunc AutoCompleteFunc

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
	Examples []CommandExample

	// cobraCmd is the internal cobra.Command for the Command.
	cobraCmd *cobra.Command
}

// CommandExample represents an example of how to use a command. It is used to provide examples in the help output for
// the command.
type CommandExample struct {
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

// validateArgumentDefinitions checks argument configuration and installs runtime count and choice checks ahead of the
// command's custom [ValidateArgsFunc].
func (c *Command) validateArgumentDefinitions() error {
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
		if len(arg.AutoCompleteExtensions) > 0 && (len(arg.Choices) > 0 || arg.ChoicesFunc != nil) {
			return fmt.Errorf("argument %q cannot have both AutoCompleteExtensions and Choices or ChoicesFunc", arg.Name)
		}
		if err := arg.validateChoiceDefinitions(); err != nil {
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
	var validationFunc ValidateArgsFunc
	if numArgs > 0 && hasRepeatable {
		validationFunc = ValidateMinArgs(numArgs)
	} else if numArgs > 0 {
		validationFunc = ValidateExactArgs(numArgs)
	}

	if validationFunc != nil {
		existingValidateArgsFunc := c.ValidateArgsFunc
		c.ValidateArgsFunc = func(ctx context.Context, args *Arguments) error {
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

			if existingValidateArgsFunc == nil {
				return nil
			}

			return existingValidateArgsFunc(ctx, args)
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

		if err := c.validateArgumentValues(cmd, resolved); err != nil {
			return err
		}

		// Silence the usage for errors that might occur in the RunFunc of the command
		cmd.SilenceUsage = true
		return c.RunFunc(cmd.Context(), resolved, out)
	}
}

// validateArgumentValues runs the configured count, choice, and custom checks on resolved positional argument values.
func (c *Command) validateArgumentValues(cmd *cobra.Command, args *Arguments) error {
	if c.ValidateArgsFunc == nil {
		return nil
	}
	if err := c.ValidateArgsFunc(cmd.Context(), args); err != nil {
		if e, ok := errors.AsType[Error](err); ok {
			return e
		}
		return Errorf("input validation failed: %v", err)
	}
	return nil
}

// validateFlagValues invokes the early flag-value validator without resolving positional arguments.
func (c *Command) validateFlagValues(ctx context.Context) error {
	if c.ValidateFlagsFunc == nil {
		return nil
	}
	return c.ValidateFlagsFunc(ctx)
}

// validateDefinition checks the command's configuration before initializing its Cobra command.
func (c *Command) validateDefinition() error {
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

	return c.validateArgumentDefinitions()
}

// init validates and initializes the cobra.Command.
func (c *Command) init(cmd string, out *OutputWriter, usageTemplate string, config *viper.Viper, syncInheritedConfigAndFlags syncConfigAndFlagsFunc, validateInheritedFlags ValidateFlagsFunc, preparers completionPreparers) error {
	if err := c.validateDefinition(); err != nil {
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
	// Completion skips command hooks, so collect the ancestor flag validators separately from flag synchronization.
	validateCompletionFlags := func(ctx context.Context) error {
		if validateInheritedFlags != nil {
			if err := validateInheritedFlags(ctx); err != nil {
				return err
			}
		}
		return c.validateFlagValues(ctx)
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
		ValidArgsFunction: preparers.wrapArgumentCompletion(c.autocomplete()),
		PersistentPreRunE: func(co *cobra.Command, args []string) error {
			if co.HasSubCommands() {
				// commands with subcommands does not have their own RunFunc, so we can skip validation
				return nil
			}

			if err := c.validateFlagValues(co.Context()); err != nil {
				return err
			}

			// Keep parent validation in the persistent hook so it also runs when a child command is executed.
			// Leaf commands validate in cobraRun after prompting for missing arguments and before RunFunc.
			if len(c.SubCommands) > 0 {
				resolved, err := c.resolveArguments(co.Context(), args, out, false)
				if err != nil {
					return err
				}
				return c.validateArgumentValues(co, resolved)
			}
			return nil
		},
	}
	preparers[c.cobraCmd] = completionPreparation{
		syncFlags:     syncConfigAndFlags,
		validateFlags: validateCompletionFlags,
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

	if err := setupFlags(c.cobraCmd, c.Args, c.Flags, c.cobraCmd.Flags(), preparers); err != nil {
		return fmt.Errorf("failed to setup flags: %w", err)
	}

	if err := setupFlags(c.cobraCmd, c.Args, c.StickyFlags, c.cobraCmd.PersistentFlags(), preparers); err != nil {
		return fmt.Errorf("failed to setup persistent flags: %w", err)
	}

	commandsAndAliases := make([]string, 0)
	for _, sub := range c.SubCommands {
		if err := sub.init(cmd, out, usageTemplate, config, syncConfigAndFlags, validateCompletionFlags, preparers); err != nil {
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

// resolveArguments uses execution-local definitions and values to avoid retaining dynamic choices between runs. Earlier
// arguments are resolved and validated before a later argument's ChoicesFunc is invoked.
func (c *Command) resolveArguments(ctx context.Context, supplied []string, out *OutputWriter, prompt bool) (*Arguments, error) {
	definitions := slices.Clone(c.Args)
	values := slices.Clone(supplied)
	if count := len(definitions); count > 0 && (len(values) >= count || !prompt) {
		validateCount := ValidateExactArgs(count)
		if definitions[count-1].Repeatable {
			validateCount = ValidateMinArgs(count)
		}
		if err := validateCount(ctx, newArguments(definitions, values)); err != nil {
			return nil, err
		}
	}

	for i := range definitions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		arg := definitions[i]
		if arg.ChoicesFunc != nil {
			// Only the resolved prefix is exposed; current and later values cannot influence this callback.
			resolved, err := arg.resolveChoices(ctx, newArguments(definitions[:i], slices.Clone(values[:i])))
			if err != nil {
				return nil, err
			}
			arg = resolved
			definitions[i] = resolved
		}

		if i >= len(values) {
			p, err := arg.promptText(ctx, newArguments(definitions[:i], slices.Clone(values[:i])))
			if err != nil {
				return nil, fmt.Errorf("failed to get prompt for argument %q: %w", arg.Name, err)
			}
			arg.Prompt = p
			answers, err := arg.promptValues(ctx, out, i+1)
			if err != nil {
				return nil, fmt.Errorf("failed to prompt for argument %q: %w", arg.Name, err)
			}
			values = append(values, answers...)
		}
		end := i + 1
		if arg.Repeatable {
			end = len(values)
		}
		for j := i; j < end; j++ {
			value, err := arg.resolveChoice(values[j])
			if err != nil {
				return nil, err
			}
			values[j] = value
		}
	}
	return newArguments(definitions, values), nil
}

func (c *Command) autocomplete() cobra.CompletionFunc {
	if c.AutoCompleteFunc == nil {
		for _, arg := range c.Args {
			if len(arg.Choices) > 0 || arg.ChoicesFunc != nil || len(arg.AutoCompleteExtensions) > 0 {
				return c.autocompleteArguments()
			}
		}
		return nil
	}

	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completions, activeHelp := c.AutoCompleteFunc(cmd.Context(), newArguments(c.Args, args), toComplete)
		if activeHelp != "" {
			completions = cobra.AppendActiveHelp(completions, activeHelp)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// autocompleteArguments completes the current argument without running ValidateArgsFunc or interactive prompts.
// The shared completion wrapper synchronizes and validates flags before this callback runs.
func (c *Command) autocompleteArguments() cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		index := len(args)
		if index >= len(c.Args) {
			if len(c.Args) == 0 || !c.Args[len(c.Args)-1].Repeatable {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			index = len(c.Args) - 1
		}
		arg := c.Args[index]
		if len(arg.Choices) == 0 && arg.ChoicesFunc == nil && len(arg.AutoCompleteExtensions) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		resolved, err := c.resolveCompletionArgument(cmd.Context(), args, index)
		if err != nil {
			return completionError(cmd, err)
		}
		if len(resolved.AutoCompleteExtensions) > 0 {
			return completeFiles(resolved.AutoCompleteExtensions, resolved.Prompt)
		}
		var completions []string
		for _, choice := range resolved.Choices {
			if completion, ok := resolved.completeChoice(choice, toComplete); ok {
				completions = append(completions, completion)
			}
		}
		if resolved.Prompt != "" {
			completions = cobra.AppendActiveHelp(completions, resolved.Prompt)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// resolveCompletionArgument resolves earlier arguments and the current argument's choices and help, not future arguments.
func (c *Command) resolveCompletionArgument(ctx context.Context, supplied []string, index int) (*Argument, error) {
	definitions := slices.Clone(c.Args[:index+1])
	values := slices.Clone(supplied[:index])
	for i := range definitions {
		arg := definitions[i]
		if arg.ChoicesFunc != nil {
			resolved, err := arg.resolveChoices(ctx, newArguments(definitions[:i], slices.Clone(values[:i])))
			if err != nil {
				return nil, err
			}
			arg = resolved
			definitions[i] = resolved
		}
		if i < index {
			value, err := arg.resolveChoice(values[i])
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
	}
	arg := definitions[index]
	// already supplied values of the current repeatable argument are validated but not exposed to ChoicesFunc.
	for _, value := range supplied[index:] {
		if _, err := arg.resolveChoice(value); err != nil {
			return nil, err
		}
	}
	p, err := arg.promptText(ctx, newArguments(definitions[:index], slices.Clone(values)))
	if err != nil {
		return nil, err
	}
	arg.Prompt = p
	return &arg, nil
}
