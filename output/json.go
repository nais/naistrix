package output

import (
	"encoding/json"
	"io"
)

const defaultJSONIdent = "  "

// JSONOptionFunc is a function that can be used to configure the JSON renderer.
type JSONOptionFunc func(*JSON)

// JSONWithPrettyOutput can be used to render "pretty" JSON instead of just a string.
func JSONWithPrettyOutput() JSONOptionFunc {
	return func(j *JSON) {
		j.prettify = true
	}
}

// JSONWithIndentChar can be used to set the indent character(s) used when rendering "pretty" JSON. The default is two
// spaces.
func JSONWithIndentChar(indent string) JSONOptionFunc {
	return func(j *JSON) {
		j.indentChar = indent
	}
}

type JSON struct {
	prettify   bool
	indentChar string
	writer     io.Writer
}

func NewJSON(w io.Writer, opts ...JSONOptionFunc) *JSON {
	j := &JSON{
		writer:     w,
		indentChar: defaultJSONIdent,
	}

	for _, opt := range opts {
		opt(j)
	}

	return j
}

func (j *JSON) Render(v any) error {
	enc := json.NewEncoder(j.writer)
	if j.prettify {
		enc.SetIndent("", j.indentChar)
	}
	return enc.Encode(v)
}
