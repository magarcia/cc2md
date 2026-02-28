package parser

import (
	"encoding/json"
	"regexp"
	"strings"
)

var noiseTypes = map[string]bool{
	"idle_notification":   true,
	"shutdown_approved":   true,
	"shutdown_request":    true,
	"teammate_terminated": true,
}

var (
	teammateMsgRe    = regexp.MustCompile(`(?s)<teammate-message\s[^>]*teammate_id="([^"]*)"[^>]*>(.*?)</teammate-message>`)
	taskNotifSummary = regexp.MustCompile(`(?s)<summary>(.*?)</summary>`)
	taskNotifResult  = regexp.MustCompile(`(?s)<result>(.*?)</result>`)
	agentNameRe      = regexp.MustCompile(`Agent "([^"]+)"`)
	commandNameRe    = regexp.MustCompile(`<command-name>/?(.*?)</command-name>`)
	stdoutRe         = regexp.MustCompile(`(?s)<local-command-stdout>(.*?)</local-command-stdout>`)
	stderrRe         = regexp.MustCompile(`(?s)<local-command-stderr>(.*?)</local-command-stderr>`)
)

// ExtractMetadata iterates lines and extracts session metadata.
func ExtractMetadata(lines []JsonlLine) SessionMetadata {
	meta := SessionMetadata{Model: "unknown"}

	for _, line := range lines {
		if meta.SessionID == "" && line.SessionID != "" {
			meta.SessionID = line.SessionID
		}
		if meta.WorkingDirectory == "" && line.Cwd != "" {
			meta.WorkingDirectory = line.Cwd
		}
		if meta.Version == "" && line.Version != "" {
			meta.Version = line.Version
		}
		if meta.Date == "" && line.Timestamp != "" {
			meta.Date = line.Timestamp
		}

		if line.Type == "assistant" {
			var msg AssistantMessage
			if err := json.Unmarshal(line.Message, &msg); err == nil && msg.Model != "" {
				meta.Model = msg.Model
				break
			}
		}
	}

	return meta
}

// BuildTurns processes JSONL lines into conversation turns, linking tool calls with results.
func BuildTurns(lines []JsonlLine) []ConversationTurn {
	var turns []ConversationTurn
	pendingToolCalls := make(map[string]*LinkedToolCall)

	for _, line := range lines {
		switch line.Type {
		case "user":
			processUserLine(line, &turns, pendingToolCalls)
		case "assistant":
			processAssistantLine(line, &turns, pendingToolCalls)
		}
	}

	return turns
}

func processUserLine(line JsonlLine, turns *[]ConversationTurn, pendingToolCalls map[string]*LinkedToolCall) {
	var msg UserMessage
	if err := json.Unmarshal(line.Message, &msg); err != nil {
		return
	}

	// Try array content first (tool results)
	if blocks, err := msg.ContentAsBlocks(); err == nil && blocks != nil {
		// Process tool results
		for i := range blocks {
			if blocks[i].Type == "tool_result" {
				if pending, ok := pendingToolCalls[blocks[i].ToolUseID]; ok {
					result := extractResultText(&blocks[i])
					pending.Result = &result
					pending.IsError = blocks[i].IsError
					delete(pendingToolCalls, blocks[i].ToolUseID)
				}
			}
		}

		// Check for user text alongside tool results
		var textParts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		}
		if len(textParts) > 0 {
			*turns = append(*turns, ConversationTurn{
				Type:      "user",
				Text:      textParts,
				Timestamp: line.Timestamp,
			})
		}
		return
	}

	// Try string content
	content, ok := msg.ContentAsString()
	if !ok || strings.TrimSpace(content) == "" {
		return
	}

	if isAgentMessage(content) {
		agentTurn := parseAgentMessage(content)
		if agentTurn != nil {
			agentTurn.Timestamp = line.Timestamp
			*turns = append(*turns, *agentTurn)
		}
	} else if isLocalCommand(content) {
		cmdTurn := parseLocalCommand(content)
		if cmdTurn != nil {
			if n := len(*turns); n > 0 && (*turns)[n-1].Type == "local-command" {
				(*turns)[n-1].Text = append((*turns)[n-1].Text, cmdTurn.Text...)
			} else {
				cmdTurn.Timestamp = line.Timestamp
				*turns = append(*turns, *cmdTurn)
			}
		}
	} else {
		*turns = append(*turns, ConversationTurn{
			Type:      "user",
			Text:      []string{content},
			Timestamp: line.Timestamp,
		})
	}
}

