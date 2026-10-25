package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/model"
)

func noteRowLine(t *testing.T, view, summary string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, " "+summary+" ") || strings.HasSuffix(strings.TrimRight(strings.Trim(strings.TrimSpace(line), "│"), " "), summary) {
			return line
		}
	}
	t.Fatalf("no row %q in\n%s", summary, view)
	return ""
}

func assertNoteBox(t *testing.T, view, summary, box string) {
	t.Helper()
	if line := noteRowLine(t, view, summary); !strings.Contains(line, box+" "+summary) {
		t.Errorf("%q should show %q: %q", summary, box, line)
	}
}

func TestNoteRowsTickTheSelectionAndLeaveOthersEmptyOrDone(t *testing.T) {
	m := appStateModel(t, &appstate.State{}, tagGroupNotes(time.Now())...)
	selectSummary(t, &m, "active manual one")
	view := stripANSI(m.View())
	assertNoteBox(t, view, "active manual one", tickedBoxIcon)
	assertNoteBox(t, view, "active my pr", emptyBoxIcon)
	assertNoteBox(t, view, "closed manual", "✔")

	next, cmd := pressKey(t, m, "J")
	m = applyAllMessages(next, cmd)
	next, cmd = pressKey(t, m, "J")
	m = applyAllMessages(next, cmd)
	view = stripANSI(m.View())
	for _, summary := range []string{"active manual one", "active my pr", "active manual two"} {
		assertNoteBox(t, view, summary, tickedBoxIcon)
	}
	assertNoteBox(t, view, "carried manual", emptyBoxIcon)
	assertNoteBox(t, view, "closed my pr", "✔")
}

func TestASelectedDoneNoteKeepsItsTick(t *testing.T) {
	m := appStateModel(t, &appstate.State{}, tagGroupNotes(time.Now())...)
	selectSummary(t, &m, "closed manual")
	assertNoteBox(t, stripANSI(m.View()), "closed manual", "✔")
}

func TestArchivedPendingNotesUseTheDashboardEmptyBox(t *testing.T) {
	if pendingBox := archivedNoteBox(&model.Note{Status: model.StatusActive}); !strings.Contains(pendingBox, emptyBoxIcon) {
		t.Errorf("archived pending box = %q, want %q", pendingBox, emptyBoxIcon)
	}
	if doneBox := archivedNoteBox(&model.Note{Status: model.StatusDone}); !strings.Contains(doneBox, "✔") {
		t.Errorf("archived done box = %q", doneBox)
	}
}
