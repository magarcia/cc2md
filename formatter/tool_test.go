package formatter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/magarcia/ccsession-viewer/parser"
)

func strPtr(s string) *string { return &s }

func makeCall(name string, input map[string]interface{}, result *string) parser.LinkedToolCall {
	return parser.LinkedToolCall{
		ID:     "test-id",
		Name:   name,
		Input:  input,
		Result: result,
	}
}

func TestFormatToolCalls(t *testing.T) {
	opts := ToolFormatOptions{Collapse: false, MaxLines: 50}

	t.Run("formats a Bash tool call with command inline summary", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "ls -la"}, strPtr("file.txt"))},
			opts,
		)
		if !strings.Contains(result, "**Bash** `ls -la`") {
			t.Errorf("expected Bash summary, got: %s", result)
		}
		if !strings.Contains(result, "file.txt") {
			t.Errorf("expected file.txt in output, got: %s", result)
		}
	})

	t.Run("formats a Read tool call with file_path inline summary", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Read", map[string]interface{}{"file_path": "/src/index.ts"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**Read** `/src/index.ts`") {
			t.Errorf("expected Read summary, got: %s", result)
		}
	})

	t.Run("formats Edit, Write, Glob, Grep with their respective summaries", func(t *testing.T) {
		cases := []struct {
			name   string
			input  map[string]interface{}
			expect string
		}{
			{"Edit", map[string]interface{}{"file_path": "a.ts"}, "**Edit** `a.ts`"},
			{"Write", map[string]interface{}{"file_path": "b.ts"}, "**Write** `b.ts`"},
			{"Glob", map[string]interface{}{"pattern": "*.ts"}, "**Glob** `*.ts`"},
			{"Grep", map[string]interface{}{"pattern": "TODO"}, "**Grep** `TODO`"},
		}
		for _, tc := range cases {
			result := FormatToolCalls(
				[]parser.LinkedToolCall{makeCall(tc.name, tc.input, nil)},
				opts,
			)
			if !strings.Contains(result, tc.expect) {
				t.Errorf("%s: expected %s, got: %s", tc.name, tc.expect, result)
			}
		}
	})

	t.Run("formats WebSearch with query", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("WebSearch", map[string]interface{}{"query": "vitest docs"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**WebSearch** `vitest docs`") {
			t.Errorf("expected WebSearch summary, got: %s", result)
		}
	})

	t.Run("formats WebFetch with url", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("WebFetch", map[string]interface{}{"url": "https://example.com"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**WebFetch** `https://example.com`") {
			t.Errorf("expected WebFetch summary, got: %s", result)
		}
	})

	t.Run("formats Task with description", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Task", map[string]interface{}{"description": "research API"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**Task** `research API`") {
			t.Errorf("expected Task summary, got: %s", result)
		}
	})

	t.Run("formats TaskCreate with subject", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("TaskCreate", map[string]interface{}{"subject": "fix bug"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**TaskCreate** `fix bug`") {
			t.Errorf("expected TaskCreate summary, got: %s", result)
		}
	})

	t.Run("formats TaskUpdate with taskId", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("TaskUpdate", map[string]interface{}{"taskId": "42"}, nil)},
			opts,
		)
		if !strings.Contains(result, "**TaskUpdate** `42`") {
			t.Errorf("expected TaskUpdate summary, got: %s", result)
		}
	})

	t.Run("shows no inline summary for unknown tools", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("CustomTool", map[string]interface{}{"foo": "bar"}, nil)},
			opts,
		)
		if result != "- **CustomTool**" {
			t.Errorf("expected '- **CustomTool**', got: %s", result)
		}
	})

	t.Run("cleans up MCP tool names", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("mcp__slack__send_message", map[string]interface{}{}, nil)},
			opts,
		)
		if !strings.Contains(result, "`send_message`") {
			t.Errorf("expected send_message summary, got: %s", result)
		}
	})

	t.Run("shows header only when result is nil", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "echo hi"}, nil)},
			opts,
		)
		if result != "- **Bash** `echo hi`" {
			t.Errorf("expected header only, got: %s", result)
		}
	})

	t.Run("shows header only when result is empty string", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "echo"}, strPtr(""))},
			opts,
		)
		if result != "- **Bash** `echo`" {
			t.Errorf("expected header only, got: %s", result)
		}
	})

	t.Run("renders output in code block when not collapsed", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "ls"}, strPtr("a\nb"))},
			ToolFormatOptions{Collapse: false, MaxLines: 50},
		)
		if !strings.Contains(result, "```") {
			t.Errorf("expected code block, got: %s", result)
		}
		if strings.Contains(result, "<details>") {
			t.Errorf("should not contain details tag, got: %s", result)
		}
	})

	t.Run("collapse wraps single tool call with singular label", func(t *testing.T) {
		collapseOpts := ToolFormatOptions{Collapse: true, MaxLines: 50}
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "ls"}, strPtr("output"))},
			collapseOpts,
		)
		if !strings.Contains(result, "<details>") {
			t.Errorf("expected <details>, got: %s", result)
		}
		if !strings.Contains(result, "<summary>Tool call (1)</summary>") {
			t.Errorf("expected singular label, got: %s", result)
		}
		if !strings.Contains(result, "**Bash** `ls`") {
			t.Errorf("expected Bash summary, got: %s", result)
		}
		if !strings.Contains(result, "```") {
			t.Errorf("expected code block, got: %s", result)
		}
		if !strings.Contains(result, "</details>") {
			t.Errorf("expected </details>, got: %s", result)
		}
	})

	t.Run("collapse wraps multiple tool calls with plural label", func(t *testing.T) {
		collapseOpts := ToolFormatOptions{Collapse: true, MaxLines: 50}
		result := FormatToolCalls(
			[]parser.LinkedToolCall{
				makeCall("Bash", map[string]interface{}{"command": "ls"}, strPtr("files")),
				makeCall("Read", map[string]interface{}{"file_path": "a.ts"}, strPtr("content")),
				makeCall("Grep", map[string]interface{}{"pattern": "TODO"}, strPtr("match")),
			},
			collapseOpts,
		)
		if !strings.Contains(result, "<summary>Tool calls (3)</summary>") {
			t.Errorf("expected plural label, got: %s", result)
		}
		if !strings.Contains(result, "**Bash** `ls`") {
			t.Errorf("expected Bash, got: %s", result)
		}
		if !strings.Contains(result, "**Read** `a.ts`") {
			t.Errorf("expected Read, got: %s", result)
		}
		if !strings.Contains(result, "**Grep** `TODO`") {
			t.Errorf("expected Grep, got: %s", result)
		}
	})

	t.Run("does not wrap individual tools in their own details", func(t *testing.T) {
		collapseOpts := ToolFormatOptions{Collapse: true, MaxLines: 50}
		result := FormatToolCalls(
			[]parser.LinkedToolCall{
				makeCall("Bash", map[string]interface{}{"command": "ls"}, strPtr("files")),
				makeCall("Read", map[string]interface{}{"file_path": "a.ts"}, strPtr("content")),
			},
			collapseOpts,
		)
		count := strings.Count(result, "<details>")
		if count != 1 {
			t.Errorf("expected exactly 1 <details>, got %d in: %s", count, result)
		}
	})

	t.Run("truncates long output", func(t *testing.T) {
		lines := make([]string, 100)
		for i := range lines {
			lines[i] = fmt.Sprintf("line %d", i)
		}
		longOutput := strings.Join(lines, "\n")
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "big"}, strPtr(longOutput))},
			ToolFormatOptions{Collapse: false, MaxLines: 10},
		)
		if !strings.Contains(result, "... (truncated)") {
			t.Errorf("expected truncation marker, got: %s", result)
		}
	})

	t.Run("truncates long inline summaries", func(t *testing.T) {
		longCmd := strings.Repeat("x", 200)
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": longCmd}, nil)},
			opts,
		)
		if !strings.Contains(result, "...") {
			t.Errorf("expected truncation, got: %s", result)
		}
		if len(result) >= 200 {
			t.Errorf("expected result shorter than 200 chars, got %d", len(result))
		}
	})

	t.Run("formats multiple tool calls separated by double newline", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{
				makeCall("Read", map[string]interface{}{"file_path": "a.ts"}, nil),
				makeCall("Read", map[string]interface{}{"file_path": "b.ts"}, nil),
			},
			opts,
		)
		expected := "- **Read** `a.ts`\n\n- **Read** `b.ts`"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestFormatToolCalls_CommonMark(t *testing.T) {
	t.Run("collapse is ignored — no <details> produced", func(t *testing.T) {
		result := FormatToolCalls(
			[]parser.LinkedToolCall{makeCall("Bash", map[string]interface{}{"command": "ls"}, strPtr("output"))},
			ToolFormatOptions{Collapse: true, MaxLines: 50, Flavor: FlavorCommonMark},
		)
		if strings.Contains(result, "<details>") {
			t.Errorf("CommonMark should not produce <details>, got: %s", result)
		}
		if !strings.Contains(result, "**Bash** `ls`") {
			t.Errorf("expected Bash summary, got: %s", result)
		}
	})

	t.Run("collapse false produces same output as GFM collapse false", func(t *testing.T) {
		calls := []parser.LinkedToolCall{makeCall("Read", map[string]interface{}{"file_path": "a.ts"}, nil)}
		gfm := FormatToolCalls(calls, ToolFormatOptions{Collapse: false, MaxLines: 50, Flavor: FlavorGFM})
		cm := FormatToolCalls(calls, ToolFormatOptions{Collapse: false, MaxLines: 50, Flavor: FlavorCommonMark})
		if gfm != cm {
			t.Errorf("expected identical output without collapse:\ngfm: %s\ncm:  %s", gfm, cm)
		}
	})
}
