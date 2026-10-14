package naistrix

import "io"

// SetCompletionErrorWriterForTest sets a custom error writer for tests.
func SetCompletionErrorWriterForTest(app *Application, writer io.Writer) {
	app.rootCommand.SetErr(writer)
}
