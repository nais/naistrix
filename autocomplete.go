package naistrix

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// AutoCompleteFunc is a function that will be executed to provide auto-completion suggestions for a command.
//
// The args passed to this function is the arguments passed to the command by the user. toComplete is the current input
// that the user is typing, and it can be used to filter the suggestions to be returned.
//
// The first return value is a slice of strings that will be used as suggestions, and the second return value is a
// string that will be used as active help text in the shell while performing auto-complete. Return an empty slice and
// an empty string if you don't want to generate any completions.
type AutoCompleteFunc func(ctx context.Context, args *Arguments, toComplete string) (completions []string, activeHelp string)

// syncConfigAndFlagsFunc is responsible for loading application configuration and the synchronization of global,
// ancestor, and current-command flag structs for the target command. Cobra must have parsed the target command's flags
// before it is called. It must not run validation, prompt the user, or write ordinary command output.
type syncConfigAndFlagsFunc func(cmd *cobra.Command) error

func (c *Command) autocomplete(syncConfigAndFlags syncConfigAndFlagsFunc) cobra.CompletionFunc {
	if len(c.AutoCompleteExtensions) > 0 {
		return autocompleteFiles(c.AutoCompleteExtensions)
	}

	if c.AutoCompleteFunc == nil {
		for _, arg := range c.Args {
			if len(arg.Choices) > 0 || arg.ChoicesFunc != nil {
				return c.autocompleteChoices(syncConfigAndFlags)
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

// autocompleteChoices completes the current argument without running command validation or interactive prompts.
func (c *Command) autocompleteChoices(syncConfigAndFlags syncConfigAndFlagsFunc) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		index := len(args)
		if index >= len(c.Args) {
			if len(c.Args) == 0 || !c.Args[len(c.Args)-1].Repeatable {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			index = len(c.Args) - 1
		}
		arg := c.Args[index]
		if len(arg.Choices) == 0 && arg.ChoicesFunc == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := syncConfigAndFlags(cmd); err != nil {
			return completionError(cmd, err)
		}
		resolved, err := c.resolveCompletionChoices(cmd.Context(), args, index)
		if err != nil {
			return completionError(cmd, err)
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

// resolveCompletionChoices resolves only earlier arguments and the current choice source, not future arguments.
func (c *Command) resolveCompletionChoices(ctx context.Context, supplied []string, index int) (*Argument, error) {
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

// completeChoice preserves the typed prefix for case-insensitive matches because shells may filter results again.
// Command execution subsequently normalizes the completed value to the configured spelling.
func (a Argument) completeChoice(choice, prefix string) (string, bool) {
	if !a.ChoicesCaseInsensitive {
		return choice, strings.HasPrefix(choice, prefix)
	}
	choiceRunes := []rune(choice)
	prefixLength := len([]rune(prefix))
	if prefixLength > len(choiceRunes) {
		return "", false
	}

	choicePrefix := string(choiceRunes[:prefixLength])
	if !strings.EqualFold(choicePrefix, prefix) {
		return "", false
	}

	suffix := string(choiceRunes[prefixLength:])
	return prefix + suffix, true
}

// completionError displays diagnostics as active help and on stderr, without suggesting files.
func completionError(cmd *cobra.Command, err error) ([]string, cobra.ShellCompDirective) {
	message := fmt.Sprintf("Argument completion failed: %v", err)
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), message)
	// Completion is line-oriented; keep multiline errors from becoming ordinary suggestions.
	help := strings.Join(strings.Fields(message), " ")
	// ShellCompDirectiveError makes shells discard active help along with suggestions.
	return cobra.AppendActiveHelp(nil, help), cobra.ShellCompDirectiveNoFileComp
}

func autocompleteFiles(ext []string) cobra.CompletionFunc {
	slices.Sort(ext)
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		helpSuffix := ""
		if num := len(ext); num > 0 {
			formatted := make([]string, num)
			for i, e := range ext {
				formatted[i] = "*." + e
			}
			helpSuffix = " (" + strings.Join(formatted[:num-1], ", ") + " or " + formatted[num-1] + ")"
		}

		ext = cobra.AppendActiveHelp(ext, fmt.Sprintf("Select a file%s.", helpSuffix))
		return ext, cobra.ShellCompDirectiveFilterFileExt
	}
}