func processAssistantLine(line JsonlLine, turns *[]ConversationTurn, pendingToolCalls map[string]*LinkedToolCall) {
	var msg AssistantMessage
	if err := json.Unmarshal(line.Message, &msg); err != nil {
		return
	}

	turn := ConversationTurn{
		Type:      "assistant",
		Timestamp: line.Timestamp,
	}

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				turn.Text = append(turn.Text, block.Text)
			}
		case "tool_use":
			linked := LinkedToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.InputAsMap(),
			}
			turn.ToolCalls = append(turn.ToolCalls, linked)
			pendingToolCalls[block.ID] = &turn.ToolCalls[len(turn.ToolCalls)-1]
		case "thinking":
			if strings.TrimSpace(block.Thinking) != "" {
				turn.Thinking = append(turn.Thinking, block.Thinking)
			}
		}
	}

	// Merge into previous assistant turn if consecutive
	if n := len(*turns); n > 0 && (*turns)[n-1].Type == "assistant" {
		(*turns)[n-1].Text = append((*turns)[n-1].Text, turn.Text...)
		(*turns)[n-1].ToolCalls = append((*turns)[n-1].ToolCalls, turn.ToolCalls...)
		(*turns)[n-1].Thinking = append((*turns)[n-1].Thinking, turn.Thinking...)
		// Re-register ALL pending tool call pointers — append may have reallocated the slice
		mergedTurn := &(*turns)[n-1]
		for i := range mergedTurn.ToolCalls {
			if _, ok := pendingToolCalls[mergedTurn.ToolCalls[i].ID]; ok {
				pendingToolCalls[mergedTurn.ToolCalls[i].ID] = &mergedTurn.ToolCalls[i]
			}
		}
	} else if len(turn.Text) > 0 || len(turn.ToolCalls) > 0 || len(turn.Thinking) > 0 {
		*turns = append(*turns, turn)
		// Re-register pending tool calls to point to the actual slice element
		lastTurn := &(*turns)[len(*turns)-1]
		for i := range lastTurn.ToolCalls {
			pendingToolCalls[lastTurn.ToolCalls[i].ID] = &lastTurn.ToolCalls[i]
		}
	}
}

func extractResultText(block *ContentBlock) string {
	if s, ok := block.ResultContentAsString(); ok {
		return s
	}
	if blocks, err := block.ResultContentAsBlocks(); err == nil && blocks != nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func isNoiseJson(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return false
	}
	if t, ok := parsed["type"].(string); ok {
		return noiseTypes[t]
	}
	return false
}

func isAgentMessage(content string) bool {
	return strings.Contains(content, "<teammate-message") || strings.Contains(content, "<task-notification>")
}

func parseAgentMessage(content string) *ConversationTurn {
	if turn := parseTeammateMessage(content); turn != nil {
		return turn
	}
	return parseTaskNotification(content)
}

func parseTeammateMessage(content string) *ConversationTurn {
	if !strings.Contains(content, "<teammate-message") {
		return nil
	}

	match := teammateMsgRe.FindStringSubmatch(content)
	if match == nil {
		return nil
	}

	teammateName := match[1]
	body := match[2]

	var filtered []string
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if isNoiseJson(line) {
			continue
		}
		filtered = append(filtered, line)
	}

	if len(filtered) == 0 {
		return nil
	}

	return &ConversationTurn{
		Type:         "teammate",
		Text:         []string{strings.Join(filtered, "\n")},
		TeammateName: teammateName,
	}
}

func parseTaskNotification(content string) *ConversationTurn {
	if !strings.Contains(content, "<task-notification>") {
		return nil
	}

	resultMatch := taskNotifResult.FindStringSubmatch(content)
	if resultMatch == nil {
		return nil
	}

	summaryMatch := taskNotifSummary.FindStringSubmatch(content)
	summary := ""
	if summaryMatch != nil {
		summary = summaryMatch[1]
	}

	nameMatch := agentNameRe.FindStringSubmatch(summary)
	teammateName := "agent"
	if nameMatch != nil {
		teammateName = nameMatch[1]
	}

	body := strings.TrimSpace(resultMatch[1])
	if body == "" {
		return nil
	}

	return &ConversationTurn{
		Type:         "teammate",
		Text:         []string{body},
		TeammateName: teammateName,
	}
}

func isLocalCommand(content string) bool {
	return strings.Contains(content, "<local-command-stdout>") ||
		strings.Contains(content, "<local-command-stderr>") ||
		strings.Contains(content, "<command-name>")
}

func parseLocalCommand(content string) *ConversationTurn {
	commandMatch := commandNameRe.FindStringSubmatch(content)
	commandName := ""
	if commandMatch != nil {
		commandName = strings.TrimSpace(commandMatch[1])
	}

	stdoutMatch := stdoutRe.FindStringSubmatch(content)
	output := ""
	if stdoutMatch != nil {
		output = strings.TrimSpace(stdoutMatch[1])
	}

	stderrMatch := stderrRe.FindStringSubmatch(content)
	errOutput := ""
	if stderrMatch != nil {
		errOutput = strings.TrimSpace(stderrMatch[1])
	}

	var parts []string
	if output != "" {
		parts = append(parts, output)
	}
	if errOutput != "" {
		parts = append(parts, errOutput)
	}
	combined := strings.Join(parts, "\n")

	if commandName == "" && combined == "" {
		return nil
	}

	var text string
	if commandName != "" {
		text = "/" + commandName
		if combined != "" {
			text += "\n" + combined
		}
	} else {
		text = combined
	}

	return &ConversationTurn{
		Type: "local-command",
		Text: []string{text},
	}
}
