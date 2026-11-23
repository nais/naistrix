package main_test

import (
	"os"

	"github.com/nais/naistrix"
)

func Example_yaml() {
	err := naistrix.NewCodedError("permission_denied", "You cannot update this resource.", nil)
	if err := naistrix.RenderError(os.Stdout, err, naistrix.ErrorFormatYAML); err != nil {
		panic(err)
	}
	// Output:
	// code: permission_denied
	// message: You cannot update this resource.
}
