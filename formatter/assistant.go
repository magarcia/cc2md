package formatter

import (
	"regexp"
	"strings"
)

var setextHrRegex = regexp.MustCompile(`(?m)^-{3,}$`)

func FormatAssistantText(texts []string) string {
	escaped := make([]string, len(texts))
	for i, t := range texts {
		escaped[i] = EscapeSetextHrs(t)
	}
	return strings.Join(escaped, "\n\n")
}

// EscapeSetextHrs replaces bare --- lines with <hr> to prevent setext H2 headings.
func EscapeSetextHrs(text string) string {
	return setextHrRegex.ReplaceAllString(text, "<hr>")
}

func FormatThinking(blocks []string, collapse bool, flavor MarkdownFlavor) string {
	if len(blocks) == 0 {
		return ""
	}

	escaped := make([]string, len(blocks))
	for i, b := range blocks {
		escaped[i] = EscapeSetextHrs(b)
	}
	combined := strings.Join(escaped, "\n\n<hr>\n\n")

	if flavor == FlavorCommonMark || !collapse {
		return "**Thinking:**\n\n" + combined
	}

	return strings.Join([]string{
		"<details>",
		"<summary>Thinking</summary>",
		"",
		combined,
		"",
		"</details>",
	}, "\n")
}
