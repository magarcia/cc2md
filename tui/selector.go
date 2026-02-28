package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/magarcia/ccsession-viewer/discovery"
)

type sessionItem struct {
	entry discovery.SessionEntry
}

func (s sessionItem) FilterValue() string {
	return s.entry.Name + " " + s.entry.Project
}

func (s sessionItem) Title() string {
	return s.entry.Name
}

func (s sessionItem) Description() string {
	return s.entry.ModifiedAt.Format("2006-01-02 15:04") + "  " + s.entry.Project
}

type model struct {
	list     list.Model
	selected string
	quitting bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "enter":
				if item, ok := m.list.SelectedItem().(sessionItem); ok {
					m.selected = item.entry.Path
				}
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func PickSession(sessions []discovery.SessionEntry) (string, error) {
	if len(sessions) == 0 {
		return "", nil
	}

	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{entry: s}
	}

	delegate := list.NewDefaultDelegate()
	highlight := lipgloss.Color("#7D56F4")
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(highlight).
		BorderLeftForeground(highlight)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(highlight).
		BorderLeftForeground(highlight)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a session  (/ to filter, enter to open)"
	l.DisableQuitKeybindings()

	p := tea.NewProgram(model{list: l}, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running session selector: %w", err)
	}

	if final, ok := result.(model); ok {
		return final.selected, nil
	}
	return "", nil
}
