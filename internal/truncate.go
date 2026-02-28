package internal

import "strings"

type TruncateResult struct {
	Text       string
	Truncated  bool
	TotalLines int
}

func TruncateLines(text string, maxLines int) TruncateResult {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return TruncateResult{Text: text, Truncated: false, TotalLines: len(lines)}
	}
	return TruncateResult{
		Text:       strings.Join(lines[:maxLines], "\n") + "\n... (truncated)",
		Truncated:  true,
		TotalLines: len(lines),
	}
}

func TruncateString(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
