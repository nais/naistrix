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
// The args passed to this function is the arguments added to the command, in the same order. toComplete is the current
// input that the user is typing, and it can be used to filter the suggestions to be returned.
//
// The first return value is a slice of strings that will be used as suggestions, and the second return value is a
// string that will be used as active help text in the shell while performing auto-complete.
type AutoCompleteFunc func(ctx context.Context, args []string, toComplete string) (completions []string, activeHelp string)

// TODO: no need for this once the `SetDefaultShellCompDirective()` function is available in cobra (>v1.9.1)
func noAutocomplete() cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func autocomplete(autoCompleteFunc AutoCompleteFunc, autoCompleteFilesExtensions []string) cobra.CompletionFunc {
	if len(autoCompleteFilesExtensions) > 0 {
		return autocompleteFiles(autoCompleteFilesExtensions)
	}

	if autoCompleteFunc == nil {
		return noAutocomplete()
	}

	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completions, activeHelp := autoCompleteFunc(cmd.Context(), args, toComplete)
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
