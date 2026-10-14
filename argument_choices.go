package naistrix

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// validateChoices rejects duplicate options, including case-folded duplicates when matching is case-insensitive.
func (a Argument) validateChoices() error {
	for i, choice := range a.Choices {
		for _, previous := range a.Choices[:i] {
			if previous == choice || a.ChoicesCaseInsensitive && strings.EqualFold(previous, choice) {
				return fmt.Errorf("argument %q contains duplicate choices: %q", a.Name, choice)
			}
		}
	}
	return nil
}

// resolveChoices snapshots dynamic choices without changing the command's argument definition.
func (a Argument) resolveChoices(ctx context.Context, prefix *Arguments) (Argument, error) {
	if a.ChoicesFunc == nil {
		return a, nil
	}
	if err := ctx.Err(); err != nil {
		return a, err
	}
	choices, err := a.ChoicesFunc(ctx, prefix)
	if err != nil {
		return a, fmt.Errorf("failed to resolve choices for argument %q: %w", a.Name, err)
	}
	if err := ctx.Err(); err != nil {
		return a, err
	}
	if len(choices) == 0 {
		return a, Errorf("no choices available for argument %q", a.Name)
	}
	a.Choices = slices.Clone(choices)
	if err := a.validateChoices(); err != nil {
		return a, err
	}
	return a, nil
}

// resolveChoice validates a supplied value and returns the canonical spelling of a matching choice.
func (a Argument) resolveChoice(value string) (string, error) {
	canonical := a.normalizeChoice(value)
	if len(a.Choices) > 0 && !slices.Contains(a.Choices, canonical) {
		return "", Errorf("invalid value %q for argument %q: expected one of %s", value, a.Name, strings.Join(a.Choices, ", "))
	}
	return canonical, nil
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
