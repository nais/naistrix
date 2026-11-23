package naistrix_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestRenderErrorJSON(t *testing.T) {
	cause := errors.New("private upstream response")
	coded := naistrix.NewCodedError("permission_denied", "You cannot update this resource.", cause)
	wrapped := fmt.Errorf("private request context: %w", coded)
	if !errors.Is(wrapped, cause) {
		t.Fatal("wrapped error lost its cause")
	}
	var got *naistrix.CodedError
	if !errors.As(wrapped, &got) || got != coded {
		t.Fatal("wrapped error lost its coded error")
	}

	tests := []struct {
		name        string
		err         error
		wantCode    string
		wantMessage string
	}{
		{
			name:        "wrapped coded error exposes only public details",
			err:         wrapped,
			wantCode:    "permission_denied",
			wantMessage: "You cannot update this resource.",
		},
		{
			name:        "ordinary wrapped error",
			err:         fmt.Errorf("load resource: %w", errors.New("not found")),
			wantCode:    "error",
			wantMessage: "load resource: not found",
		},
		{
			name:        "missing code does not expose cause",
			err:         naistrix.NewCodedError("", "Update failed.", cause),
			wantCode:    "error",
			wantMessage: "Update failed.",
		},
		{
			name:        "terminal styling is removed",
			err:         errors.New("\x1b[31mfailed\x1b[0m"),
			wantCode:    "error",
			wantMessage: "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := naistrix.RenderError(&out, tt.err, naistrix.ErrorFormatJSON); err != nil {
				t.Fatal(err)
			}
			decoder := json.NewDecoder(&out)
			var result map[string]string
			if err := decoder.Decode(&result); err != nil {
				t.Fatal(err)
			}
			if len(result) != 2 || result["code"] != tt.wantCode || result["message"] != tt.wantMessage {
				t.Fatalf("unexpected error details: %#v", result)
			}
			var extra any
			if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
				t.Fatalf("expected exactly one JSON document, got %v", err)
			}
		})
	}
}

func TestCodedErrorMarshalOmitsCause(t *testing.T) {
	coded := naistrix.NewCodedError("denied", "Access denied.", &naistrix.Error{Message: "private details"})
	data, err := json.Marshal(coded)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields["Code"] != coded.Code || fields["Message"] != coded.Message {
		t.Fatalf("expected only the public fields, got %s", data)
	}
}

func TestRenderErrorLegacy(t *testing.T) {
	err := naistrix.Errorf("Cannot find team %q.", "example")
	for _, format := range []naistrix.ErrorFormat{naistrix.ErrorFormatText, naistrix.ErrorFormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			var out bytes.Buffer
			if renderErr := naistrix.RenderError(&out, err, format); renderErr != nil {
				t.Fatal(renderErr)
			}
			if format == naistrix.ErrorFormatText {
				if got, want := out.String(), err.Error()+"\n"; got != want {
					t.Fatalf("text = %q, want %q", got, want)
				}
				return
			}
			var result struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if decodeErr := json.Unmarshal(out.Bytes(), &result); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if result.Code != "error" || !strings.Contains(result.Message, err.Message) || strings.Contains(result.Message, "\x1b") {
				t.Fatalf("unexpected legacy error details: %#v", result)
			}
		})
	}
}

func TestRenderErrorFailures(t *testing.T) {
	writeErr := errors.New("destination closed")
	for _, format := range []naistrix.ErrorFormat{naistrix.ErrorFormatText, naistrix.ErrorFormatJSON} {
		t.Run(string(format), func(t *testing.T) {
			err := naistrix.RenderError(errorRenderWriter{err: writeErr}, errors.New("failed"), format)
			if !errors.Is(err, writeErr) {
				t.Fatalf("render error = %v, want %v", err, writeErr)
			}
		})
	}

	var out bytes.Buffer
	if err := naistrix.RenderError(&out, nil, naistrix.ErrorFormatJSON); err != nil || out.Len() != 0 {
		t.Fatalf("nil error produced output %q or error %v", out.String(), err)
	}
	if err := naistrix.RenderError(&out, errors.New("failed"), "unsupported"); err == nil || out.Len() != 0 {
		t.Fatalf("unsupported format produced output %q or no error", out.String())
	}
}

func TestApplicationUsageOnError(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		suppress    bool
		wantUsage   bool
		wantFailure bool
	}{
		{name: "default flag error", args: []string{"read", "--unknown"}, wantUsage: true, wantFailure: true},
		{name: "quiet flag error", args: []string{"read", "--unknown"}, suppress: true, wantFailure: true},
		{name: "quiet argument error", args: []string{"read"}, suppress: true, wantFailure: true},
		{name: "explicit help remains available", args: []string{"read", "--help"}, suppress: true, wantUsage: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			options := []naistrix.ApplicationOptionFunc{naistrix.ApplicationWithWriter(&out)}
			if tt.suppress {
				options = append(options, naistrix.ApplicationWithUsageOnError(false))
			}
			app, _, err := naistrix.NewApplication("app", "title", "v0.0.0", options...)
			if err != nil {
				t.Fatal(err)
			}
			if err := app.AddCommand(&naistrix.Command{
				Name: "read", Title: "Read a resource",
				Args: []naistrix.Argument{{Name: "resource"}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					t.Error("handler must not run after invalid input or help")
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			err = app.Run(naistrix.RunWithArgs(tt.args))
			if (err != nil) != tt.wantFailure {
				t.Fatalf("run error = %v, want failure %v", err, tt.wantFailure)
			}
			if got := strings.Contains(out.String(), "Usage:"); got != tt.wantUsage {
				t.Fatalf("usage present = %v, want %v; output: %q", got, tt.wantUsage, out.String())
			}
			if tt.suppress && tt.wantFailure && out.Len() != 0 {
				t.Fatalf("error unexpectedly wrote result output: %q", out.String())
			}
		})
	}
}

type errorRenderWriter struct {
	err error
}

func (w errorRenderWriter) Write([]byte) (int, error) {
	return 0, w.err
}
