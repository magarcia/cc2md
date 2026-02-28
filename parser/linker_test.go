package parser

import (
	"encoding/json"
	"testing"
)

func baseLine(overrides ...func(*JsonlLine)) JsonlLine {
	msg, _ := json.Marshal(UserMessage{Role: "user", Content: mustMarshal("")})
	line := JsonlLine{
		IsSidechain: false,
		UserType:    "external",
		Cwd:         "/project",
		SessionID:   "sess-1",
		Version:     "1.0.0",
		Type:        "user",
		UUID:        "uuid-1",
		Timestamp:   "2025-01-01T00:00:00Z",
		Message:     msg,
	}
	for _, o := range overrides {
		o(&line)
	}
	return line
}

// toolResultJSON builds a tool_result block as raw JSON so the "content" field
// is properly included (ContentBlock.ResultContent has json:"-").
func toolResultJSON(toolUseID string, content interface{}, isError ...bool) json.RawMessage {
	m := map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
	}
	if content != nil {
		m["content"] = content
	}
	if len(isError) > 0 && isError[0] {
		m["is_error"] = true
	}
	return mustMarshal(m)
}

// textBlockJSON builds a text block as raw JSON.
func textBlockJSON(text string) json.RawMessage {
	return mustMarshal(map[string]string{"type": "text", "text": text})
}

// userLineWithRawBlocks creates a user line from raw JSON block fragments.
func userLineWithRawBlocks(blocks ...json.RawMessage) JsonlLine {
	// Build the array manually
	arr := []byte{'['}
	for i, b := range blocks {
		if i > 0 {
			arr = append(arr, ',')
		}
		arr = append(arr, b...)
	}
	arr = append(arr, ']')

	msg := mustMarshal(UserMessage{Role: "user", Content: json.RawMessage(arr)})
	return baseLine(func(l *JsonlLine) {
		l.Type = "user"
		l.Message = msg
	})
}

func userLine(content interface{}) JsonlLine {
	var rawContent json.RawMessage
	switch v := content.(type) {
	case string:
		rawContent = mustMarshal(v)
	default:
		rawContent = mustMarshal(v)
	}
	msg := mustMarshal(UserMessage{Role: "user", Content: rawContent})
	return baseLine(func(l *JsonlLine) {
		l.Type = "user"
		l.Message = msg
	})
}

func assistantLine(content []ContentBlock, model ...string) JsonlLine {
	m := ""
	if len(model) > 0 {
		m = model[0]
	}
	msg := mustMarshal(AssistantMessage{Role: "assistant", Content: content, Model: m})
	return baseLine(func(l *JsonlLine) {
		l.Type = "assistant"
		l.Message = msg
	})
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// --- ExtractMetadata tests ---

func TestExtractMetadata_BasicFields(t *testing.T) {
	lines := []JsonlLine{userLine("hello")}
	meta := ExtractMetadata(lines)
	assertEqual(t, "sess-1", meta.SessionID, "sessionId")
	assertEqual(t, "/project", meta.WorkingDirectory, "cwd")
	assertEqual(t, "1.0.0", meta.Version, "version")
	assertEqual(t, "2025-01-01T00:00:00Z", meta.Date, "date")
}

func TestExtractMetadata_ModelFromAssistant(t *testing.T) {
	lines := []JsonlLine{
		userLine("hi"),
		assistantLine([]ContentBlock{{Type: "text", Text: "hello"}}, "claude-3-opus"),
	}
	meta := ExtractMetadata(lines)
	assertEqual(t, "claude-3-opus", meta.Model, "model")
}

func TestExtractMetadata_DefaultModel(t *testing.T) {
	meta := ExtractMetadata([]JsonlLine{userLine("hi")})
	assertEqual(t, "unknown", meta.Model, "model")
}

func TestExtractMetadata_FirstNonEmptyValues(t *testing.T) {
	line1 := baseLine(func(l *JsonlLine) {
		l.SessionID = "first"
		l.Cwd = ""
	})
	line2 := baseLine(func(l *JsonlLine) {
		l.SessionID = "second"
		l.Cwd = "/real"
	})
	meta := ExtractMetadata([]JsonlLine{line1, line2})
	assertEqual(t, "first", meta.SessionID, "sessionId")
	assertEqual(t, "/real", meta.WorkingDirectory, "cwd")
}

// --- BuildTurns tests ---

func TestBuildTurns_UserStringContent(t *testing.T) {
	turns := BuildTurns([]JsonlLine{userLine("Hello there")})
	assertLen(t, turns, 1)
	assertEqual(t, "user", turns[0].Type, "type")
	assertSliceEqual(t, []string{"Hello there"}, turns[0].Text, "text")
}

func TestBuildTurns_SkipEmptyString(t *testing.T) {
	turns := BuildTurns([]JsonlLine{userLine(""), userLine("   ")})
	assertLen(t, turns, 0)
}

func TestBuildTurns_AssistantTextBlocks(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{{Type: "text", Text: "Hi!"}}),
	})
	assertLen(t, turns, 1)
	assertEqual(t, "assistant", turns[0].Type, "type")
	assertSliceEqual(t, []string{"Hi!"}, turns[0].Text, "text")
}

