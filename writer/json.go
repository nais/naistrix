package writer

import (
	"encoding/json"
	"io"
)

const indent = "  "

type JSON struct {
	prettify bool
	w        io.Writer
}

func NewJSON(w io.Writer, prettify bool) *JSON {
	return &JSON{
		prettify: prettify,
		w:        w,
	}
}

func (j *JSON) Write(v any) error {
	enc := json.NewEncoder(j.w)
	if j.prettify {
		enc.SetIndent("", indent)
	}
	return enc.Encode(v)
}
