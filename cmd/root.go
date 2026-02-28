package cmd

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/magarcia/ccsession-viewer/discovery"
	"github.com/magarcia/ccsession-viewer/formatter"
	"github.com/magarcia/ccsession-viewer/internal/pager"
	"github.com/magarcia/ccsession-viewer/parser"
	"github.com/magarcia/ccsession-viewer/tui"
)

var (
	appVersion string

	lastFlag     int
	outputFlag   string
	styleFlag    string
	widthFlag    int
	thinkingFlag bool
	collapseFlag bool
	maxLinesFlag int
	rawFlag      bool
	markdownFlag string
	noPagerFlag  bool
)

func SetVersionInfo(version, commit, date string) {
	appVersion = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

var rootCmd = &cobra.Command{
	Use:   "cc2md [options] [file]",
	Short: "Convert Claude Code session logs to shareable markdown",
	Long:  "Convert Claude Code session logs (.jsonl) into clean, shareable markdown with terminal rendering via glamour.",
	Args:  cobra.MaximumNArgs(1),
	Example: `  # Open interactive session picker
  cc2md

  # Convert the most recent session (TTY output with pager)
  cc2md --last 1

  # Convert a specific file to a markdown file
  cc2md session.jsonl --output session.md

  # Output GFM markdown to stdout
  cc2md --raw

  # List all sessions as JSON
  cc2md list --json

  # Filter sessions by project
  cc2md list myproject`,
}

func init() {
	rootCmd.RunE = runRoot
	rootCmd.Flags().IntVarP(&lastFlag, "last", "n", 1, "Convert the Nth most recent session")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write to file instead of stdout")
	rootCmd.Flags().StringVarP(&styleFlag, "style", "s", "auto", "Glamour style: auto, dark, light, notty")
	rootCmd.Flags().IntVarP(&widthFlag, "width", "w", 0, "Word wrap width (default: terminal width)")
	rootCmd.Flags().BoolVarP(&thinkingFlag, "thinking", "t", false, "Include thinking blocks")
	rootCmd.Flags().BoolVarP(&collapseFlag, "collapse", "c", true, "Collapse tool calls and thinking into <details> tags")
	rootCmd.Flags().IntVar(&maxLinesFlag, "max-lines", 100, "Max lines per tool output before truncation")
	rootCmd.Flags().BoolVar(&rawFlag, "raw", false, "Output raw markdown (skip glamour rendering)")
	rootCmd.Flags().StringVarP(&markdownFlag, "markdown", "m", "", "Markdown flavor: gfm, commonmark")
	rootCmd.Flags().BoolVar(&noPagerFlag, "no-pager", false, "Disable pager even on TTY")
}

func runRoot(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("last") && len(args) > 0 {
		return fmt.Errorf("cannot use --last with a file argument")
	}
	if markdownFlag != "" {
		if _, err := formatter.ParseFlavor(markdownFlag); err != nil {
			return err
		}
	}

	path, err := resolveSessionPath(cmd, args)
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	return renderSession(path, cmd)
}

func resolveSessionPath(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	if cmd.Flags().Changed("last") {
		sessions := discovery.ListSessions("")
		if len(sessions) == 0 {
			return "", fmt.Errorf("no sessions found in ~/.claude/projects/")
		}
		if lastFlag > len(sessions) {
			return "", fmt.Errorf("only %d session(s) found, requested #%d", len(sessions), lastFlag)
		}
		return sessions[lastFlag-1].Path, nil
	}

	if isStdinTTY() && isStdoutTTY() {
		sessions := discovery.ListSessions("")
		if len(sessions) == 0 {
			fmt.Fprintln(os.Stderr, "No sessions found")
			os.Exit(1)
		}
		selected, err := tui.PickSession(sessions)
		if err != nil {
			return "", fmt.Errorf("session selector: %w", err)
		}
		return selected, nil
	}

	if !isStdinTTY() {
		fmt.Fprintln(os.Stderr, "No session file specified. Run cc2md --help for usage.")
		os.Exit(1)
	}

	// TTY stdin, non-TTY stdout, no --last: use most recent
	sessions := discovery.ListSessions("")
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found in ~/.claude/projects/")
	}
	return sessions[0].Path, nil
}