func TestBuildTurns_ThinkingBlocks(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "thinking", Thinking: "Let me think..."},
			{Type: "text", Text: "Answer"},
		}),
	})
	assertSliceEqual(t, []string{"Let me think..."}, turns[0].Thinking, "thinking")
	assertSliceEqual(t, []string{"Answer"}, turns[0].Text, "text")
}

func TestBuildTurns_SkipEmptyTextAndThinking(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "text", Text: "   "},
			{Type: "thinking", Thinking: "  "},
			{Type: "text", Text: "real"},
		}),
	})
	assertSliceEqual(t, []string{"real"}, turns[0].Text, "text")
	assertLen(t, turns[0].Thinking, 0)
}

func TestBuildTurns_ToolLinking(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "tool_use", ID: "tool-1", Name: "Bash", Input: mustMarshal(map[string]string{"command": "ls"})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("tool-1", "file.txt"),
		),
	})
	assertLen(t, turns, 1)
	assertLen(t, turns[0].ToolCalls, 1)
	assertEqual(t, "Bash", turns[0].ToolCalls[0].Name, "name")
	assertPtrEqual(t, "file.txt", turns[0].ToolCalls[0].Result, "result")
}

func TestBuildTurns_ToolResultArrayContent(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "tool_use", ID: "tool-1", Name: "Read", Input: mustMarshal(map[string]interface{}{})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("tool-1", []map[string]string{
				{"type": "text", "text": "line1"},
				{"type": "text", "text": "line2"},
			}),
		),
	})
	assertPtrEqual(t, "line1\nline2", turns[0].ToolCalls[0].Result, "result")
}

func TestBuildTurns_ToolResultNoContent(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "tool_use", ID: "tool-1", Name: "Bash", Input: mustMarshal(map[string]interface{}{})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("tool-1", nil),
		),
	})
	assertPtrEqual(t, "", turns[0].ToolCalls[0].Result, "result")
}

func TestBuildTurns_ErrorResults(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "tool_use", ID: "tool-1", Name: "Bash", Input: mustMarshal(map[string]interface{}{})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("tool-1", "error msg", true),
		),
	})
	if !turns[0].ToolCalls[0].IsError {
		t.Error("expected IsError to be true")
	}
}

func TestBuildTurns_UserTextAlongsideToolResults(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{
			{Type: "tool_use", ID: "tool-1", Name: "Bash", Input: mustMarshal(map[string]interface{}{})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("tool-1", "ok"),
			textBlockJSON("User followup"),
		),
	})
	assertLen(t, turns, 2)
	assertEqual(t, "user", turns[1].Type, "type")
	assertSliceEqual(t, []string{"User followup"}, turns[1].Text, "text")
}

func TestBuildTurns_MergeConsecutiveAssistant(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{{Type: "text", Text: "Part 1"}}),
		assistantLine([]ContentBlock{{Type: "text", Text: "Part 2"}}),
	})
	assertLen(t, turns, 1)
	assertSliceEqual(t, []string{"Part 1", "Part 2"}, turns[0].Text, "text")
}

func TestBuildTurns_NoMergeWhenSeparatedByUser(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{{Type: "text", Text: "A1"}}),
		userLine("User msg"),
		assistantLine([]ContentBlock{{Type: "text", Text: "A2"}}),
	})
	assertLen(t, turns, 3)
	assertSliceEqual(t, []string{"A1"}, turns[0].Text, "text[0]")
	assertSliceEqual(t, []string{"A2"}, turns[2].Text, "text[2]")
}

func TestBuildTurns_SkipEmptyAssistant(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		assistantLine([]ContentBlock{{Type: "text", Text: "   "}}),
	})
	assertLen(t, turns, 0)
}

func TestBuildTurns_TeammateMessage(t *testing.T) {
	content := `<teammate-message teammate_id="researcher" summary="Found the issue">
I found the bug in line 42 of app.ts.
The variable is undefined.
</teammate-message>`
	turns := BuildTurns([]JsonlLine{userLine(content)})
	assertLen(t, turns, 1)
	assertEqual(t, "teammate", turns[0].Type, "type")
	assertEqual(t, "researcher", turns[0].TeammateName, "teammateName")
	assertSliceEqual(t, []string{"I found the bug in line 42 of app.ts.\nThe variable is undefined."}, turns[0].Text, "text")
}

