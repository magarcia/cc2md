package formatter

import (
	"fmt"
	"time"

	"github.com/magarcia/ccsession-viewer/parser"
)

func FormatMetadata(meta parser.SessionMetadata, flavor MarkdownFlavor) string {
	date := meta.Date
	if t, err := time.Parse(time.RFC3339Nano, meta.Date); err == nil {
		date = t.Local().Format("01/02/2006, 15:04")
	}

	var legend string
	if flavor == FlavorCommonMark {
		legend = `> **Note:** Blue blocks are **user** messages

> **Tip:** Green blocks are **agent** (teammate) reports

> Plain quoted lines are **Claude** responses`
	} else {
		legend = `> [!NOTE]
> Blue blocks are **user** messages

> [!TIP]
> Green blocks are **agent** (teammate) reports

> Plain quoted lines are **Claude** responses`
	}

	return fmt.Sprintf(`# Session

%s

| Field | Value |
|-------|-------|
| Date | %s |
| Model | %s |
| Working Directory | %s |
| Session | %s |
| Claude Code | v%s |`, legend, date, meta.Model, meta.WorkingDirectory, meta.SessionID, meta.Version)
}
