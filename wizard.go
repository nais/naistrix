package naistrix

import (
	"fmt"
	"slices"

	"github.com/nais/naistrix/wizardoption"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func (r *commandRegistry) shouldRunWizard(cmd *cobra.Command, args []string) bool {
	definition := r.commands[cmd]
	if definition == nil || definition.WizardFunc == nil || len(args) != 0 || !argumentsInteractive(cmd.Context()) {
		return false
	}
	inputFlag := false
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if !r.flags[flag].ignored {
			inputFlag = true
		}
	})
	return !inputFlag
}

func (c *Command) collectWizardInput(cmd *cobra.Command, out *OutputWriter, registry *commandRegistry) ([]string, error) {
	ctx := cmd.Context()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	args, err := c.WizardFunc(ctx, out)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if validate := registry.preparations[cmd].validateFlags; validate != nil {
		if err := validate(ctx); err != nil {
			return nil, err
		}
	}
	if len(c.Args) == 0 && len(args) > 0 {
		return nil, ValidateExactArgs(0)(ctx, newArguments(c.Args, args))
	}

	// Preserve ancestor argument validation, which was deferred along with their
	// flag validators. No synchronization or prompting occurs after collection.
	var ancestors []*cobra.Command
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		ancestors = append(ancestors, parent)
	}
	slices.Reverse(ancestors)
	for _, parent := range ancestors {
		definition := registry.commands[parent]
		if definition == nil {
			continue
		}
		resolved, err := definition.resolveArguments(ctx, args, out, false)
		if err != nil {
			return nil, err
		}
		if err := definition.validateArgumentValues(cmd, resolved); err != nil {
			return nil, err
		}
	}
	return args, nil
}

func (c *Command) printWizardPreview(cmd *cobra.Command, args *Arguments, out *OutputWriter, registry *commandRegistry) {
	if slices.Contains(c.WizardOptions, wizardoption.WithoutCommandPreview()) {
		return
	}
	preview, redacted, err := registry.wizardPreview(cmd, args)
	if err != nil {
		_, _ = fmt.Fprintf(out.diagnosticWriter, "Command preview unavailable: %s.\n", err)
		return
	}
	if redacted {
		_, _ = fmt.Fprintf(out.diagnosticWriter, "To run this non-interactively next time, replace the redacted values:\n\n  %s\n\n", preview)
	} else {
		_, _ = fmt.Fprintf(out.diagnosticWriter, "To run this non-interactively next time:\n\n  %s\n\n", preview)
	}
}