func TestBuildTurns_FilterIdleNotification(t *testing.T) {
	content := `<teammate-message teammate_id="worker" summary="Status">
Working on the task.
{"type":"idle_notification","idle_since":"2025-01-01"}
Almost done.
</teammate-message>`
	turns := BuildTurns([]JsonlLine{userLine(content)})
	assertLen(t, turns, 1)
	assertEqual(t, "teammate", turns[0].Type, "type")
	assertSliceEqual(t, []string{"Working on the task.\nAlmost done."}, turns[0].Text, "text")
}

func TestBuildTurns_FilterShutdownNoiseJSON(t *testing.T) {
	content := `<teammate-message teammate_id="helper" summary="Shutting down">
{"type":"shutdown_request","requestId":"abc-123"}
{"type":"shutdown_approved","requestId":"abc-123"}
{"type":"teammate_terminated","teammate_id":"helper"}
Goodbye!
</teammate-message>`
	turns := BuildTurns([]JsonlLine{userLine(content)})
	assertLen(t, turns, 1)
	assertSliceEqual(t, []string{"Goodbye!"}, turns[0].Text, "text")
}

func TestBuildTurns_AllNoiseTeammateMessage(t *testing.T) {
	content := `<teammate-message teammate_id="helper" summary="Noise">
{"type":"idle_notification","idle_since":"2025-01-01"}
{"type":"shutdown_approved","requestId":"abc-123"}
</teammate-message>`
	turns := BuildTurns([]JsonlLine{userLine(content)})
	assertLen(t, turns, 0)
}

func TestBuildTurns_RegularUserNotTeammate(t *testing.T) {
	turns := BuildTurns([]JsonlLine{userLine("Fix the bug in teammate-message handler")})
	assertLen(t, turns, 1)
	assertEqual(t, "user", turns[0].Type, "type")
}

func TestBuildTurns_RealisticConversation(t *testing.T) {
	turns := BuildTurns([]JsonlLine{
		userLine("Fix the bug"),
		assistantLine([]ContentBlock{
			{Type: "thinking", Thinking: "I need to read the file"},
			{Type: "text", Text: "Let me look at the code."},
			{Type: "tool_use", ID: "t1", Name: "Read", Input: mustMarshal(map[string]string{"file_path": "src/app.ts"})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("t1", "const x = 1;"),
		),
		assistantLine([]ContentBlock{
			{Type: "text", Text: "I found the issue."},
			{Type: "tool_use", ID: "t2", Name: "Edit", Input: mustMarshal(map[string]string{"file_path": "src/app.ts"})},
		}),
		userLineWithRawBlocks(
			toolResultJSON("t2", "ok"),
		),
		assistantLine([]ContentBlock{{Type: "text", Text: "Done!"}}),
	})

	assertLen(t, turns, 2)

	assertEqual(t, "user", turns[0].Type, "turns[0].type")
	assertSliceEqual(t, []string{"Fix the bug"}, turns[0].Text, "turns[0].text")

	assertEqual(t, "assistant", turns[1].Type, "turns[1].type")
	assertSliceEqual(t, []string{"I need to read the file"}, turns[1].Thinking, "turns[1].thinking")
	assertSliceEqual(t, []string{"Let me look at the code.", "I found the issue.", "Done!"}, turns[1].Text, "turns[1].text")
	assertLen(t, turns[1].ToolCalls, 2)
	assertPtrEqual(t, "const x = 1;", turns[1].ToolCalls[0].Result, "tool[0].result")
	assertPtrEqual(t, "ok", turns[1].ToolCalls[1].Result, "tool[1].result")
}

// --- Test helpers ---

func assertEqual(t *testing.T, expected, actual, label string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %q, got %q", label, expected, actual)
	}
}

func assertLen(t *testing.T, slice interface{}, expected int) {
	t.Helper()
	var got int
	switch s := slice.(type) {
	case []ConversationTurn:
		got = len(s)
	case []LinkedToolCall:
		got = len(s)
	case []string:
		got = len(s)
	default:
		t.Fatalf("assertLen: unsupported type %T", slice)
	}
	if got != expected {
		t.Errorf("expected length %d, got %d", expected, got)
	}
}

func assertSliceEqual(t *testing.T, expected, actual []string, label string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("%s: expected %v, got %v", label, expected, actual)
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("%s[%d]: expected %q, got %q", label, i, expected[i], actual[i])
		}
	}
}

func assertPtrEqual(t *testing.T, expected string, actual *string, label string) {
	t.Helper()
	if actual == nil {
		t.Errorf("%s: expected %q, got nil", label, expected)
		return
	}
	if *actual != expected {
		t.Errorf("%s: expected %q, got %q", label, expected, *actual)
	}
}
