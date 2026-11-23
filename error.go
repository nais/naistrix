package naistrix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

// Error represents an error type that can be used to format error messages for end-users.
type Error struct {
	// Message is the error message that will be displayed to the end-user.
	Message string
}

// Error returns the error message formatted with [pterm.Error]. This method satisfies the error interface.
func (e Error) Error() string {
	return pterm.Error.Sprint(e.Message)
}

// Errorf formats an error message using the provided format and arguments. The returned type satisfies the error
// interface.
func Errorf(format string, a ...any) Error {
	return Error{Message: fmt.Sprintf(format, a...)}
}

// CodedError associates a stable machine-readable code and public message with an optional cause. Callers define their
// own codes. The cause remains available to [errors.Is] and [errors.As], but is not included in the public message or
// structured output.
type CodedError struct {
	Code    string
	Message string
	err     error
}

// NewCodedError creates an error with a public code and message and an optional cause. Pass nil if there is no
// underlying error.
func NewCodedError(code, message string, err error) *CodedError {
	return &CodedError{Code: code, Message: message, err: err}
}

// Error returns the public message without exposing the underlying cause.
func (e *CodedError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *CodedError) Unwrap() error {
	return e.err
}

// ErrorFormat selects how [RenderError] presents an error.
type ErrorFormat string

const (
	// ErrorFormatText preserves the error's ordinary text presentation.
	ErrorFormatText ErrorFormat = "text"
	// ErrorFormatJSON writes one JSON object with code and message fields.
	ErrorFormatJSON ErrorFormat = "json"
	// ErrorFormatYAML writes one YAML document with code and message fields.
	ErrorFormatYAML ErrorFormat = "yaml"
)

// RenderError writes an error to w without choosing an exit status or terminating the process. A nil error produces no
// output. JSON and YAML use the first wrapped [CodedError]'s public message and code, falling back to "error" for an
// empty code. Other errors use code "error" and their ordinary error string. JSON and YAML messages omit terminal
// styling. A write failure is returned as-is in every format. Applications remain responsible for choosing an
// appropriate destination and format.
func RenderError(w io.Writer, err error, format ErrorFormat) error {
	if err == nil {
		return nil
	}

	switch format {
	case ErrorFormatText:
		_, writeErr := fmt.Fprintln(w, err)
		return writeErr
	case ErrorFormatJSON, ErrorFormatYAML:
		code, message := "error", err.Error()
		if coded, ok := errors.AsType[*CodedError](err); ok {
			if coded.Code != "" {
				code = coded.Code
			}
			message = coded.Message
		}
		details := struct {
			Code    string `json:"code" yaml:"code"`
			Message string `json:"message" yaml:"message"`
		}{
			Code:    code,
			Message: pterm.RemoveColorFromString(message),
		}
		var data []byte
		var encodeErr error
		if format == ErrorFormatJSON {
			data, encodeErr = json.Marshal(details)
		} else {
			data, encodeErr = yaml.Marshal(details)
		}
		if encodeErr != nil {
			return encodeErr
		}
		if format == ErrorFormatJSON {
			data = append(data, '\n')
		}
		_, writeErr := w.Write(data)
		return writeErr
	default:
		return fmt.Errorf("unsupported error format %q", format)
	}
}
