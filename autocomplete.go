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

func (c *Command) autocomplete() cobra.CompletionFunc {
	if len(c.AutoCompleteExtensions) > 0 {
		return autocompleteFiles(c.AutoCompleteExtensions)
	}

	if c.AutoCompleteFunc == nil {
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
