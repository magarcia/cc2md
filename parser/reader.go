package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var skipTypes = map[string]bool{
	"progress":              true,
	"system":                true,
	"file-history-snapshot": true,
}

// ReadSessionFile reads a JSONL session file and returns filtered lines.
func ReadSessionFile(filePath string) ([]JsonlLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var lines []JsonlLine
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		var line JsonlLine
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			continue
		}

		if skipTypes[line.Type] {
			continue
		}
		if line.IsMeta {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning session file: %w", err)
	}

	return lines, nil
}
