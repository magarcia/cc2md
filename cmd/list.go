package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/magarcia/ccsession-viewer/discovery"
)

type sessionJSON struct {
	SessionID  string `json:"session_id"`
	Name       string `json:"name"`
	Project    string `json:"project"`
	Path       string `json:"path"`
	ModifiedAt string `json:"modified_at"`
	SizeBytes  int64  `json:"size_bytes"`
}

var jsonFlag bool

var listCmd = &cobra.Command{
	Use:   "list [project]",
	Short: "List available Claude Code sessions",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON array")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	sessions := discovery.ListSessions(filter)
	if jsonFlag {
		return printSessionsJSON(sessions)
	}

	fmt.Println(discovery.FormatSessionList(sessions))
	return nil
}

func printSessionsJSON(sessions []discovery.SessionEntry) error {
	out := make([]sessionJSON, len(sessions))
	for i, s := range sessions {
		out[i] = sessionJSON{
			SessionID:  s.SessionID,
			Name:       s.Name,
			Project:    s.Project,
			Path:       s.Path,
			ModifiedAt: s.ModifiedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			SizeBytes:  s.Size,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