func renderSession(filePath string, cmd *cobra.Command) error {
	lines, err := parser.ReadSessionFile(filePath)
	if err != nil {
		return fmt.Errorf("reading session: %w", err)
	}

	meta := parser.ExtractMetadata(lines)
	turns := parser.BuildTurns(lines)
	flavor := determineFlavor(cmd)

	md := formatter.FormatSession(meta, turns, formatter.FormatOptions{
		IncludeThinking: thinkingFlag,
		Collapse:        collapseFlag,
		MaxLines:        maxLinesFlag,
		Flavor:          flavor,
	})

	if outputFlag != "" {
		if err := os.WriteFile(outputFlag, []byte(md), 0o644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Written to %s\n", outputFlag)
		return nil
	}

	if rawFlag || !isStdoutTTY() {
		_, err := os.Stdout.WriteString(md)
		return err
	}

	rendered, err := renderWithGlamour(md, flavor)
	if err != nil {
		_, err = os.Stdout.WriteString(md)
		return err
	}

	if noPagerFlag {
		_, err = os.Stdout.WriteString(rendered)
		return err
	}

	p, w, err := pager.Start(os.Stdout)
	if err != nil {
		_, err = os.Stdout.WriteString(rendered)
		return err
	}
	if p != nil {
		defer p.Stop()
	}

	if _, err = fmt.Fprint(w, rendered); err != nil {
		if errors.As(err, &pager.ErrClosedPagerPipe{}) {
			return nil
		}
		return err
	}
	return nil
}

func determineFlavor(cmd *cobra.Command) formatter.MarkdownFlavor {
	if markdownFlag != "" {
		f, _ := formatter.ParseFlavor(markdownFlag)
		return f
	}
	if outputFlag != "" || rawFlag {
		return formatter.FlavorGFM
	}
	if isStdoutTTY() {
		return formatter.FlavorCommonMark
	}
	return formatter.FlavorGFM
}

func isStdoutTTY() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
func isStdinTTY() bool  { return term.IsTerminal(int(os.Stdin.Fd())) }

func getTerminalWidth() int {
	if widthFlag > 0 {
		return widthFlag
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func glamourStyle() glamour.TermRendererOption {
	switch styleFlag {
	case "dark":
		return glamour.WithStylePath("dark")
	case "light":
		return glamour.WithStylePath("light")
	case "notty":
		return glamour.WithStylePath("notty")
	default:
		return glamour.WithAutoStyle()
	}
}

func renderWithGlamour(md string, flavor formatter.MarkdownFlavor) (string, error) {
	processed := md
	if flavor != formatter.FlavorCommonMark {
		processed = preprocessAlerts(processed)
		processed = preprocessDetails(processed)
	}

	renderer, err := glamour.NewTermRenderer(
		glamourStyle(),
		glamour.WithWordWrap(getTerminalWidth()),
	)
	if err != nil {
		return "", err
	}

	return renderer.Render(processed)
}

var alertBlockRe = regexp.MustCompile(`(?m)(?:^> \[!(NOTE|TIP)\]\n)((?:^>.*\n?)*)`)

func preprocessAlerts(md string) string {
	return alertBlockRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := alertBlockRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}

		alertType := sub[1]
		body := sub[2]

		var lines []string
		for _, line := range strings.Split(body, "\n") {
			switch {
			case strings.HasPrefix(line, "> "):
				lines = append(lines, line[2:])
			case line == ">":
				lines = append(lines, "")
			case line != "":
				lines = append(lines, line)
			}
		}
		content := strings.TrimSpace(strings.Join(lines, "\n"))

		var color lipgloss.Color
		var icon, label string
		switch alertType {
		case "NOTE":
			color = lipgloss.Color("12")
			icon, label = "\U0001f4dd", "Note"
		case "TIP":
			color = lipgloss.Color("10")
			icon, label = "\U0001f4a1", "Tip"
		}

		header := lipgloss.NewStyle().Bold(true).Foreground(color).Render(icon + " " + label)
		styled := lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(color).
			PaddingLeft(1).
			Render(header + "\n" + content)

		return styled + "\n\n"
	})
}

var detailsRe = regexp.MustCompile(`(?s)<details>\s*\n?\s*<summary>(.*?)</summary>\s*\n?(.*?)\s*</details>`)

func preprocessDetails(md string) string {
	return detailsRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := detailsRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		summary := strings.TrimSpace(sub[1])
		inner := strings.TrimSpace(sub[2])

		result := "**" + summary + "**"
		if inner != "" {
			result += "\n\n" + inner
		}
		return result
	})
}

func Execute() {
	rootCmd.Version = appVersion
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
