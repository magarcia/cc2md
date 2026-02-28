package formatter

import (
	"regexp"
	"strings"
)

var (
	reSystemReminder   = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
	reCommandName      = regexp.MustCompile(`(?s)<command-name>.*?</command-name>\s*`)
	reCommandMessage   = regexp.MustCompile(`(?s)<command-message>.*?</command-message>\s*`)
	reCommandArgsOpen  = regexp.MustCompile(`<command-args>`)
	reCommandArgsClose = regexp.MustCompile(`</command-args>`)
	reLocalCmdCaveat   = regexp.MustCompile(`(?s)<local-command-caveat>.*?</local-command-caveat>\s*`)
	reLocalCmdStdout   = regexp.MustCompile(`(?s)<local-command-stdout>(.*?)</local-command-stdout>`)
	reLocalCmdStderr   = regexp.MustCompile(`(?s)<local-command-stderr>(.*?)</local-command-stderr>`)
	reDashLine         = regexp.MustCompile(`(?m)^-{3,}$`)
)

func FormatUserTurn(texts []string, timestamp string, flavor MarkdownFlavor) string {
	combined := strings.Join(texts, "\n")

	cleaned := reSystemReminder.ReplaceAllString(combined, "")
	cleaned = reCommandName.ReplaceAllString(cleaned, "")
	cleaned = reCommandMessage.ReplaceAllString(cleaned, "")
	cleaned = reCommandArgsOpen.ReplaceAllString(cleaned, "")
	cleaned = reCommandArgsClose.ReplaceAllString(cleaned, "")
	cleaned = reLocalCmdCaveat.ReplaceAllString(cleaned, "")
	cleaned = reLocalCmdStdout.ReplaceAllString(cleaned, "$1")
	cleaned = reLocalCmdStderr.ReplaceAllString(cleaned, "$1")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return ""
	}

	body := prefixLines(cleaned, "> ")
	tl := formatTimeLabel(timestamp)

	if flavor == FlavorCommonMark {
		return "> **Note:** **User**" + tl + "\n>\n" + body
	}
	return "> [!NOTE]\n> **User**" + tl + "\n>\n" + body
}

func FormatTeammateMessage(name, content, timestamp string, flavor MarkdownFlavor) string {
	trimmed := strings.TrimSpace(content)
	trimmed = reDashLine.ReplaceAllString(trimmed, "")
	body := prefixLines(trimmed, "> ")
	tl := formatTimeLabel(timestamp)

	if flavor == FlavorCommonMark {
		return "> **Tip:** **Agent: " + name + "**" + tl + "\n>\n" + body
	}
	return "> [!TIP]\n> **Agent: " + name + "**" + tl + "\n>\n" + body
}

func FormatLocalCommand(texts []string, timestamp string) string {
	tl := formatTimeLabel(timestamp)

	var commands, outputs []string
	for _, text := range texts {
		if strings.HasPrefix(text, "/") {
			commands = append(commands, text)
		} else {
			outputs = append(outputs, text)
		}
	}

	label := "local command"
	if len(commands) > 0 {
		label = strings.Join(commands, ", ")
	}

	output := strings.TrimSpace(strings.Join(outputs, "\n"))
	if output == "" {
		return "`" + label + "`" + tl
	}
	return "`" + label + "`" + tl + "\n```\n" + output + "\n```"
}

func formatTimeLabel(timestamp string) string {
	if timestamp == "" {
		return ""
	}
	return " `" + timestamp + "`"
}

func prefixLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
