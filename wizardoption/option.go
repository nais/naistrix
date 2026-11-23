// Package wizardoption provides options for command wizards.
package wizardoption

// Option configures a wizard. Its zero value leaves the default behavior unchanged.
type Option struct {
	withoutCommandPreview bool
}

// WithoutCommandPreview disables the equivalent CLI command preview without
// changing input collection, validation, or execution.
func WithoutCommandPreview() Option {
	return Option{withoutCommandPreview: true}
}
