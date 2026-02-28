package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/magarcia/ccsession-viewer/discovery"
)

func makeEntry(name, project, path string) discovery.SessionEntry {
	return discovery.SessionEntry{
		Name:       name,
		Project:    project,
		Path:       path,
		SessionID:  "test-id",
		ModifiedAt: time.Date(2026, 2, 28, 10, 30, 0, 0, time.UTC),
	}
}

func makeModel(sessions []discovery.SessionEntry) model {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{entry: s}
	}
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 80, 24)
	l.DisableQuitKeybindings()
	return model{list: l}
}

func TestSessionItemFilterValue(t *testing.T) {
	item := sessionItem{entry: makeEntry("My Session", "my-project", "/path/to/file.jsonl")}
	got := item.FilterValue()
	if !strings.Contains(got, "My Session") {
		t.Errorf("FilterValue() = %q, want to contain name", got)
	}
	if !strings.Contains(got, "my-project") {
		t.Errorf("FilterValue() = %q, want to contain project", got)
	}
}

func TestSessionItemTitle(t *testing.T) {
	item := sessionItem{entry: makeEntry("My Session", "my-project", "/path/to/file.jsonl")}
	if got := item.Title(); got != "My Session" {
		t.Errorf("Title() = %q, want %q", got, "My Session")
	}
}

func TestSessionItemDescription(t *testing.T) {
	item := sessionItem{entry: makeEntry("My Session", "my-project", "/path/to/file.jsonl")}
	desc := item.Description()
	if !strings.Contains(desc, "2026-02-28 10:30") {
		t.Errorf("Description() = %q, want to contain formatted date", desc)
	}
	if !strings.Contains(desc, "my-project") {
		t.Errorf("Description() = %q, want to contain project", desc)
	}
}

func TestPickSession_Empty(t *testing.T) {
	path, err := PickSession(nil)
	if err != nil {
		t.Fatalf("PickSession(nil) error = %v", err)
	}
	if path != "" {
		t.Errorf("PickSession(nil) = %q, want empty", path)
	}

	path, err = PickSession([]discovery.SessionEntry{})
	if err != nil {
		t.Fatalf("PickSession([]) error = %v", err)
	}
	if path != "" {
		t.Errorf("PickSession([]) = %q, want empty", path)
	}
}

func TestModel_EnterSetsSelected(t *testing.T) {
	m := makeModel([]discovery.SessionEntry{
		makeEntry("Session One", "project-a", "/path/one.jsonl"),
		makeEntry("Session Two", "project-b", "/path/two.jsonl"),
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := result.(model)
	if final.selected != "/path/one.jsonl" {
		t.Errorf("selected = %q, want %q", final.selected, "/path/one.jsonl")
	}
	if !final.quitting {
		t.Error("quitting should be true after Enter")
	}
}

func TestModel_CtrlCQuitsWithNoSelection(t *testing.T) {
	m := makeModel([]discovery.SessionEntry{
		makeEntry("Session One", "project-a", "/path/one.jsonl"),
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	final := result.(model)
	if final.selected != "" {
		t.Errorf("selected = %q, want empty after ctrl+c", final.selected)
	}
	if !final.quitting {
		t.Error("quitting should be true after ctrl+c")
	}
}

func TestModel_QKeyQuits(t *testing.T) {
	m := makeModel([]discovery.SessionEntry{
		makeEntry("Session One", "project-a", "/path/one.jsonl"),
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	final := result.(model)
	if final.selected != "" {
		t.Errorf("selected = %q, want empty after q", final.selected)
	}
	if !final.quitting {
		t.Error("quitting should be true after q")
	}
}

func TestModel_ViewEmptyWhenQuitting(t *testing.T) {
	m := makeModel([]discovery.SessionEntry{
		makeEntry("Session One", "project-a", "/path/one.jsonl"),
	})
	m.quitting = true

	if got := m.View(); got != "" {
		t.Errorf("View() when quitting = %q, want empty", got)
	}
}
