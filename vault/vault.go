// Package vault exports Claude Code sessions as one markdown note per session
// into a notes folder (e.g. an Obsidian vault), keeping notes up to date as
// sessions continue.
package vault

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magarcia/ccsession-viewer/formatter"
	"github.com/magarcia/ccsession-viewer/parser"
)

// Status describes what Export did with a session.
type Status string

const (
	Written   Status = "written"
	Renamed   Status = "renamed"
	Unchanged Status = "unchanged"
	Empty     Status = "empty"
)

const maxTitleRunes = 80

// Index maps session IDs to the note filenames already present in the folder.
type Index map[string]string

// LoadIndex scans the markdown notes in dir for a session_id frontmatter field.
func LoadIndex(dir string) (Index, error) {
	idx := Index{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if id := readSessionID(filepath.Join(dir, e.Name())); id != "" {
			idx[id] = e.Name()
		}
	}
	return idx, nil
}

// readSessionID returns the session_id field of a note's frontmatter, if any.
func readSessionID(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 30 && scanner.Scan(); i++ {
		line := scanner.Text()
		if i == 0 && line != "---" {
			return ""
		}
		if i > 0 && line == "---" {
			return ""
		}
		if v, ok := strings.CutPrefix(line, "session_id: "); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Export writes the session at sessionPath as a note into dir and updates idx.
// A note is rewritten only when the session file is newer than the note,
// unless force is set. When the session title changes, the note is renamed.
func Export(sessionPath, dir string, idx Index, opts formatter.FormatOptions, force bool) (string, Status, error) {
	sessionInfo, err := os.Stat(sessionPath)
	if err != nil {
		return "", "", err
	}
	sessionID := strings.TrimSuffix(filepath.Base(sessionPath), ".jsonl")

	existing := idx[sessionID]
	if existing != "" && !force {
		if noteInfo, err := os.Stat(filepath.Join(dir, existing)); err == nil && !noteInfo.ModTime().Before(sessionInfo.ModTime()) {
			return existing, Unchanged, nil
		}
	}

	lines, err := parser.ReadSessionFile(sessionPath)
	if err != nil {
		return "", "", err
	}
	meta := parser.ExtractMetadata(lines)
	turns := parser.BuildTurns(lines)
	if !hasConversation(turns) {
		return "", Empty, nil
	}
	if meta.SessionID == "" {
		meta.SessionID = sessionID
	}
	if meta.Title == "" {
		meta.Title = firstUserMessage(turns)
	}

	opts.Frontmatter = true
	md := formatter.FormatSession(meta, turns, opts)

	name := noteName(meta, idx)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	if err := writeAtomic(filepath.Join(dir, name), []byte(md)); err != nil {
		return "", "", err
	}

	status := Written
	if existing != "" && existing != name {
		_ = os.Remove(filepath.Join(dir, existing))
		status = Renamed
	}
	idx[meta.SessionID] = name
	return name, status, nil
}

func hasConversation(turns []parser.ConversationTurn) bool {
	for _, t := range turns {
		if t.Type == "assistant" {
			return true
		}
	}
	return false
}

func firstUserMessage(turns []parser.ConversationTurn) string {
	for _, t := range turns {
		if t.Type != "user" {
			continue
		}
		text := strings.Join(strings.Fields(formatter.CleanUserText(strings.Join(t.Text, "\n"))), " ")
		if r := []rune(text); len(r) > maxTitleRunes {
			text = string(r[:maxTitleRunes])
		}
		if text != "" {
			return text
		}
	}
	return ""
}

// noteName builds "2006-01-02 1504 <project> - <title>.md", disambiguated with
// the short session ID if another session already uses the same name.
func noteName(meta parser.SessionMetadata, idx Index) string {
	date := "0000-00-00 0000"
	if t, err := time.Parse(time.RFC3339Nano, meta.Date); err == nil {
		date = t.Local().Format("2006-01-02 1504")
	}

	title := sanitize(meta.Title)
	if r := []rune(title); len(r) > maxTitleRunes {
		title = strings.TrimSpace(string(r[:maxTitleRunes])) + "…"
	}
	if title == "" {
		title = "Untitled"
	}

	base := date
	if meta.WorkingDirectory != "" {
		base += " " + sanitize(filepath.Base(meta.WorkingDirectory))
	}
	base += " - " + title

	name := base + ".md"
	for id, other := range idx {
		if other == name && id != meta.SessionID {
			short := meta.SessionID
			if len(short) > 8 {
				short = short[:8]
			}
			return fmt.Sprintf("%s (%s).md", base, short)
		}
	}
	return name
}

// Characters that are invalid in file names or break Obsidian links.
var reUnsafe = regexp.MustCompile(`[/\\:*?"<>|#^\[\]\x00-\x1f]+`)

func sanitize(s string) string {
	s = reUnsafe.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.Trim(s, " .")
}

// writeAtomic writes via a hidden temp file so a note app never sees a partial note.
func writeAtomic(path string, data []byte) error {
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
