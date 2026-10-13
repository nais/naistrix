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
			choices, err := arg.ChoicesFunc(ctx, newArguments(definitions[:i], slices.Clone(values[:i])))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve choices for argument %q: %w", arg.Name, err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if len(choices) == 0 {
				return nil, Errorf("no choices available for argument %q", arg.Name)
			}
			arg.Choices = slices.Clone(choices)
			if err := arg.validateChoices(); err != nil {
				return nil, err
			}
			definitions[i] = arg
		}

		if i >= len(values) {
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
			value := arg.normalizeChoice(values[j])
			if len(arg.Choices) > 0 && !slices.Contains(arg.Choices, value) {
				return nil, Errorf("invalid value %q for argument %q: expected one of %s", values[j], arg.Name, strings.Join(arg.Choices, ", "))
			}
			values[j] = value
		}
	}
	return newArguments(definitions, values), nil
}
