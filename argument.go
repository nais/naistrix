package naistrix

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nais/naistrix/input"
)

const (
	// defaultArgumentPrompt is the text question used when an argument has no custom Prompt or Choices.
	defaultArgumentPrompt = "Enter a value"

	// defaultArgumentSelectPrompt is the selection question used when an argument has Choices but no custom Prompt.
	defaultArgumentSelectPrompt = "Select a value"

	// repeatableArgumentPrompt is the text question used after collecting the first value of a repeatable argument.
	repeatableArgumentPrompt = "Enter another value, or press Enter to finish"

	// repeatableArgumentConfirm is the confirmation format used after each repeatable choice; %s is the argument name.
	repeatableArgumentConfirm = "Add another %s?"
)

var (
	promptArgument       = input.Input
	selectArgument       = input.Select[string]
	confirmArgument      = input.Confirm
	argumentsInteractive = input.IsInteractive
)

// Argument represents a positional argument for a command. All arguments for a command will be grouped together in a
// string slice, and the arguments will be injected into the command's [RunFunc] (amongst others) in the order they are
// defined.
type Argument struct {
	// Name is the name of the argument, used for help output. This field is required.
	Name string

	// Repeatable can be used for repeatable arguments. Only the last argument for a command can be repeatable.
	Repeatable bool

	// Prompt overrides the default question shown when this argument is missing in an interactive terminal. The prompt
	// will be prefixed argument position and uppercase name, as in "#1 NAME:". Automatic choice completion also uses
	// this text as shell active help, without that prefix. Mutually exclusive with [Argument.PromptFunc].
	Prompt string

	// PromptFunc renders the question for a missing argument and the active help for automatic choice completion. As
	// with [Argument.Prompt], interactive questions are prefixed with the argument position and uppercase name. Shell
	// active help uses the returned text without that prefix.
	//
	// Flags are synchronized before it runs, and args contains only earlier validated and normalized arguments. It is
	// called only when text is needed, at most once per argument per execution or completion request. An empty result
	// uses the default question when prompting and adds no active help during completion. Mutually exclusive with
	// [Argument.Prompt].
	PromptFunc ArgumentPromptFunc

	// Choices lists the valid values for this argument. If empty, any value is allowed. When prompting, these values
	// are presented as a selectable list instead of a text prompt. They also provide shell completion unless the
	// command configures [Command.AutoCompleteFunc] or [Command.AutoCompleteExtensions]. Mutually exclusive with
	// [Argument.ChoicesFunc].
	Choices []string

	// ChoicesFunc resolves allowed values at execution time, after flags are synchronized. Its args contain only
	// earlier positional arguments, already validated and normalized. The function is called at most once per argument
	// per execution, including for values supplied on the command line. An empty result means no valid choices. It also
	// supplies automatic shell completion and may be called again for each completion request. Mutually exclusive with
	// [Argument.Choices].
	ChoicesFunc ArgumentChoicesFunc

	// ChoicesCaseInsensitive accepts [Argument.Choices] regardless of case, using Unicode case folding. Accepted values
	// are normalized to the configured choice before [ValidateFunc] and [RunFunc] receive them. Choices must be
	// distinct under case-insensitive matching. This also applies to [Argument.ChoicesFunc] results and has no effect
	// without either source.
	ChoicesCaseInsensitive bool
}

// ArgumentChoicesFunc resolves allowed values for an argument using the command context and earlier resolved arguments.
// Returned values are used for both interactive selection and validation.
type ArgumentChoicesFunc func(ctx context.Context, args *Arguments) ([]string, error)

// ArgumentPromptFunc renders argument text using the command context and earlier resolved arguments.
type ArgumentPromptFunc func(ctx context.Context, args *Arguments) (string, error)

// Arguments represents the arguments sent to a command.
type Arguments struct {
	// args holds the command arguments provided by the user.
	args []*userArg

	// count includes the total amount of user supplied arguments.
	count int
}

type userArg struct {
	name       string
	repeatable bool
	value      any
}

