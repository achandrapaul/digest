package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestClampingKeepsPopupSelectionsInRange(t *testing.T) {
	m := mixedGitDetailsModel(t)
	m.git.gitPopupCursor.selected = 99
	m.clampScreenSelection()
	if m.git.gitPopupCursor.selected != 2 {
		t.Fatalf("git selection = %d", m.git.gitPopupCursor.selected)
	}
	archive := openArchiveModel(t)
	archive.notesUI.archivedCursor.selected = 99
	archive.clampScreenSelection()
	if archive.notesUI.archivedCursor.selected != 2 {
		t.Fatalf("archive selection = %d", archive.notesUI.archivedCursor.selected)
	}
	archive.notesUI.archivedCursor.selected = -2
	archive.clampScreenSelection()
	if archive.notesUI.archivedCursor.selected != 0 {
		t.Fatalf("archive selection = %d", archive.notesUI.archivedCursor.selected)
	}
}

func TestRejectCommentBoxTakesTypedText(t *testing.T) {
	m := rejectCommentModel(t)
	m = press(t, m, runes("x"))
	if m.review.rejectInput.Value() != "x" {
		t.Fatalf("value = %q", m.review.rejectInput.Value())
	}
}

func TestHalfPagesUseAMinimumStepOnTinyTerminals(t *testing.T) {
	m := manyTodayNotesModel(t, 30)
	m.height = 3
	next, _ := m.dashboardHalfPageDown(tea.KeyMsg{})
	if next.(Model).selected != 2 {
		t.Fatalf("selected = %d", next.(Model).selected)
	}
	m.selected = 10
	next, _ = m.dashboardHalfPageUp(tea.KeyMsg{})
	if next.(Model).selected != 8 {
		t.Fatalf("selected = %d", next.(Model).selected)
	}
}

func TestRowKeysWithNothingSelectedDoNothing(t *testing.T) {
	m := syncTestModel(t)
	m.notes = nil
	m.contentVersion++
	m.selected = 99
	for name, action := range map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd){
		"preview": m.openSelectedPreview,
		"open":    m.openSelectedItem,
		"delete":  m.deleteSelectedItem,
		"copy":    m.copyPreviewItem,
		"stop":    m.previewStop,
		"enter":   m.previewEnter,
		"work on": m.toggleWorkOn,
	} {
		if next, cmd := action(tea.KeyMsg{}); cmd != nil || next.(Model).mode != m.mode {
			t.Fatalf("%s should do nothing", name)
		}
	}
}

func TestOpeningARepoRowShowsItsDetailsAndCopyTakesTheName(t *testing.T) {
	m := gitStripTestModel(t)
	selectNavKind(t, &m, KindGitRepo)
	next, _ := m.openSelectedItem(tea.KeyMsg{})
	if next.(Model).mode != ViewGitDetails {
		t.Fatalf("enter on a repo row opens its details, mode %v", next.(Model).mode)
	}
	if next, _ := m.previewEnter(tea.KeyMsg{}); next.(Model).mode != m.mode {
		t.Fatal("enter in a repo preview does nothing")
	}
	copied := captureClipboard(t)
	m.copyPreviewItem(tea.KeyMsg{})
	if len(*copied) != 1 || (*copied)[0] != m.allNavItems()[m.selected].GitRepo.Name {
		t.Fatalf("copied = %v", *copied)
	}
}

func TestRunningAJobThatFailsToStartShowsTheError(t *testing.T) {
	tempDigestRoot(t)
	m, _, _ := jobTestModel(t)
	startJobProcess = func(launch jobLaunch) error {
		if launch.dryRun {
			return nil
		}
		return errors.New("no shell")
	}
	m.jobs.jobToExecute = "nightly"
	m.modeStack = []ViewMode{ViewDashboard}
	next, cmd := m.confirmDelete(tea.KeyMsg{})
	if settled := settleBackground(t, next.(Model), cmd); latestMessageText(settled) != "no shell" {
		t.Fatalf("message = %q", latestMessageText(settled))
	}
}

func TestArchivingFromTheDashboardSavesTheNote(t *testing.T) {
	m := noteSaveModel(t, dashboardNotes(time.Now())...)
	selectNavKind(t, &m, KindCarriedNote)
	note := m.allNavItems()[m.selected].Note
	m.notesUI.deleteTargetNotes = []*model.Note{note}
	m.modeStack = []ViewMode{ViewDashboard}
	if _, cmd := m.confirmDelete(tea.KeyMsg{}); cmd == nil || note.Status != model.StatusArchived {
		t.Fatalf("status = %v", note.Status)
	}
}

func TestSearchKeysWithoutResultsDoNothing(t *testing.T) {
	m := typeQuery(t, searchTestModel(t), "zzzz")
	for _, binding := range m.searchBindings() {
		for _, key := range binding.binding.Keys() {
			if key == "tab" || key == "enter" {
				t.Fatalf("%s is bound without results", key)
			}
		}
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyTab}, {Type: tea.KeyEnter}} {
		if next := press(t, m, key); next.mode != ViewSearch || next.searchPreviewing {
			t.Fatalf("%s left the search, mode %v", key, next.mode)
		}
	}
}

func TestDismissingAnErrorRefreshesTheArchive(t *testing.T) {
	m := openArchiveModel(t)
	m.modeStack, m.mode = []ViewMode{ViewArchived}, ViewError
	if next, _ := m.dismissError(tea.KeyMsg{}); next.(Model).mode != ViewArchived {
		t.Fatalf("mode = %v", next.(Model).mode)
	}
}
