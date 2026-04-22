package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

type YAML struct {
	writer io.Writer
}

func NewYAML(w io.Writer) *YAML {
	return &YAML{
		writer: w,
	}
}

func (y *YAML) Render(v any) error {
	return yaml.NewEncoder(y.writer).Encode(v)
}
