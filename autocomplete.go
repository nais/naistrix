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
//
// Flags are synchronized and ancestor-to-child [ValidateFlagsFunc] callbacks run before this function.
type AutoCompleteFunc func(ctx context.Context, args *Arguments, toComplete string) (completions []string, activeHelp string)

// syncConfigAndFlagsFunc is responsible for loading application configuration and the synchronization of global,
// ancestor, and current-command flag structs for the target command. Cobra must have parsed the target command's flags
// before it is called. It must not run validation, prompt the user, or write ordinary command output.
type syncConfigAndFlagsFunc func(cmd *cobra.Command) error

// completionPreparers associates each execution target with its synchronization and ancestor-to-child validation
// chain. Flag callbacks may be inherited, so preparation must use the target, not the command defining the flag.
type completionPreparers map[*cobra.Command]completionPreparation

type completionPreparation struct {
	syncFlags     syncConfigAndFlagsFunc
	validateFlags ValidateFlagsFunc
}

// wrapArgumentCompletion validates execution prerequisites only after all target flags have been synchronized.
func (p completionPreparers) wrapArgumentCompletion(fn cobra.CompletionFunc) cobra.CompletionFunc {
	return p.wrapFlagCompletion(func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if validate := p[cmd].validateFlags; validate != nil {
			if err := validate(cmd.Context()); err != nil {
				return completionError(cmd, err)
			}
		}
		if fn == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fn(cmd, args, toComplete)
	})
}

// wrapFlagCompletion synchronizes flags without validating execution prerequisites. Required flags may still be
// missing, including the flag being completed. Each callback is responsible for its own completion prerequisites.
func (p completionPreparers) wrapFlagCompletion(fn cobra.CompletionFunc) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		syncFlags := p[cmd].syncFlags
		if syncFlags == nil {
			// Cobra's generated commands have no Naistrix validator, but still need synchronized global flags.
			syncFlags = p[cmd.Root()].syncFlags
		}
		if syncFlags != nil {
			if err := syncFlags(cmd); err != nil {
				return completionError(cmd, err)
			}
		}
		return fn(cmd, args, toComplete)
	}
}

// completionError displays diagnostics as active help and on stderr, without suggesting files.
func completionError(cmd *cobra.Command, err error) ([]string, cobra.ShellCompDirective) {
	message := fmt.Sprintf("Completion failed: %v", err)
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), message)
	// Completion is line-oriented; keep multiline errors from becoming ordinary suggestions.
	help := strings.Join(strings.Fields(message), " ")
	return cobra.AppendActiveHelp(nil, help), cobra.ShellCompDirectiveNoFileComp
}

func autocompleteFiles(ext []string) cobra.CompletionFunc {
	ext = slices.Clone(ext)
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

		return cobra.AppendActiveHelp(slices.Clone(ext), fmt.Sprintf("Select a file%s.", helpSuffix)), cobra.ShellCompDirectiveFilterFileExt
	}
}
