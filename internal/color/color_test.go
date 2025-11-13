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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Colorize(tt.in); got != tt.out {
				t.Errorf("Colorize() = %v, want %v", got, tt.out)
			}
		})
	}
}
