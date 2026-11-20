package formatter

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/magarcia/ccsession-viewer/parser"
)

// FormatFrontmatter renders YAML frontmatter for note-taking apps such as Obsidian.
func FormatFrontmatter(meta parser.SessionMetadata) string {
	var b strings.Builder
	b.WriteString("---\n")
	field := func(key, value string) {
		if value != "" {
			fmt.Fprintf(&b, "%s: %s\n", key, value)
		}
	}
	field("title", strconv.Quote(meta.Title))
	field("date", localTime(meta.Date))
	field("updated", localTime(meta.LastActivity))
	if meta.WorkingDirectory != "" {
		field("project", strconv.Quote(filepath.Base(meta.WorkingDirectory)))
		field("cwd", strconv.Quote(meta.WorkingDirectory))
	}
	if meta.GitBranch != "" {
		field("git_branch", strconv.Quote(meta.GitBranch))
	}
	field("session_id", meta.SessionID)
	field("model", meta.Model)
	field("claude_code", meta.Version)
	b.WriteString("tags:\n  - claude-code\n")
	b.WriteString("---")
	return b.String()
}

func localTime(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return ""
	}
	return t.Local().Format("2006-01-02T15:04:05")
}
