package discovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/magarcia/ccsession-viewer/internal"
)

// SessionEntry represents a single Claude Code session log file.
type SessionEntry struct {
	Path       string
	SessionID  string
	Project    string
	ModifiedAt time.Time
	Size       int64
	Name       string
}

// Compiled regexes for stripping XML tags from user messages.
var (
	// Tags where we strip the tag AND its content.
	reStripWithContent = regexp.MustCompile(
		`(?s)<(?:system-reminder|command-name|command-message|local-command-caveat|local-command-stdout|local-command-stderr|task-notification)[^>]*>.*?</(?:system-reminder|command-name|command-message|local-command-caveat|local-command-stdout|local-command-stderr|task-notification)>`,
	)
	reStripTeammate = regexp.MustCompile(`(?s)<teammate-message[^>]*>.*?</teammate-message>`)
	// Tags where we strip only the tags, keeping inner text.
	reStripTagOnly = regexp.MustCompile(`</?command-args[^>]*>`)
)

// ListSessions scans ~/.claude/projects/ for .jsonl session files.
// If projectFilter is non-empty, only projects whose decoded or raw name
// contains the filter (case-insensitive) are included.
func ListSessions(projectFilter string) []SessionEntry {
	return listSessionsIn(defaultProjectsDir(), defaultSessionNamesDir(), projectFilter)
}

func listSessionsIn(projectsDir, sessionNamesDir, projectFilter string) []SessionEntry {
	dirEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}

	filterLower := strings.ToLower(strings.TrimSpace(projectFilter))

	var entries []SessionEntry
	for _, d := range dirEntries {
		if !d.IsDir() {
			continue
		}
		name := d.Name()

		if filterLower != "" {
			decoded := decodeProjectName(name)
			if !strings.Contains(strings.ToLower(decoded), filterLower) &&
				!strings.Contains(strings.ToLower(name), filterLower) {
				continue
			}
		}

		projectPath := filepath.Join(projectsDir, name)
		files, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			filePath := filepath.Join(projectPath, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}
			sessionID := strings.TrimSuffix(f.Name(), ".jsonl")

			sessionName := readSessionName(sessionNamesDir, sessionID)
			if sessionName == "" {
				sessionName = extractFirstUserMessage(filePath, 60)
			}
			if sessionName == "" {
				sessionName = "(no title)"
			}

			entries = append(entries, SessionEntry{
				Path:       filePath,
				SessionID:  sessionID,
				Project:    decodeProjectName(name),
				ModifiedAt: info.ModTime(),
				Size:       info.Size(),
				Name:       sessionName,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ModifiedAt.After(entries[j].ModifiedAt)
	})

	return entries
}

// readSessionName reads a session name from ~/.claude/session-names/<id>.name.
func readSessionName(sessionNamesDir, sessionID string) string {
	if sessionNamesDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(sessionNamesDir, sessionID+".name"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// extractFirstUserMessage reads the first user message from a JSONL file
// and returns a cleaned, truncated version suitable as a session name.
func extractFirstUserMessage(filePath string, maxLen int) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	// Only scan the first 20 lines to keep it fast.
	for i := 0; i < 20 && scanner.Scan(); i++ {
		line := scanner.Bytes()

		// Quick check before full parse.
		if !strings.Contains(string(line), `"user"`) {
			continue
		}

		var jl struct {
			Type    string          `json:"type"`
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(line, &jl); err != nil {
			continue
		}
		if jl.Type != "user" {
			continue
		}

		var msg struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(jl.Message, &msg); err != nil {
			continue
		}
		if len(msg.Content) == 0 {
			continue
		}

		// Only handle string content, skip arrays.
		if msg.Content[0] != '"' {
			continue
		}
		var text string
		if err := json.Unmarshal(msg.Content, &text); err != nil {
			continue
		}

		text = stripXMLTags(text)
		text = strings.Join(strings.Fields(text), " ")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		return internal.TruncateString(text, maxLen)
	}
	return ""
}

// stripXMLTags removes known XML tags from user message content.
func stripXMLTags(s string) string {
	s = reStripWithContent.ReplaceAllString(s, "")
	s = reStripTeammate.ReplaceAllString(s, "")
	s = reStripTagOnly.ReplaceAllString(s, "")
	return s
}

// FormatSessionList returns a human-readable listing of sessions.
func FormatSessionList(sessions []SessionEntry) string {
	if len(sessions) == 0 {
		return "No sessions found."
	}

	// Compute max name width, capped at 40.
	maxNameLen := 0
	for _, s := range sessions {
		if l := len(s.Name); l > maxNameLen {
			maxNameLen = l
		}
	}
	if maxNameLen > 40 {
		maxNameLen = 40
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d session(s):\n\n", len(sessions))

	for i, s := range sessions {
		date := s.ModifiedAt.Format("2006-01-02 15:04")
		sizeKB := (s.Size + 512) / 1024
		name := s.Name
		if len(name) > 40 {
			name = internal.TruncateString(name, 37)
		}
		shortID := s.SessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		fmt.Fprintf(&b, "  %s  %-*s  %5dKB  %s  %s",
			date, maxNameLen, name, sizeKB, s.Project, shortID)
		if i < len(sessions)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func decodeProjectName(encoded string) string {
	if strings.HasPrefix(encoded, "-") {
		return strings.ReplaceAll(encoded, "-", "/")
	}
	return encoded
}

func defaultProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func defaultSessionNamesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "session-names")
}
