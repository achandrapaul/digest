package tui

import (
	"slices"
	"strings"

	"github.com/achandrapaul/digest/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func noteLinks(note *model.Note) []string {
	if note == nil {
		return nil
	}
	var links []string
	for _, link := range noteLinkPattern.FindAllString(note.Summary+"\n"+note.Body, -1) {
		link = strings.TrimRight(link, ".,;:")
		if !slices.Contains(links, link) {
			links = append(links, link)
		}
	}
	return links
}

func noteLinksLabel(links []string, onDashboard bool) string {
	label := "links"
	if len(links) == 1 {
		label = "link"
	}
	if onDashboard {
		return label
	}
	return "open " + label
}

func (m Model) openNoteLinks(tea.KeyMsg) (tea.Model, tea.Cmd) {
	item, ok := m.selectedNavItem()
	if !ok {
		return m, nil
	}
	links := noteLinks(item.Note)
	switch len(links) {
	case 0:
	case 1:
		m.openLink(links[0])
	default:
		m.notesUI.linkMenuItems = links
		m.notesUI.linkMenuSelected = 0
		m.pushMode(ViewLinkMenu)
	}
	return m, nil
}

func (m Model) linkMenuDown(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notesUI.linkMenuSelected = min(m.notesUI.linkMenuSelected+1, max(len(m.notesUI.linkMenuItems)-1, 0))
	return m, nil
}

func (m Model) linkMenuUp(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notesUI.linkMenuSelected = max(m.notesUI.linkMenuSelected-1, 0)
	return m, nil
}

func (m Model) closeLinkMenu(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = m.popMode()
	m.notesUI.linkMenuItems = nil
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
	return m, nil
}

func (m Model) chooseLink(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.notesUI.linkMenuSelected < len(m.notesUI.linkMenuItems) {
		m.openLink(m.notesUI.linkMenuItems[m.notesUI.linkMenuSelected])
	}
	return m.closeLinkMenu(msg)
}

func (m Model) renderLinkDropdown() string {
	maxLinkWidth := max(m.width-12, 20)
	var rows []string
	for index, link := range m.notesUI.linkMenuItems {
		label := ansi.Truncate(link, maxLinkWidth, "…")
		if index == m.notesUI.linkMenuSelected {
			rows = append(rows, hintKeyStyle.Render("› "+label))
		} else {
			rows = append(rows, hintTextStyle.Render("  "+label))
		}
	}
	return hintPillStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderLinkMenu() string {
	dropdown := m.renderLinkDropdown()
	return m.overlayUnderSelectedRow(dropdown, m.rightAlignedColumn(dropdown))
}

func linkMenuBindings() []keyBinding {
	return []keyBinding{
		newKeyBinding(actionChooseLink, []string{"enter"}, "enter", "open"),
		newKeyBinding(actionLinkMenuDown, []string{"j", "down"}, "j|k", "nav"),
		hiddenKeyBinding(actionLinkMenuUp, "k", "up"),
		newKeyBinding(actionCloseLinkMenu, []string{"esc", "n", "N"}, "esc", "cancel"),
	}
}
