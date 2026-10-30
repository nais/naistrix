package naistrix

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// PreRunHook is a named application-wide hook for work such as authentication. Hooks run in registration order after
// configuration and all flags on the selected command path are synchronized, but before flag validation, argument
// choices, prompts, argument validation, and [Command.RunFunc]. An error stops execution and is returned unchanged.
//
// Hooks do not run for help, version output, shell completion, or commands that only display subcommands. A top-level
// alias runs the target command's hooks once. Deprecated commands run hooks before their replacement prompt; an
// accepted replacement runs its own hooks as a separate command execution.
type PreRunHook struct {
	// Name uniquely identifies the hook for [Command.SkipPreRunHooks]. It must not be blank.
	Name string

	// RunFunc performs setup before positional arguments are resolved. It must not be nil.
	RunFunc PreRunHookFunc
}

// PreRunHookFunc performs work before command validation and argument resolution.
// The context is the one supplied through [RunWithContext], and output should use out.
type PreRunHookFunc func(ctx context.Context, out *OutputWriter) error

// ApplicationWithPreRunHooks appends application-wide hooks in registration order.
// Commands can opt out of individual hooks using [Command.SkipPreRunHooks].
func ApplicationWithPreRunHooks(hooks ...PreRunHook) ApplicationOptionFunc {
	return func(a *Application) {
		a.preRunHooks = append(a.preRunHooks, hooks...)
	}
}

func validatePreRunHooks(hooks []PreRunHook) error {
	names := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		if strings.TrimSpace(hook.Name) == "" {
			return fmt.Errorf("pre-run hook name cannot be empty")
		}
		if hook.RunFunc == nil {
			return fmt.Errorf("pre-run hook %q must have a RunFunc", hook.Name)
		}
		names = append(names, hook.Name)
	}
	if name := duplicate(names); name != "" {
		return fmt.Errorf("duplicate pre-run hook name: %q", name)
	}
	return nil
}

func (a *Application) registerCommandDefinitions(cmd *Command) {
	a.commandDefinitions[cmd.cobraCmd] = cmd
	for _, sub := range cmd.SubCommands {
		a.registerCommandDefinitions(sub)
	}
}

// prepareCommand synchronizes the entire selected command path before invoking hooks, so even early hooks and parent
// validators can use the leaf command's flags. Cobra-generated commands and alias wrappers have no definition and skip
// hooks.
func (a *Application) prepareCommand(cmd *cobra.Command) error {
	var path []*Command
	for co := cmd; co != nil; co = co.Parent() {
		if definition, ok := a.commandDefinitions[co]; ok {
			path = append(path, definition)
		}
	}
	slices.Reverse(path)
	var skipped []string
	for _, definition := range path {
		if err := definition.syncFlags(cmd, a.config); err != nil {
			return err
		}
		skipped = append(skipped, definition.SkipPreRunHooks...)
	}
	if _, ok := a.commandDefinitions[cmd]; !ok || cmd.HasSubCommands() {
		return nil
	}
	for _, hook := range a.preRunHooks {
		if slices.Contains(skipped, hook.Name) {
			continue
		}
		if err := hook.RunFunc(cmd.Context(), a.output); err != nil {
			return err
		}
	}
	return nil
}
