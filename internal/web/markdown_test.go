package web

import (
	"strings"
	"testing"
)

func TestRenderMarkdownBody(t *testing.T) {
	html, err := renderMarkdownBody("# Git status observation\n\n## Initial questions\n\n- branch name\n- clean tree\n\n**bold** and `code`\n", "Git status observation")
	if err != nil {
		t.Fatal(err)
	}
	body := string(html)
	for _, want := range []string{"<h2", "Initial questions", "<ul>", "<strong>bold</strong>", "<code>code</code>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered Markdown missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<h1") {
		t.Fatalf("matching document H1 should be omitted from detail body: %s", body)
	}
}

func TestRenderMarkdownBodyDoesNotRenderRawHTML(t *testing.T) {
	html, err := renderMarkdownBody("<script>alert('x')</script>\n", "Anything")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "<script") {
		t.Fatalf("raw HTML should not be rendered: %s", html)
	}
}
