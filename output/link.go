package output

import "github.com/savioxavier/termlink"

var supportsHyperlinks = termlink.SupportsHyperlinks

// Link represents a terminal hyperlink.
//
// It renders as a clickable hyperlink when the terminal supports hyperlinks,
// and falls back to rendering only the name when hyperlinks are not supported.
type Link struct {
	Name string
	URL  string
}

// NewLink creates a new Link.
func NewLink(name, url string) Link {
	return Link{
		Name: name,
		URL:  url,
	}
}

// String returns a string representation of the link.
func (l Link) String() string {
	if l.URL == "" {
		return l.Name
	}

	if supportsHyperlinks() {
		return termlink.Link(l.Name, l.URL)
	}

	return l.Name
}
