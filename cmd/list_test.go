package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/magarcia/ccsession-viewer/discovery"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestPrintSessionsJSON_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		if err := printSessionsJSON(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var result []sessionJSON
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d elements", len(result))
	}
}

func TestPrintSessionsJSON_Schema(t *testing.T) {
	ts := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	sessions := []discovery.SessionEntry{{
		SessionID:  "abc123",
		Name:       "Test session",
		Project:    "myproject",
		Path:       "/path/to/abc123.jsonl",
		ModifiedAt: ts,
		Size:       4096,
	}}

	out := captureStdout(t, func() {
		if err := printSessionsJSON(sessions); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var result []sessionJSON
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}

	got := result[0]
	if got.SessionID != "abc123" {
		t.Errorf("session_id: got %q, want %q", got.SessionID, "abc123")
	}
	if got.Name != "Test session" {
		t.Errorf("name: got %q, want %q", got.Name, "Test session")
	}
	if got.Project != "myproject" {
		t.Errorf("project: got %q, want %q", got.Project, "myproject")
	}
	if got.Path != "/path/to/abc123.jsonl" {
		t.Errorf("path: got %q, want %q", got.Path, "/path/to/abc123.jsonl")
	}
	if got.SizeBytes != 4096 {
		t.Errorf("size_bytes: got %d, want %d", got.SizeBytes, 4096)
	}

	parsed, err := time.Parse(time.RFC3339, got.ModifiedAt)
	if err != nil {
		t.Errorf("modified_at %q is not valid RFC 3339: %v", got.ModifiedAt, err)
	}
	if !parsed.Equal(ts) {
		t.Errorf("modified_at: got %v, want %v", parsed, ts)
	}
}
