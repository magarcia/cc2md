package formatter

import "testing"

func TestFormatUserTurn_PlainText(t *testing.T) {
	got := FormatUserTurn([]string{"Hello"}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> Hello"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_MultipleTexts(t *testing.T) {
	got := FormatUserTurn([]string{"Hello", "World"}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> Hello\n> World"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_Multiline(t *testing.T) {
	got := FormatUserTurn([]string{"line1\nline2"}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> line1\n> line2"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_StripSystemReminder(t *testing.T) {
	input := "<system-reminder>secret stuff</system-reminder>actual message"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> actual message"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_StripCommandName(t *testing.T) {
	input := "<command-name>foo</command-name>actual message"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> actual message"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_StripCommandMessage(t *testing.T) {
	input := "<command-message>foo</command-message>actual message"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> actual message"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_StripCommandArgsKeepInner(t *testing.T) {
	input := "<command-args>some args</command-args>"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> some args"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_StripLocalCommandCaveat(t *testing.T) {
	input := "<local-command-caveat>hidden</local-command-caveat>visible"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> visible"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_UnwrapLocalCommandStdout(t *testing.T) {
	input := "<local-command-stdout>output here</local-command-stdout>"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> output here"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_UnwrapLocalCommandStderr(t *testing.T) {
	input := "<local-command-stderr>error output</local-command-stderr>"
	got := FormatUserTurn([]string{input}, "", FlavorGFM)
	want := "> [!NOTE]\n> **User**\n>\n> error output"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_EmptyWhenOnlyTags(t *testing.T) {
	got := FormatUserTurn([]string{"<system-reminder>only tags</system-reminder>"}, "", FlavorGFM)
	if got != "" {
		t.Errorf("expected empty string, got: %s", got)
	}
}

func TestFormatUserTurn_EmptyForWhitespaceOnly(t *testing.T) {
	got := FormatUserTurn([]string{"   "}, "", FlavorGFM)
	if got != "" {
		t.Errorf("expected empty string, got: %s", got)
	}
}

func TestFormatUserTurn_WithTimestamp(t *testing.T) {
	got := FormatUserTurn([]string{"Hello"}, "10:30:00", FlavorGFM)
	want := "> [!NOTE]\n> **User** `10:30:00`\n>\n> Hello"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_CommonMark_PlainText(t *testing.T) {
	got := FormatUserTurn([]string{"Hello"}, "", FlavorCommonMark)
	want := "> **Note:** **User**\n>\n> Hello"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_CommonMark_WithTimestamp(t *testing.T) {
	got := FormatUserTurn([]string{"Hello"}, "10:30:00", FlavorCommonMark)
	want := "> **Note:** **User** `10:30:00`\n>\n> Hello"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatUserTurn_CommonMark_EmptyWhenOnlyTags(t *testing.T) {
	got := FormatUserTurn([]string{"<system-reminder>only tags</system-reminder>"}, "", FlavorCommonMark)
	if got != "" {
		t.Errorf("expected empty string, got: %s", got)
	}
}

func TestFormatTeammateMessage_Basic(t *testing.T) {
	got := FormatTeammateMessage("researcher", "Found the bug", "", FlavorGFM)
	want := "> [!TIP]\n> **Agent: researcher**\n>\n> Found the bug"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatTeammateMessage_Multiline(t *testing.T) {
	got := FormatTeammateMessage("tester", "line1\nline2", "", FlavorGFM)
	want := "> [!TIP]\n> **Agent: tester**\n>\n> line1\n> line2"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatTeammateMessage_TrimWhitespace(t *testing.T) {
	got := FormatTeammateMessage("builder", "  hello  ", "", FlavorGFM)
	want := "> [!TIP]\n> **Agent: builder**\n>\n> hello"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatTeammateMessage_CommonMark_Basic(t *testing.T) {
	got := FormatTeammateMessage("researcher", "Found the bug", "", FlavorCommonMark)
	want := "> **Tip:** **Agent: researcher**\n>\n> Found the bug"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatTeammateMessage_CommonMark_WithTimestamp(t *testing.T) {
	got := FormatTeammateMessage("tester", "line1\nline2", "10:00:00", FlavorCommonMark)
	want := "> **Tip:** **Agent: tester** `10:00:00`\n>\n> line1\n> line2"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatLocalCommand_Basic(t *testing.T) {
	got := FormatLocalCommand([]string{"/help"}, "")
	want := "`/help`"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatLocalCommand_WithOutput(t *testing.T) {
	got := FormatLocalCommand([]string{"/status", "all good"}, "")
	want := "`/status`\n```\nall good\n```"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatLocalCommand_WithoutCommandName(t *testing.T) {
	got := FormatLocalCommand([]string{"some output"}, "")
	want := "`local command`\n```\nsome output\n```"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatLocalCommand_WithTimestamp(t *testing.T) {
	got := FormatLocalCommand([]string{"/help"}, "10:30:00")
	want := "`/help` `10:30:00`"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
