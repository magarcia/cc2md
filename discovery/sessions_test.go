package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeProjectName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"-Users-name-project", "/Users/name/project"},
		{"-Users-magarcia-dev-ccsession-viewer", "/Users/magarcia/dev/ccsession/viewer"},
		{"normal-name", "normal-name"},
		{"singleword", "singleword"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := decodeProjectName(tt.input)
			if got != tt.want {
				t.Errorf("decodeProjectName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSessionList_Empty(t *testing.T) {
	got := FormatSessionList(nil)
	if got != "No sessions found." {
		t.Errorf("FormatSessionList(nil) = %q, want %q", got, "No sessions found.")
	}
}

func TestFormatSessionList_Entries(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	sessions := []SessionEntry{
		{
			SessionID:  "abc12345-6789-0000-0000-000000000000",
			Project:    "/Users/name/project",
			ModifiedAt: now,
			Size:       2048,
			Name:       "Fix the login bug",
		},
		{
			SessionID:  "def45678-9012-0000-0000-000000000000",
			Project:    "/Users/name/other",
			ModifiedAt: now.Add(-time.Hour),
			Size:       512,
			Name:       "Refactor auth module",
		},
	}

	got := FormatSessionList(sessions)

	if !strings.HasPrefix(got, "Found 2 session(s):\n\n") {
		t.Errorf("unexpected header: %q", got)
	}
	if !strings.Contains(got, "2025-06-15 14:30") {
		t.Error("missing date in yyyy-mm-dd hh:mm format")
	}
	if !strings.Contains(got, "Fix the login bug") {
		t.Error("missing session name")
	}
	if !strings.Contains(got, "abc12345") {
		t.Error("missing short session id")
	}
	if !strings.Contains(got, "def45678") {
		t.Error("missing short session id for second entry")
	}
	if !strings.Contains(got, "KB") {
		t.Error("missing KB size")
	}
	if !strings.Contains(got, "/Users/name/project") {
		t.Error("missing project name")
	}
}

func TestListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sessionNamesDir := filepath.Join(tmpDir, "session-names")
	_ = os.MkdirAll(sessionNamesDir, 0o755)

	// Project 1: two sessions
	proj1 := filepath.Join(tmpDir, "projects", "-Users-name-projectA")
	_ = os.MkdirAll(proj1, 0o755)
	writeFile(t, filepath.Join(proj1, "session-1.jsonl"), `{"type":"user","message":{"role":"user","content":"hello world"}}`+"\n")
	writeFile(t, filepath.Join(proj1, "session-2.jsonl"), `{"type":"user","message":{"role":"user","content":"fix the tests"}}`+"\n")
	// Non-jsonl file should be ignored
	writeFile(t, filepath.Join(proj1, "notes.txt"), "ignore me")

	// Project 2: one session with a name file
	proj2 := filepath.Join(tmpDir, "projects", "-Users-name-projectB")
	_ = os.MkdirAll(proj2, 0o755)
	writeFile(t, filepath.Join(proj2, "session-3.jsonl"), `{"type":"user","message":{"role":"user","content":"unused"}}`+"\n")
	writeFile(t, filepath.Join(sessionNamesDir, "session-3.name"), "My Named Session")

	// Non-directory entry should be ignored
	writeFile(t, filepath.Join(tmpDir, "projects", "stray-file"), "ignore")

	projectsDir := filepath.Join(tmpDir, "projects")

	t.Run("no filter", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "")
		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}
		for _, e := range entries {
			if e.Project != "/Users/name/projectA" && e.Project != "/Users/name/projectB" {
				t.Errorf("unexpected project: %s", e.Project)
			}
			if e.Name == "" {
				t.Errorf("expected non-empty name for session %s", e.SessionID)
			}
		}
	})

	t.Run("session name from file", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "projectB")
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Name != "My Named Session" {
			t.Errorf("Name = %q, want %q", entries[0].Name, "My Named Session")
		}
	})

	t.Run("session name from first user message", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "projectA")
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
		names := map[string]bool{}
		for _, e := range entries {
			names[e.Name] = true
		}
		if !names["hello world"] {
			t.Error("expected a session named 'hello world'")
		}
		if !names["fix the tests"] {
			t.Error("expected a session named 'fix the tests'")
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "projectA")
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries for projectA filter, got %d", len(entries))
		}
		for _, e := range entries {
			if e.Project != "/Users/name/projectA" {
				t.Errorf("unexpected project: %s", e.Project)
			}
		}
	})

	t.Run("filter case insensitive", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "PROJECTB")
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry for PROJECTB filter, got %d", len(entries))
		}
	})

	t.Run("filter no match", func(t *testing.T) {
		entries := listSessionsIn(projectsDir, sessionNamesDir, "nonexistent")
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("nonexistent dir", func(t *testing.T) {
		entries := listSessionsIn("/nonexistent/path", sessionNamesDir, "")
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})
}

func TestListSessions_SessionIDFromFilename(t *testing.T) {
	tmpDir := t.TempDir()
	proj := filepath.Join(tmpDir, "myproject")
	_ = os.MkdirAll(proj, 0o755)
	writeFile(t, filepath.Join(proj, "abc-def-123.jsonl"), `{"type":"user","message":{"role":"user","content":"test"}}`+"\n")

	entries := listSessionsIn(tmpDir, "", "")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SessionID != "abc-def-123" {
		t.Errorf("SessionID = %q, want %q", entries[0].SessionID, "abc-def-123")
	}
}

func TestExtractFirstUserMessage(t *testing.T) {
	t.Run("simple string content", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		writeFile(t, f, `{"type":"user","message":{"role":"user","content":"hello world"}}`+"\n")
		got := extractFirstUserMessage(f, 60)
		if got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("skips array content", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		writeFile(t, f,
			`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"array"}]}}`+"\n"+
				`{"type":"user","message":{"role":"user","content":"fallback text"}}`+"\n",
		)
		got := extractFirstUserMessage(f, 60)
		if got != "fallback text" {
			t.Errorf("got %q, want %q", got, "fallback text")
		}
	})

	t.Run("strips XML tags", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		content := `<system-reminder>ignore this</system-reminder>actual question here`
		writeFile(t, f, `{"type":"user","message":{"role":"user","content":"`+content+`"}}`+"\n")
		got := extractFirstUserMessage(f, 60)
		if got != "actual question here" {
			t.Errorf("got %q, want %q", got, "actual question here")
		}
	})

	t.Run("truncates long content", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		long := strings.Repeat("a", 100)
		writeFile(t, f, `{"type":"user","message":{"role":"user","content":"`+long+`"}}`+"\n")
		got := extractFirstUserMessage(f, 20)
		if got != strings.Repeat("a", 20)+"..." {
			t.Errorf("got %q, want %q", got, strings.Repeat("a", 20)+"...")
		}
	})

	t.Run("skips non-user lines", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		writeFile(t, f,
			`{"type":"assistant","message":{"role":"assistant","content":[]}}`+"\n"+
				`{"type":"user","message":{"role":"user","content":"the question"}}`+"\n",
		)
		got := extractFirstUserMessage(f, 60)
		if got != "the question" {
			t.Errorf("got %q, want %q", got, "the question")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		writeFile(t, f, "")
		got := extractFirstUserMessage(f, 60)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("strips teammate-message tags", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		content := `<teammate-message teammate_id=\"lead\" summary=\"test\">ignore</teammate-message>real content`
		writeFile(t, f, `{"type":"user","message":{"role":"user","content":"`+content+`"}}`+"\n")
		got := extractFirstUserMessage(f, 60)
		if got != "real content" {
			t.Errorf("got %q, want %q", got, "real content")
		}
	})

	t.Run("keeps command-args inner text", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "test.jsonl")
		content := `<command-args>keep this</command-args>`
		writeFile(t, f, `{"type":"user","message":{"role":"user","content":"`+content+`"}}`+"\n")
		got := extractFirstUserMessage(f, 60)
		if got != "keep this" {
			t.Errorf("got %q, want %q", got, "keep this")
		}
	})
}

func TestStripXMLTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"system-reminder", "<system-reminder>hidden</system-reminder>visible", "visible"},
		{"command-name", "<command-name>cmd</command-name>text", "text"},
		{"command-message", "<command-message>msg</command-message>text", "text"},
		{"local-command-caveat", "<local-command-caveat>x</local-command-caveat>y", "y"},
		{"local-command-stdout", "<local-command-stdout>out</local-command-stdout>z", "z"},
		{"local-command-stderr", "<local-command-stderr>err</local-command-stderr>w", "w"},
		{"task-notification", "<task-notification>note</task-notification>rest", "rest"},
		{"teammate-message", `<teammate-message foo="bar">inner</teammate-message>outer`, "outer"},
		{"command-args keeps inner", "<command-args>inner</command-args>", "inner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripXMLTags(tt.input)
			got = strings.TrimSpace(got)
			if got != tt.want {
				t.Errorf("stripXMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
