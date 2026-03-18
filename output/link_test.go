package output

import (
	"testing"

	"github.com/savioxavier/termlink"
)

func TestLink_String(t *testing.T) {
	t.Cleanup(func() {
		supportsHyperlinks = termlink.SupportsHyperlinks
	})

	t.Run("returns termlink when hyperlinks are supported", func(t *testing.T) {
		supportsHyperlinks = func() bool { return true }

		link := NewLink("Nais", "https://nais.io")

		want := termlink.Link("Nais", "https://nais.io")
		if got := link.String(); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("returns name when hyperlinks are not supported", func(t *testing.T) {
		supportsHyperlinks = func() bool { return false }

		link := NewLink("Nais", "https://nais.io")

		want := "Nais"
		if got := link.String(); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("returns name when url is empty", func(t *testing.T) {
		supportsHyperlinks = func() bool { return true }

		link := NewLink("Nais", "")

		want := "Nais"
		if got := link.String(); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}
