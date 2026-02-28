package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempJSONL(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, line := range lines {
		_, _ = f.WriteString(line + "\n")
	}
	return path
}

func TestReadSessionFile_ValidLines(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"type":"user","uuid":"u1","timestamp":"2025-01-01T00:00:00Z","message":{"role":"user","content":"hello"},"sessionId":"s1","userType":"external","cwd":"/tmp","version":"1.0"}`,
		`{"type":"assistant","uuid":"u2","timestamp":"2025-01-01T00:01:00Z","message":{"role":"assistant","content":[]},"sessionId":"s1","userType":"external","cwd":"/tmp","version":"1.0"}`,
	})

	lines, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].Type != "user" {
		t.Errorf("expected type 'user', got %q", lines[0].Type)
	}
	if lines[1].Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", lines[1].Type)
	}
}

func TestReadSessionFile_SkipsFilteredTypes(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"type":"user","uuid":"u1","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		`{"type":"progress","uuid":"u2","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		`{"type":"system","uuid":"u3","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		`{"type":"file-history-snapshot","uuid":"u4","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		`{"type":"assistant","uuid":"u5","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
	})

	lines, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (user + assistant), got %d", len(lines))
	}
	if lines[0].UUID != "u1" {
		t.Errorf("expected first line uuid 'u1', got %q", lines[0].UUID)
	}
	if lines[1].UUID != "u5" {
		t.Errorf("expected second line uuid 'u5', got %q", lines[1].UUID)
	}
}

func TestReadSessionFile_SkipsMetaLines(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"type":"user","uuid":"u1","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1","isMeta":true}`,
		`{"type":"user","uuid":"u2","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
	})

	lines, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].UUID != "u2" {
		t.Errorf("expected uuid 'u2', got %q", lines[0].UUID)
	}
}

func TestReadSessionFile_SkipsEmptyLines(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"type":"user","uuid":"u1","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		"",
		"   ",
		`{"type":"assistant","uuid":"u2","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
	})

	lines, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestReadSessionFile_SkipsMalformedJSON(t *testing.T) {
	path := writeTempJSONL(t, []string{
		`{"type":"user","uuid":"u1","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
		`{not valid json`,
		`{"type":"assistant","uuid":"u2","timestamp":"t","message":{},"sessionId":"s","userType":"e","cwd":"/","version":"1"}`,
	})

	lines, err := ReadSessionFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestReadSessionFile_FileNotFound(t *testing.T) {
	_, err := ReadSessionFile("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
