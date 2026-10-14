package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/model"

	tea "github.com/charmbracelet/bubbletea"
)

type noteRangeAction int

const (
	noteRangeNone noteRangeAction = iota
	noteRangeDone
	noteRangeActive
	noteRangeStartToday
	noteRangeBackToPending
)

func (m *Model) clearNoteRange() {
	m.notesUI.rangeActive = false
	m.notesUI.rangeAnchor = 0
}

func (m Model) noteRangeBounds() (low, high int, active bool) {
	if !m.notesUI.rangeActive {
		return m.selected, m.selected, false
	}
	return min(m.notesUI.rangeAnchor, m.selected), max(m.notesUI.rangeAnchor, m.selected), true
}

func (m Model) noteRangeFrameAnchor() int {
	if !m.notesUI.rangeActive {
		return -1
	}
	return m.notesUI.rangeAnchor
}

func (m Model) rangeNotes() []*model.Note {
	if !m.notesUI.rangeActive {
		if item, ok := m.selectedNavItem(); ok && item.Note != nil {
			return []*model.Note{item.Note}
		}
		return nil
	}
	items := m.allNavItems()
	low, high, _ := m.noteRangeBounds()
	var notes []*model.Note
	for _, item := range items[min(max(low, 0), len(items)):min(high+1, len(items))] {
		if item.Note != nil {
			notes = append(notes, item.Note)
		}
	}
	return notes
}

func (m Model) extendNoteRange(delta int) (tea.Model, tea.Cmd) {
	navItems := m.allNavItems()
	if len(navItems) == 0 {
		return m, nil
	}
	if !m.notesUI.rangeActive {
		m.notesUI.rangeAnchor, m.notesUI.rangeActive = m.selected, true
	}
	m.selected = max(min(m.selected+delta, len(navItems)-1), 0)
	m.updateScrollOffset()
	return m, nil
}

func (m Model) extendNoteRangeDown(tea.KeyMsg) (tea.Model, tea.Cmd) { return m.extendNoteRange(1) }

func (m Model) extendNoteRangeUp(tea.KeyMsg) (tea.Model, tea.Cmd) { return m.extendNoteRange(-1) }

func doneRangeTargets(notes []*model.Note) ([]*model.Note, noteRangeAction) {
	active := slices.DeleteFunc(slices.Clone(notes), func(n *model.Note) bool { return n.Status == model.StatusDone })
	if len(active) > 0 {
		return active, noteRangeDone
	}
	return notes, noteRangeActive
}

func (m Model) workOnRangeTargets(notes []*model.Note, now time.Time) ([]*model.Note, noteRangeAction) {
	eligible := slices.DeleteFunc(slices.Clone(notes), func(n *model.Note) bool { return !m.canWorkOn(n) })
	notStarted := slices.DeleteFunc(slices.Clone(eligible), func(n *model.Note) bool { return n.StartedOn(now) })
	if len(notStarted) > 0 {
		return notStarted, noteRangeStartToday
	}
	return eligible, noteRangeBackToPending
}

func noteRangePrompt(action noteRangeAction, targets []*model.Note) (title, prompt string) {
	single, plural := "", ""
	switch action {
	case noteRangeDone:
		title, single, plural = " DONE CONFIRMATION ", "Mark this note done?", "Mark these %d notes done?"
	case noteRangeActive:
		title, single, plural = " ACTIVE CONFIRMATION ", "Move this note back to active?", "Move these %d notes back to active?"
	case noteRangeStartToday:
		title, single, plural = " TODAY CONFIRMATION ", "Start this note today?", "Start these %d notes today?"
	case noteRangeBackToPending:
		title, single, plural = " PENDING CONFIRMATION ", "Mark this note pending?", "Mark these %d notes pending?"
	}
	if len(targets) == 1 {
		return title, fmt.Sprintf("%s\n\n\"%s\"", single, targets[0].Summary)
	}
	return title, fmt.Sprintf(plural, len(targets))
}

func (m Model) beginNoteRangeConfirm(action noteRangeAction, targets []*model.Note) (tea.Model, tea.Cmd) {
	if len(targets) == 0 {
		return m, nil
	}
	m.clearPendingConfirms()
	m.notesUI.deleteTargetNotes = targets
	m.notesUI.pendingRangeAction = action
	m.confirmTitle, m.confirmPrompt = noteRangePrompt(action, targets)
	m.pushMode(ViewDeleteConfirm)
	return m, nil
}

func (m Model) applyNoteRangeAction(action noteRangeAction, targets []*model.Note) (tea.Model, tea.Cmd) {
	m.clearNoteRange()
	now := time.Now()
	switch action {
	case noteRangeDone, noteRangeActive:
		for _, note := range targets {
			m.toggleNoteDone(note)
		}
	case noteRangeStartToday, noteRangeBackToPending:
		for _, note := range targets {
			note.ToggleWorkingOn(now)
		}
		m.postMessage(messageSourceNotes, messageSuccess, workOnNotice(action, len(targets)))
		m.selectNoteByID(targets[0].ID)
		m.updateScrollOffset()
	}
	return m, m.saveNotesCmd(targets...)
}

func workOnNotice(action noteRangeAction, count int) string {
	switch {
	case action == noteRangeStartToday && count == 1:
		return "started today"
	case action == noteRangeStartToday:
		return fmt.Sprintf("started %d notes today", count)
	case count == 1:
		return "marked pending"
	}
	return fmt.Sprintf("marked %d notes pending", count)
}

func noteCopyText(note *model.Note) string {
	return strings.TrimSpace(fmt.Sprintf("%s\n\n%s", note.Summary, note.Body))
}

func (m Model) copyNoteRange(tea.KeyMsg) (tea.Model, tea.Cmd) {
	notes := m.rangeNotes()
	texts := make([]string, len(notes))
	for index, note := range notes {
		texts[index] = noteCopyText(note)
	}
	m.copyText(strings.Join(texts, "\n\n"))
	return m, nil
}
