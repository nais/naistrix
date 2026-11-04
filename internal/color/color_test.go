package color

import "testing"

func TestColorize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "no tags",
			in:   "This is a test string.",
			out:  "This is a test string.",
		},
		{
			name: "info tag",
			in:   "This is an <info>informational</info> message.",
			out:  "This is an \x1b[96minformational\x1b[0m message.",
		},
		{
			name: "warn tag",
			in:   "This is a <warn>warning</warn> message.",
			out:  "This is a \x1b[33mwarning\x1b[0m message.",
		},
		{
			name: "error tag",
			in:   "This is an <error>error</error> message.",
			out:  "This is an \x1b[91merror\x1b[0m message.",
		},
		{
			name: "mixed tags",
			in:   "<info>Info</info>, <warn>Warn</warn>, and <error>Error</error> messages.",
			out:  "\x1b[96mInfo\x1b[0m, \x1b[33mWarn\x1b[0m, and \x1b[91mError\x1b[0m messages.",
		},
		{
			name: "standalone tags",
			in:   "<info>stray and </error>",
			out:  "<info>stray and </error>",
		},
		{
			name: "standalone opening tag",
			in:   "A stray <warn> tag.",
			out:  "A stray <warn> tag.",
		},
		{
			name: "standalone closing tag",
			in:   "A stray </info> tag.",
			out:  "A stray </info> tag.",
		},
		{
			name: "multiline",
			in:   "<info>First line.\nSecond line.</info>",
			out:  "\x1b[96mFirst line.\x1b[0m\n\x1b[96mSecond line.\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Colorize(tt.in); got != tt.out {
				t.Errorf("Colorize() = %v, want %v", got, tt.out)
			}
		})
	}
}

func TestStripTags(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text", in: "No formatting.", want: "No formatting."},
		{name: "info", in: "Use <info>--team</info>.", want: "Use --team."},
		{name: "warn", in: "<warn>Warning</warn>", want: "Warning"},
		{name: "error", in: "<error>Error</error>", want: "Error"},
		{name: "mixed", in: "<info>Info</info>, <warn>Warn</warn>, <error>Error</error>.", want: "Info, Warn, Error."},
		{name: "unknown", in: "<value> and <custom>text</custom>", want: "<value> and <custom>text</custom>"},
		{name: "mismatched", in: "<info>text</warn>", want: "<info>text</warn>"},
		{name: "standalone opening", in: "A stray <info> tag.", want: "A stray <info> tag."},
		{name: "standalone closing", in: "A stray </error> tag.", want: "A stray </error> tag."},
		{name: "multiline", in: "<info>First line.\nSecond line.</info>", want: "First line.\nSecond line."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripTags(tt.in); got != tt.want {
				t.Errorf("StripTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
