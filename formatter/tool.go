package formatter

import (
	"fmt"
	"strings"

	"github.com/magarcia/ccsession-viewer/internal"
	"github.com/magarcia/ccsession-viewer/parser"
)

type ToolFormatOptions struct {
	Collapse bool
	MaxLines int
	Flavor   MarkdownFlavor
}

func FormatToolCalls(calls []parser.LinkedToolCall, opts ToolFormatOptions) string {
	parts := make([]string, len(calls))
	for i, c := range calls {
		parts[i] = formatSingleTool(c, opts)
	}
	body := strings.Join(parts, "\n\n")

	if opts.Collapse && opts.Flavor != FlavorCommonMark {
		label := fmt.Sprintf("Tool calls (%d)", len(calls))
		if len(calls) == 1 {
			label = "Tool call (1)"
		}
		return fmt.Sprintf("<details>\n<summary>%s</summary>\n\n%s\n\n</details>", label, body)
	}
	return body
}

var summaryKeys = map[string]string{
	"Bash":       "command",
	"Read":       "file_path",
	"Write":      "file_path",
	"Edit":       "file_path",
	"Glob":       "pattern",
	"Grep":       "pattern",
	"WebSearch":  "query",
	"WebFetch":   "url",
	"Task":       "description",
	"TaskCreate": "subject",
	"TaskUpdate": "taskId",
}

func formatSingleTool(call parser.LinkedToolCall, opts ToolFormatOptions) string {
	summary := inlineSummary(call)

	header := fmt.Sprintf("- **%s**", call.Name)
	if summary != "" {
		header += fmt.Sprintf(" `%s`", summary)
	}

	if call.Result == nil || *call.Result == "" {
		return header
	}

	result := *call.Result
	if opts.MaxLines > 0 {
		result = internal.TruncateLines(result, opts.MaxLines).Text
	}

	return fmt.Sprintf("%s\n\n  ```\n%s\n  ```", header, indentBlock(result, 2))
}

func inlineSummary(call parser.LinkedToolCall) string {
	if key, ok := summaryKeys[call.Name]; ok {
		if val, ok := call.Input[key]; ok {
			if s, ok := val.(string); ok {
				return internal.TruncateString(s, 80)
			}
		}
		return ""
	}

	if strings.HasPrefix(call.Name, "mcp__") {
		parts := strings.Split(call.Name, "__")
		return parts[len(parts)-1]
	}

	return ""
}

func indentBlock(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
