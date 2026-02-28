package internal

import "testing"

func TestTruncateLines_WithinLimit(t *testing.T) {
	text := "line1\nline2\nline3"
	result := TruncateLines(text, 5)
	if result.Text != text {
		t.Errorf("expected unchanged text, got %q", result.Text)
	}
	if result.Truncated {
		t.Error("expected Truncated=false")
	}
	if result.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", result.TotalLines)
	}
}

func TestTruncateLines_ExactlyAtLimit(t *testing.T) {
	text := "a\nb\nc"
	result := TruncateLines(text, 3)
	if result.Text != text {
		t.Errorf("expected unchanged text, got %q", result.Text)
	}
	if result.Truncated {
		t.Error("expected Truncated=false")
	}
	if result.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", result.TotalLines)
	}
}

func TestTruncateLines_ExceedingLimit(t *testing.T) {
	text := "a\nb\nc\nd\ne"
	result := TruncateLines(text, 2)
	expected := "a\nb\n... (truncated)"
	if result.Text != expected {
		t.Errorf("expected %q, got %q", expected, result.Text)
	}
	if !result.Truncated {
		t.Error("expected Truncated=true")
	}
	if result.TotalLines != 5 {
		t.Errorf("expected TotalLines=5, got %d", result.TotalLines)
	}
}

func TestTruncateLines_SingleLine(t *testing.T) {
	result := TruncateLines("hello", 1)
	if result.Text != "hello" {
		t.Errorf("expected 'hello', got %q", result.Text)
	}
	if result.Truncated {
		t.Error("expected Truncated=false")
	}
	if result.TotalLines != 1 {
		t.Errorf("expected TotalLines=1, got %d", result.TotalLines)
	}
}

func TestTruncateString_WithinLimit(t *testing.T) {
	if got := TruncateString("short", 10); got != "short" {
		t.Errorf("expected 'short', got %q", got)
	}
}

func TestTruncateString_ExactlyAtLimit(t *testing.T) {
	if got := TruncateString("12345", 5); got != "12345" {
		t.Errorf("expected '12345', got %q", got)
	}
}

func TestTruncateString_ExceedingLimit(t *testing.T) {
	if got := TruncateString("hello world", 5); got != "hello..." {
		t.Errorf("expected 'hello...', got %q", got)
	}
}