// newArguments creates a new Arguments instance based on the command definition and the arguments provided by the user.
func newArguments(commandArgs []Argument, userArgs []string) *Arguments {
	a := make([]*userArg, 0)

	for i, commandArg := range commandArgs {
		if i >= len(userArgs) {
			break
		}

		var v any
		if commandArg.Repeatable {
			values := userArgs[i:]
			if commandArg.ChoicesCaseInsensitive && len(commandArg.Choices) > 0 {
				values = slices.Clone(values)
				for j, value := range values {
					values[j] = commandArg.normalizeChoice(value)
				}
			}
			v = values
		} else {
			v = commandArg.normalizeChoice(userArgs[i])
		}

		a = append(a, &userArg{
			name:       commandArg.Name,
			repeatable: commandArg.Repeatable,
			value:      v,
		})
	}

	return &Arguments{
		args:  a,
		count: len(userArgs),
	}
}

// normalizeChoice returns the configured spelling for a case-insensitive choice, preserving unmatched input for validation.
func (a Argument) normalizeChoice(value string) string {
	if !a.ChoicesCaseInsensitive {
		return value
	}

	for _, choice := range a.Choices {
		if strings.EqualFold(choice, value) {
			return choice
		}
	}
	return value
}

// promptText returns prompt text based on a callback or a statically configured string.
func (a Argument) promptText(ctx context.Context, prefix *Arguments) (string, error) {
	if a.PromptFunc != nil {
		return a.PromptFunc(ctx, prefix)
	}
	return a.Prompt, nil
}

// promptValues collects one or more values for a missing argument at the given 1-based position.
func (a Argument) promptValues(ctx context.Context, out *OutputWriter, position int) ([]string, error) {
	name := strings.ToUpper(a.Name)
	question := a.Prompt
	if question == "" {
		question = defaultArgumentPrompt
		if len(a.Choices) > 0 {
			question = defaultArgumentSelectPrompt
		}
	}
	prefix := fmt.Sprintf("#%d %s: ", position, name)
	prompt := prefix + question
	var values []string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var value string
		var err error
		if len(a.Choices) > 0 {
			value, err = selectArgument(prompt, a.Choices)
		} else {
			value, err = promptArgument(prompt)
		}
		if err != nil {
			return nil, err
		}
		if len(a.Choices) == 0 && strings.TrimSpace(value) == "" {
			if len(values) > 0 {
				return values, nil
			}
			out.Warnln("A value is required. Please try again.")
			continue
		}
		values = append(values, value)
		if !a.Repeatable {
			return values, nil
		}
		if len(a.Choices) > 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			more, err := confirmArgument(fmt.Sprintf(repeatableArgumentConfirm, name))
			if err != nil {
				return nil, err
			}
			if !more {
				return values, nil
			}
		} else {
			prompt = prefix + repeatableArgumentPrompt
		}
	}
}

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

// Len returns the number of supplied positional values, including all values for repeatable arguments.
func (a *Arguments) Len() int {
	return a.count
}

// All returns the command arguments as a slice of strings.
func (a *Arguments) All() []string {
	ret := make([]string, 0)
	for _, arg := range a.args {
		if arg.repeatable {
			return append(ret, arg.value.([]string)...)
		}

		ret = append(ret, arg.value.(string))
	}
	return ret
}

// Get retrieves a single argument by name. Using this for a repeatable argument or an argument that does not exist will
// cause a panic as a safeguard for the implementor.
func (a *Arguments) Get(name string) string {
	for _, arg := range a.args {
		if arg.name == name && !arg.repeatable {
			return arg.value.(string)
		}
	}
	panic(`"` + name + `" is not a valid argument`)
}

// GetRepeatable retrieves a single argument by name. Using this for a non-repeatable argument or an argument that does
// not exist will cause a panic as a safeguard for the implementor.
func (a *Arguments) GetRepeatable(name string) []string {
	for _, arg := range a.args {
		if arg.name == name && arg.repeatable {
			return arg.value.([]string)
		}
	}
	panic(`"` + name + `" is not a valid repeatable argument`)
}
