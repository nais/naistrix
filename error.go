package naistrix

import (
	"fmt"

	"github.com/pterm/pterm"
)

// Error represents an error type that can be used to format error messages for end-users.
type Error struct {
	// Message is the error message that will be displayed to the end-user.
	Message string
}

// Error returns the error message formatted with pterm.Error. This method satisfies the error interface.
func (e Error) Error() string {
	return pterm.Error.Sprint(e.Message)
}

// Errorf formats an error message using the provided format and arguments. The returned type satisfies the error
// interface.
func Errorf(format string, a ...any) Error {
	return Error{Message: fmt.Sprintf(format, a...)}
}
