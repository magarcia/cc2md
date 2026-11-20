package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/magarcia/ccsession-viewer/formatter"
	"github.com/magarcia/ccsession-viewer/parser"
)

const testID = "11111111-2222-3333-4444-555555555555"

func copySession(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, testID+".jsonl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExport(t *testing.T) {
	t.Setenv("TZ", "UTC")
	session := copySession(t, t.TempDir())
	out := t.TempDir()
	idx := Index{}
	opts := formatter.FormatOptions{IncludeThinking: true, Collapse: true, MaxLines: 100, Flavor: formatter.FlavorGFM}

	name, status, err := Export(session, out, idx, opts, false)
	if err != nil {
		t.Fatal(err)
	}
	if status != Written {
		t.Fatalf("status = %s, want written", status)
	}
	if !strings.HasSuffix(name, " my-proj - Fix bug in x.go.md") || !strings.HasPrefix(name, "2026-10-09 ") {
		t.Errorf("unexpected name %q", name)
	}

	data, err := os.ReadFile(filepath.Join(out, name))
	if err != nil {
		t.Fatal(err)
	}
	md := string(data)
	for _, want := range []string{"---\ntitle: \"Fix bug: in x.go\"\n", "session_id: " + testID, "project: \"my-proj\"", "# Fix bug: in x.go", "Please fix the bug", "Let me look"} {
		if !strings.Contains(md, want) {
			t.Errorf("note missing %q", want)
		}
	}
	if strings.Contains(md, "ide_opened_file") {
		t.Error("IDE context was not stripped")
	}

	// The index is rebuilt from frontmatter, and an unchanged session is skipped.
	idx, err = LoadIndex(out)
	if err != nil {
		t.Fatal(err)
	}
	if idx[testID] != name {
		t.Fatalf("index = %v", idx)
	}
	if _, status, _ := Export(session, out, idx, opts, false); status != Unchanged {
		t.Errorf("second export status = %s, want unchanged", status)
	}

	// A renamed session replaces the old note.
	f, _ := os.OpenFile(session, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"type":"custom-title","customTitle":"Renamed","sessionId":"` + testID + `"}` + "\n")
	_ = f.Close()
	future := time.Now().Add(time.Minute)
	_ = os.Chtimes(session, future, future)

	newName, status, err := Export(session, out, idx, opts, false)
	if err != nil || status != Renamed {
		t.Fatalf("status = %s, err = %v, want renamed", status, err)
	}
	if !strings.HasSuffix(newName, " my-proj - Renamed.md") {
		t.Errorf("unexpected name %q", newName)
	}
	if _, err := os.Stat(filepath.Join(out, name)); !os.IsNotExist(err) {
		t.Error("old note was not removed")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 1 {
		t.Errorf("expected exactly one file in output dir, got %d", len(entries))
	}
}

func TestNoteNameCollision(t *testing.T) {
	idx := Index{"other-session": "2026-10-09 0957 p - Same.md"}
	meta := testMeta("Same")
	meta.Date = "2026-10-09T09:57:00Z"
	t.Setenv("TZ", "UTC")
	got := noteName(meta, idx)
	if got != "2026-10-09 0957 p - Same (11111111).md" {
		t.Errorf("got %q", got)
	}
}

func TestSanitize(t *testing.T) {
	got := sanitize(`a/b: c#d [e]|f?  `)
	if got != "a b c d e f" {
		t.Errorf("got %q", got)
	}
}
func testMeta(title string) parser.SessionMetadata {
	return parser.SessionMetadata{SessionID: testID, WorkingDirectory: "/x/p", Title: title}
}
