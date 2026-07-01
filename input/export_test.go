package input

// SetInteractive overrides the interactivity detection and returns a function that restores the previous behaviour. It
// is only available to tests.
func SetInteractive(f func() bool) (restore func()) {
	prev := interactive
	interactive = f
	return func() { interactive = prev }
}
