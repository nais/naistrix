package naistrix

import (
	"context"
	"fmt"
	"strings"
)

// ErrDeprecatedCommandWithoutReplacement is returned when a deprecated command does not have any replacement command.
var ErrDeprecatedCommandWithoutReplacement = &DeprecatedCommandError{}

// DeprecatedCommandError represents an error indicating that a command is deprecated, optionally suggesting a
// replacement and whether the user has chosen to execute the replacement or not.
type DeprecatedCommandError struct {
	// Replacement holds the suggested replacement command arguments / flags
	Replacement []string

	// ExecuteReplacement indicates whether the user has chosen to execute the replacement command.
	ExecuteReplacement bool
}

// Error returns the error message indicating that the command is deprecated, and suggests a replacement if available.
func (e *DeprecatedCommandError) Error() string {
	msg := "the command is deprecated"

	if len(e.Replacement) > 0 {
		msg += fmt.Sprintf(", please use %q instead", strings.Join(e.Replacement, " "))
	}

	return msg
}

// DeprecatedCommand represents a command that has been deprecated.
type DeprecatedCommand struct {
	replacementFunc DeprecatedCommandReplacementFunc
}

// DeprecatedCommandReplacementFunc is a function that generates the replacement command arguments for a deprecated
// command. When invoked it receives the current context and command arguments, and returns a slice of strings
// representing the replacement command and its arguments. Do not include the application name in the returned slice.
type DeprecatedCommandReplacementFunc func(context.Context, *Arguments) []string

// DeprecatedWithReplacement creates a DeprecatedCommand that specifies a replacement command using the provided
// slice. Do not include the application name in the slice, only the replacement command along with args and flags.
//
// Examples:
//
// 1. For a simple command replacement:
// []string{"new-command"}
//
// 2. For a command replacement with arguments and flags:
// []string{"new-command", "arg", "--flag", "value"}
func DeprecatedWithReplacement(args []string) *DeprecatedCommand {
	return &DeprecatedCommand{
		replacementFunc: func(context.Context, *Arguments) []string {
			return args
		},
	}
}

// DeprecatedWithReplacementFunc creates a DeprecatedCommand that specifies a replacement command using the provided
// DeprecationRunFunc. This allows for dynamic generation of the replacement command based on the current context,
// arguments and flags.
func DeprecatedWithReplacementFunc(fn DeprecatedCommandReplacementFunc) *DeprecatedCommand {
	return &DeprecatedCommand{
		replacementFunc: fn,
	}
}

// DeprecatedWithoutReplacement creates a DeprecatedCommand that does not have any replacement command.
func DeprecatedWithoutReplacement() *DeprecatedCommand {
	return &DeprecatedCommand{}
}
