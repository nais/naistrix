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
	return redirectPromptOutputForTest(t, output)
}

// CapturePromptOutputForTest captures all prompt output. Like DiscardPromptOutputForTest,
// it changes shared output targets and must not be used in parallel tests.
func CapturePromptOutputForTest(t *testing.T) (read func() string, restore func()) {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "prompt-output-*")
	if err != nil {
		t.Fatalf("failed to open captured prompt output: %v", err)
	}
	return func() string {
		t.Helper()
		data, err := os.ReadFile(output.Name())
		if err != nil {
			t.Fatalf("failed to read captured prompt output: %v", err)
		}
		return string(data)
	}, redirectPromptOutputForTest(t, output)
}

func redirectPromptOutputForTest(t *testing.T, output *os.File) (restore func()) {
	t.Helper()
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
