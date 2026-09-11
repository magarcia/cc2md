package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/magarcia/ccsession-viewer/parser"
)

type FormatOptions struct {
	CleanControlChars bool
	IncludeThinking   bool
	Collapse          bool
	MaxLines          int
	Flavor            MarkdownFlavor
}

// C0 control code entities, so we don't have to keep Fprintf'ing them.
var controlEntities = [0x20]string{
	"&#0;", "&#1;", "&#2;", "&#3;", "&#4;", "&#5;", "&#6;", "&#7;",
	"&#8;", "\t", "\n", "&#11;", "&#12;", "\r", "&#14;", "&#15;",
	"&#16;", "&#17;", "&#18;", "&#19;", "&#20;", "&#21;", "&#22;", "&#23;",
	"&#24;", "&#25;", "&#26;", "&#27;", "&#28;", "&#29;", "&#30;", "&#31;",
}

// If the session isn't interactive, then control characters can be problematic.
// Encode C0 control characters (U+0000–U+001F) other than TAB, LF and CR, plus
// plus DEL (U+007F), as decimal HTML character references (e.g. ESC -> "&#27;").
func cleanControlChars(s string, opts FormatOptions) string {

	// If we're in interactive mode, don't sanitize control characters; the terminal
	// should handle them properly.
	if !opts.CleanControlChars {
		return s
	}

	// Detect if the string contains any of the forbidden control characters
	// and record the location of the first one so we can jump right to it when
	// encoding below.
	firstLocation := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x7f || (c < 0x20 && c != '\t' && c != '\n' && c != '\r') {
			firstLocation = i
			break
		}
	}

	if firstLocation == -1 { // String's clean, so don't bother encoding.
		return s
	}

	// Otherwise, encode the control characters as decimal HTML character references.
	var b strings.Builder
	b.Grow(len(s))

	// The start of the string up to the first control character is not affected, so
	// just copy it over rather than reiterating each char.
	if firstLocation > 0 {
		b.WriteString(s[:firstLocation])
	}

	// Loop through the rest of the string, encoding C0 and DEL characters.
	// (Quicker to sanitize lots of shorter strings rather than the final buffer)
	for i := firstLocation; i < len(s); i++ {

		c := s[i]
		if c < 32 {
			// C0, incl. \n, \r, and \t, but those pass unchanged
			b.WriteString(controlEntities[c])
		} else if c == 127 {
			// 127 = DEL. Special case rather than bloating controlEntities
			b.WriteString("&#127;")
		} else {
			// Regular characters or UTF-8 multibyte, presumably.
			b.WriteByte(c)
		}
	}

	return b.String()
}

func FormatSession(meta parser.SessionMetadata, turns []parser.ConversationTurn, opts FormatOptions) string {

	m := FormatMetadata(meta, opts.Flavor)
	sections := []string{cleanControlChars(m, opts)}

	for _, turn := range turns {
		ts := formatTimestamp(turn.Timestamp)

		switch turn.Type {
		case "user":
			if s := FormatUserTurn(turn.Text, ts, opts.Flavor); s != "" {
				sections = append(sections, cleanControlChars(s, opts))
			}

		case "local-command":
			if s := FormatLocalCommand(turn.Text, ts); s != "" {
				sections = append(sections, cleanControlChars(s, opts))
			}

		case "teammate":
			name := turn.TeammateName
			if name == "" {
				name = "agent"
			}
			content := strings.Join(turn.Text, "\n")
			s := FormatTeammateMessage(name, content, ts, opts.Flavor)
			sections = append(sections, cleanControlChars(s, opts))

		case "assistant":
			var parts []string
			if opts.IncludeThinking && len(turn.Thinking) > 0 {
				parts = append(parts, FormatThinking(turn.Thinking, opts.Collapse, opts.Flavor))
			}
			if len(turn.Text) > 0 {
				parts = append(parts, FormatAssistantText(turn.Text))
			}
			if len(turn.ToolCalls) > 0 {
				parts = append(parts, FormatToolCalls(turn.ToolCalls, ToolFormatOptions{
					Collapse: opts.Collapse,
					MaxLines: opts.MaxLines,
					Flavor:   opts.Flavor,
				}))
			}
			if len(parts) > 0 {
				for i, p := range parts {
					parts[i] = cleanControlChars(p, opts)
				}
				body := strings.Join(parts, "\n\n")
				if ts != "" {
					body = fmt.Sprintf("> **Claude** `%s`\n\n%s", ts, body)
				}
				sections = append(sections, body)
			}
		}
	}

	return strings.Join(sections, "\n\n") + "\n"
}

func formatTimestamp(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return ""
	}
	return t.Local().Format("01/02 15:04:05")
}
