package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/magarcia/ccsession-viewer/discovery"
	"github.com/magarcia/ccsession-viewer/formatter"
	"github.com/magarcia/ccsession-viewer/vault"
)

var (
	syncDirFlag      string
	syncHookFlag     bool
	syncForceFlag    bool
	syncThinkingFlag bool
	syncMaxLinesFlag int
)

var syncCmd = &cobra.Command{
	Use:   "sync [project]",
	Short: "Export sessions as one markdown note each into a folder (e.g. an Obsidian vault)",
	Long: `Export Claude Code sessions as one markdown note per session into a folder,
such as an Obsidian vault. Notes get YAML frontmatter and are named
"<date> <time> <project> - <title>.md". Only sessions that changed since the
last export are rewritten; notes are renamed when the session title changes.

With --hook, the session to export is read from the JSON a Claude Code hook
receives on stdin, so a Stop or SessionEnd hook keeps the folder current.`,
	Example: `  # Export all sessions
  cc2md sync --dir ~/Obsidian/Vault/Claude\ Code

  # Export only sessions of matching projects
  cc2md sync myproject --dir ~/notes/claude

  # As a Claude Code Stop or SessionEnd hook command
  cc2md sync --hook --dir ~/notes/claude`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVarP(&syncDirFlag, "dir", "d", "", "Folder to write the notes to (required)")
	syncCmd.Flags().BoolVar(&syncHookFlag, "hook", false, "Export the session named by Claude Code hook JSON on stdin")
	syncCmd.Flags().BoolVar(&syncForceFlag, "force", false, "Rewrite notes even if the session is unchanged")
	syncCmd.Flags().BoolVarP(&syncThinkingFlag, "thinking", "t", true, "Include thinking blocks")
	syncCmd.Flags().IntVar(&syncMaxLinesFlag, "max-lines", 100, "Max lines per tool output before truncation")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if syncDirFlag == "" {
		return errors.New("--dir is required")
	}
	dir, err := expandHome(syncDirFlag)
	if err != nil {
		return err
	}

	idx, err := vault.LoadIndex(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}
	opts := formatter.FormatOptions{
		IncludeThinking: syncThinkingFlag,
		Collapse:        true,
		MaxLines:        syncMaxLinesFlag,
		Flavor:          formatter.FlavorGFM,
	}

	if syncHookFlag {
		path, err := hookTranscriptPath(os.Stdin)
		if err != nil {
			return err
		}
		_, _, err = vault.Export(path, dir, idx, opts, syncForceFlag)
		return err
	}

	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}
	counts := map[vault.Status]int{}
	var failed int
	for _, s := range discovery.ListSessions(filter) {
		name, status, err := vault.Export(s.Path, dir, idx, opts, syncForceFlag)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "%s: %v\n", s.Path, err)
			continue
		}
		counts[status]++
		if status == vault.Written || status == vault.Renamed {
			fmt.Fprintf(os.Stderr, "%-9s %s\n", status, name)
		}
	}
	fmt.Fprintf(os.Stderr, "%d written, %d renamed, %d unchanged, %d empty, %d failed → %s\n",
		counts[vault.Written], counts[vault.Renamed], counts[vault.Unchanged], counts[vault.Empty], failed, dir)
	if failed > 0 {
		return fmt.Errorf("%d session(s) failed", failed)
	}
	return nil
}

// hookTranscriptPath reads the transcript path from Claude Code hook input.
func hookTranscriptPath(r io.Reader) (string, error) {
	var in struct {
		TranscriptPath string `json:"transcript_path"`
	}
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		return "", fmt.Errorf("reading hook input: %w", err)
	}
	if in.TranscriptPath == "" {
		return "", errors.New("hook input has no transcript_path")
	}
	return expandHome(in.TranscriptPath)
}

func expandHome(path string) (string, error) {
	if path == "~" || len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}
