package naistrix

import (
	"fmt"

	"github.com/pterm/pterm"
)

// Err represents an error type that can be used to format error messages for end-users.
type Err struct {
	Message string
}

// Error returns the error message formatted with pterm.Error. This method satisfies the error interface.
func (e Err) Error() string {
	return pterm.Error.Sprintf("%s", e.Message)
}

// Errorf formats an error message using the provided format and arguments. The returned type satisfies the error
// interface.
func Errorf(format string, a ...any) Err {
	return Err{Message: fmt.Sprintf(format, a...)}
}
