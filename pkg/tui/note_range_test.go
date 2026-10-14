package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"

	tea "github.com/charmbracelet/bubbletea"
)

func rangeTestModel(t *testing.T) Model {
	t.Helper()
	now := time.Now()
	m := noteSaveModel(t,
		&model.Note{Summary: "carried task", Status: model.StatusActive, Source: model.SourceManual, Created: now.AddDate(0, 0, -3), Updated: now.AddDate(0, 0, -3)},
		&model.Note{Summary: "fresh task", Status: model.StatusActive, Source: model.SourceManual, Created: now, Updated: now},
		&model.Note{Summary: "closed task", Status: model.StatusDone, Source: model.SourceManual, Created: now, Updated: now},
	)
	selectSummary(t, &m, "carried task")
	return m
}

func rangeSummaries(m Model) []string {
	var summaries []string
	for _, note := range m.rangeNotes() {
		summaries = append(summaries, note.Summary)
	}
	slices.Sort(summaries)
	return summaries
}

func extendRange(t *testing.T, m Model, times int) Model {
	t.Helper()
	for range times {
		m, _ = pressKey(t, m, "J")
	}
	return m
}

func TestShiftMovesBuildAnAnchoredRangeAcrossSections(t *testing.T) {
	m := extendRange(t, rangeTestModel(t), 2)
	if got := rangeSummaries(m); !slices.Equal(got, []string{"carried task", "closed task", "fresh task"}) {
		t.Fatalf("range = %v", got)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m = next.(Model)
	if got := rangeSummaries(m); !slices.Equal(got, []string{"carried task", "fresh task"}) {
		t.Fatalf("shift+up should shrink the range, got %v", got)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	m = next.(Model)
	if len(m.rangeNotes()) != 3 {
		t.Fatalf("shift+down should extend the range, got %v", rangeSummaries(m))
	}
	if pill := stripANSI(renderHintPill(hintsFromBindings(m.dashboardBindings()))); !strings.Contains(pill, "(shift)multiple selection") {
		t.Errorf("hints should show the selection key: %q", pill)
	}
}

func TestPlainNavigationAndDayChangesCancelTheRange(t *testing.T) {
	for _, key := range []string{"j", "k", "p", "t", "esc"} {
		m := extendRange(t, rangeTestModel(t), 2)
		m, _ = pressKey(t, m, key)
		if m.notesUI.rangeActive {
			t.Errorf("%q should cancel the selection", key)
		}
	}
}

func TestSpaceOnARangeMarksTheActiveNotesDoneFirst(t *testing.T) {
	m := extendRange(t, rangeTestModel(t), 2)
	m, _ = pressKey(t, m, " ")
	if m.mode != ViewDeleteConfirm || !strings.Contains(m.confirmPrompt, "Mark these 2 notes done?") {
		t.Fatalf("mode = %v prompt = %q", m.mode, m.confirmPrompt)
	}
	m, cmd := pressKey(t, m, "y")
	m, _ = applyNoteMessages(t, m, cmd)
	for _, summary := range []string{"carried task", "fresh task", "closed task"} {
		if noteBySummary(t, m, summary).Status != model.StatusDone {
			t.Errorf("%s should be done", summary)
		}
	}
	if m.notesUI.rangeActive {
		t.Error("a confirmed action should clear the selection")
	}
}

func TestSpaceOnAnAllDoneRangeMovesThemBackToActive(t *testing.T) {
	m := rangeTestModel(t)
	closedAt := noteBySummary(t, m, "closed task").Updated
	for offset, summary := range []string{"fresh task", "carried task"} {
		note := noteBySummary(t, m, summary)
		note.Status, note.Updated = model.StatusDone, closedAt.Add(time.Duration(offset+1)*time.Second)
	}
	m.contentVersion++
	selectSummary(t, &m, "carried task")
	m = extendRange(t, m, 2)
	m, _ = pressKey(t, m, " ")
	if !strings.Contains(m.confirmPrompt, "Move these 3 notes back to active?") {
		t.Fatalf("prompt = %q", m.confirmPrompt)
	}
	m, cmd := pressKey(t, m, "y")
	m, _ = applyNoteMessages(t, m, cmd)
	for _, summary := range []string{"carried task", "fresh task", "closed task"} {
		if noteBySummary(t, m, summary).Status != model.StatusActive {
			t.Errorf("%s should be active", summary)
		}
	}
}

func TestCancellingTheConfirmKeepsTheRangeAndTheNotes(t *testing.T) {
	m := extendRange(t, rangeTestModel(t), 2)
	m, _ = pressKey(t, m, " ")
	m, _ = pressKey(t, m, "n")
	if !m.notesUI.rangeActive || len(m.rangeNotes()) != 3 {
		t.Fatalf("cancel should keep the selection, got %v", rangeSummaries(m))
	}
	if noteBySummary(t, m, "carried task").Status != model.StatusActive {
		t.Error("cancel should not change any note")
	}
}

func TestDeleteOnARangeArchivesEveryNoteAfterConfirm(t *testing.T) {
	m := extendRange(t, rangeTestModel(t), 2)
	m, _ = pressKey(t, m, "d")
	if m.mode != ViewDeleteConfirm || !strings.Contains(stripANSI(m.View()), "these 3 selected notes") {
		t.Fatalf("delete should confirm the whole range, mode = %v", m.mode)
	}
	m, cmd := pressKey(t, m, "y")
	m, _ = applyNoteMessages(t, m, cmd)
	if got := m.groupNotes(); len(got.carried)+len(got.today)+len(got.todayDone) != 0 {
		t.Errorf("every selected note should be archived, left %v %v %v", groupSummaries(got.carried), groupSummaries(got.today), groupSummaries(got.todayDone))
	}
	if m.notesUI.rangeActive {
		t.Error("delete should clear the selection")
	}
}

func TestFOnARangeMovesOnlyEligibleNotesToToday(t *testing.T) {
	m := extendRange(t, rangeTestModel(t), 2)
	m, _ = pressKey(t, m, "f")
	if !strings.Contains(m.confirmPrompt, "Start this note today?") || !strings.Contains(m.confirmPrompt, "carried task") {
		t.Fatalf("prompt = %q", m.confirmPrompt)
	}
	m, cmd := pressKey(t, m, "y")
	m, _ = applyNoteMessages(t, m, cmd)
	if !noteBySummary(t, m, "carried task").StartedOn(time.Now()) {
		t.Error("the carried note should be worked on today")
	}
	if m.notesUI.rangeActive {
		t.Error("f should clear the selection")
	}
}

func TestCtrlYCopiesTheRangeAndKeepsIt(t *testing.T) {
	var copied []string
	previous := copyToClipboard
	copyToClipboard = func(text string) error { copied = append(copied, text); return nil }
	t.Cleanup(func() { copyToClipboard = previous })
	m := extendRange(t, rangeTestModel(t), 1)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	m = next.(Model)
	if len(copied) != 1 || strings.Count(copied[0], "task") != 2 || !strings.Contains(copied[0], "\n\n") {
		t.Fatalf("copied = %q", copied)
	}
	if !m.notesUI.rangeActive {
		t.Error("copy should keep the selection")
	}
}

func TestSingleRowSpaceAndFActWithoutAsking(t *testing.T) {
	m := rangeTestModel(t)
	m, _ = pressKey(t, m, " ")
	if m.mode != ViewDashboard || noteBySummary(t, m, "carried task").Status != model.StatusDone {
		t.Fatalf("space on one row should mark it done straight away, mode = %v", m.mode)
	}
	m = rangeTestModel(t)
	m, _ = pressKey(t, m, "f")
	if m.mode != ViewDashboard || !noteBySummary(t, m, "carried task").StartedOn(time.Now()) {
		t.Fatalf("f on one row should work on it today straight away, mode = %v", m.mode)
	}
}
