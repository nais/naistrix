package docformat

import "testing"

func TestMarkdown(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "plain Markdown", in: "Existing **Markdown**, `code`, and [links](https://example.com).", want: "Existing **Markdown**, `code`, and [links](https://example.com)."},
		{name: "info", in: "Use <info>--team <team></info>.", want: "Use `--team <team>`."},
		{name: "warn", in: "<warn>Warning</warn>", want: "**Warning**"},
		{name: "error", in: "<error>Error</error>", want: "**Error**"},
		{name: "mixed", in: "<info>Info</info>, <warn>Warn</warn>, <error>Error</error>.", want: "`Info`, **Warn**, **Error**."},
		{name: "stray opening", in: "A stray <info> tag.", want: "A stray &lt;info&gt; tag."},
		{name: "stray closing", in: "A stray </info> tag.", want: "A stray &lt;/info&gt; tag."},
		{name: "stray bold tags", in: "<warn>text</error>", want: "&lt;warn&gt;text&lt;/error&gt;"},
		{name: "mismatched", in: "<info>text</warn>", want: "&lt;info&gt;text&lt;/warn&gt;"},
		{name: "stray after pair", in: "<info>code</info> and </error>", want: "`code` and &lt;/error&gt;"},
		{name: "multiline", in: "<warn>First line.\nSecond line.</warn>", want: "**First line.\nSecond line.**"},
		{name: "paragraphs", in: "<info>code</info>\n\n<error>Warning</error>", want: "`code`\n\n**Warning**"},
		{name: "backticks", in: "<info>`quoted`</info>", want: "`` `quoted` ``"},
		{name: "empty pairs", in: "<info></info><warn></warn><error></error>", want: ""},
		{name: "bare placeholders", in: "Set <team> and <env>.", want: "Set &lt;team&gt; and &lt;env&gt;."},
		{name: "placeholders in bold", in: "<warn>Choose <team></warn> or <error>check <env></error>.", want: "**Choose &lt;team&gt;** or **check &lt;env&gt;**."},
		{name: "placeholders after formatting", in: "Use <info>--team</info> with <team>.", want: "Use `--team` with &lt;team&gt;."},
		{name: "other tags", in: "<custom>text</custom> and &lt;info&gt;", want: "&lt;custom&gt;text&lt;/custom&gt; and &lt;info&gt;"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Markdown(tt.in); got != tt.want {
				t.Errorf("Markdown(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
