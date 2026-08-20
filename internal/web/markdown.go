package web

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var detailMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

func renderMarkdownBody(body, title string) (template.HTML, error) {
	var output bytes.Buffer
	if err := detailMarkdown.Convert([]byte(withoutMatchingTitle(body, title)), &output); err != nil {
		return "", err
	}
	return template.HTML(output.String()), nil
}

func withoutMatchingTitle(body, title string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(line, "# ")) != strings.TrimSpace(title) || !strings.HasPrefix(strings.TrimSpace(line), "# ") {
			return strings.Join(lines, "\n")
		}
		lines = append(lines[:i], lines[i+1:]...)
		if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			lines = append(lines[:i], lines[i+1:]...)
		}
		return strings.Join(lines, "\n")
	}
	return body
}
