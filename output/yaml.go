package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAML is a renderer that encodes values as YAML and writes them to an [io.Writer]. Use [NewYAML] to construct one.
type YAML struct {
	writer io.Writer
}

// NewYAML creates a new [YAML] renderer that will write to the provided [io.Writer].
func NewYAML(w io.Writer) *YAML {
	return &YAML{
		writer: w,
	}
}

// Render encodes v as YAML and writes the result to the configured [io.Writer].
func (y *YAML) Render(v any) error {
	return yaml.NewEncoder(y.writer).Encode(v)
}
