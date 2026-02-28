package formatter

import (
	"strings"
	"testing"
)

func TestFormatThinking_EmptyBlocks(t *testing.T) {
	got := FormatThinking([]string{}, false, FlavorGFM)
	if got != "" {
		t.Errorf("expected empty string for empty blocks, got: %q", got)
	}
}

func TestFormatThinking_GFM_NotCollapsed(t *testing.T) {
	got := FormatThinking([]string{"I think..."}, false, FlavorGFM)
	want := "**Thinking:**\n\nI think..."
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatThinking_GFM_Collapsed(t *testing.T) {
	got := FormatThinking([]string{"I think..."}, true, FlavorGFM)
	if !strings.Contains(got, "<details>") {
		t.Errorf("expected <details>, got: %s", got)
	}
	if !strings.Contains(got, "<summary>Thinking</summary>") {
		t.Errorf("expected <summary>Thinking</summary>, got: %s", got)
	}
	if !strings.Contains(got, "I think...") {
		t.Errorf("expected content, got: %s", got)
	}
	if !strings.Contains(got, "</details>") {
		t.Errorf("expected </details>, got: %s", got)
	}
}

func TestFormatThinking_GFM_MultipleBlocks(t *testing.T) {
	got := FormatThinking([]string{"block one", "block two"}, false, FlavorGFM)
	if !strings.Contains(got, "block one") {
		t.Errorf("expected block one, got: %s", got)
	}
	if !strings.Contains(got, "block two") {
		t.Errorf("expected block two, got: %s", got)
	}
	if !strings.Contains(got, "<hr>") {
		t.Errorf("expected <hr> separator, got: %s", got)
	}
}

func TestFormatThinking_GFM_EscapesSetextHrs(t *testing.T) {
	got := FormatThinking([]string{"above\n---\nbelow"}, false, FlavorGFM)
	if strings.Contains(got, "\n---\n") {
		t.Errorf("expected --- to be escaped, got: %s", got)
	}
	if !strings.Contains(got, "<hr>") {
		t.Errorf("expected <hr> in place of ---, got: %s", got)
	}
}

func TestFormatThinking_CommonMark_CollapseIgnored(t *testing.T) {
	got := FormatThinking([]string{"I think..."}, true, FlavorCommonMark)
	if strings.Contains(got, "<details>") {
		t.Errorf("CommonMark should not produce <details>, got: %s", got)
	}
	want := "**Thinking:**\n\nI think..."
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatThinking_CommonMark_NotCollapsed(t *testing.T) {
	got := FormatThinking([]string{"thought"}, false, FlavorCommonMark)
	want := "**Thinking:**\n\nthought"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
