package input

import (
	"os"
	"testing"

	"atomicgo.dev/cursor"
	"github.com/pterm/pterm"
)

// SetInteractive overrides the interactivity detection and returns a function that restores the previous behavior.
func SetInteractive(f func() bool) (restore func()) {
	prev := interactive
	interactive = f
	return func() { interactive = prev }
}

// DiscardPromptOutputForTest suppresses the real prompt UI without replacing the prompt implementation. Tests must
// defer the returned restore function and must not run in parallel because output targets are shared.
func DiscardPromptOutputForTest(t *testing.T) (restore func()) {
	t.Helper()
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("failed to open discarded prompt output: %v", err)
	}
	previousStdout := os.Stdout
	previousPtermOutput := (pterm.PrefixPrinter{}).GetWriter()

	// pterm's text and selection areas write directly to os.Stdout, while styled output and cursor movement
	// use separate package-level writers. Redirect all three paths, not just pterm.SetDefaultOutput.
	os.Stdout = output
	pterm.SetDefaultOutput(output)
	cursor.SetTarget(output)
	return func() {
		os.Stdout = previousStdout
		pterm.SetDefaultOutput(previousPtermOutput)
		// The cursor library has no getter; these tests use its default stdout target outside prompt rendering.
		cursor.SetTarget(previousStdout)
		if err := output.Close(); err != nil {
			t.Errorf("failed to close discarded prompt output: %v", err)
		}
	}
}
