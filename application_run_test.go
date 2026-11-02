package naistrix

import (
	"errors"
	"io"
	"testing"
)

func TestApplicationRunGuardsBeforeOptions(t *testing.T) {
	app, _, err := NewApplication("singleusetest", "Single-use tests", "v0.0.0", ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	optionCalls := 0
	option := func(*runOptions) {
		optionCalls++
		if err := app.Run(); !errors.Is(err, ErrApplicationAlreadyRun) {
			t.Errorf("expected reentry from an option to be rejected, got %v", err)
		}
	}
	if err := app.Run(option, RunWithArgs([]string{"--help"})); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(option); !errors.Is(err, ErrApplicationAlreadyRun) {
		t.Errorf("expected subsequent run to be rejected, got %v", err)
	}
	if optionCalls != 1 {
		t.Errorf("option called %d times, want 1", optionCalls)
	}
}
