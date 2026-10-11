package tui

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/brag"
	tea "github.com/charmbracelet/bubbletea"
)

func week40() brag.Week {
	return brag.WeekOf(time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local))
}

const week40Title = "Week 40 · 28 Sep – 04 Oct"

func savedWeek40(t *testing.T, m Model) {
	t.Helper()
	if err := (&brag.Brag{Period: week40(), Created: time.Now(), Facts: "- collected", Summary: "- Did things"}).Save(m.cfg.BragDir()); err != nil {
		t.Fatal(err)
	}
}

func markBragRunning(t *testing.T, m Model, id string) {
	t.Helper()
	writeBragState(t, m, id, "brag.pid", strconv.Itoa(os.Getpid()))
}

func openBragView(t *testing.T) Model {
	t.Helper()
	m, _ := bragTestModel(t, wednesday())
	savedWeek40(t, m)
	m = press(t, m, runes("b"))
	m = selectBragRow(t, m, week40Title)
	return press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
}

func TestBragListJAndKMoveWithinTheRows(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	savedWeek40(t, m)
	m = press(t, m, runes("b"))
	m = press(t, m, runes("k"))
	if m.brag.bragCursor.selected != 0 {
		t.Fatalf("k at the top should stay, got %d", m.brag.bragCursor.selected)
	}
	m = press(t, m, runes("j"))
	m = press(t, m, runes("j"))
	m = press(t, m, runes("k"))
	if m.brag.bragCursor.selected != 1 {
		t.Fatalf("selected = %d, want 1", m.brag.bragCursor.selected)
	}
	last := len(m.bragRows()) - 1
	m.brag.bragCursor.selected = last
	if m = press(t, m, runes("j")); m.brag.bragCursor.selected != last {
		t.Fatalf("j at the bottom should stay, got %d", m.brag.bragCursor.selected)
	}
}

func TestOpeningTheBragListClampsAStaleSelection(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	savedWeek40(t, m)
	m.brag.bragCursor.selected = 999
	m = press(t, m, runes("b"))
	if m.brag.bragCursor.selected != len(m.bragRows())-1 {
		t.Fatalf("selected = %d of %d", m.brag.bragCursor.selected, len(m.bragRows()))
	}
	m.brag.bragCursor.selected = 999
	if next, _ := m.bragListEnter(tea.KeyMsg{}); next.(Model).mode != ViewBragList {
		t.Fatal("enter on no row should do nothing")
	}
}

func TestEnterOnARunningBragSaysItIsAlreadyBragging(t *testing.T) {
	m, started := bragTestModel(t, wednesday())
	markBragRunning(t, m, week40().ID())
	m.refreshBragRuns()
	m = press(t, m, runes("b"))
	m = selectBragRow(t, m, week40Title)
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.brag.bragNotice != "Already bragging about "+week40().Label()+" — see Jobs" || len(*started) != 0 {
		t.Fatalf("notice %q started %v", m.brag.bragNotice, *started)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Bragging...") || !strings.Contains(view, "Already bragging about") {
		t.Fatalf("list should show the running row and notice:\n%s", view)
	}
	if !m.anyBragRunning() {
		t.Fatal("a running brag should be reported")
	}
}

func TestBragListLabelsSavedBragsAndPadsShortLists(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	savedWeek40(t, m)
	m.height = 200
	m = press(t, m, runes("b"))
	if view := stripANSI(m.View()); !strings.Contains(view, "View your brag") {
		t.Fatalf("list:\n%s", view)
	}
}

func TestBragViewWhileRunningShowsThePulseAndNotice(t *testing.T) {
	m := openBragView(t)
	markBragRunning(t, m, week40().ID())
	m.refreshBragRuns()
	m.brag.bragNotice = "Summary updated"
	view := stripANSI(m.View())
	if !strings.Contains(view, "BRAG: "+strings.ToUpper(week40().Label())) || !strings.Contains(view, "Bragging...") || !strings.Contains(view, "Summary updated") {
		t.Fatalf("view:\n%s", view)
	}
	if m = press(t, m, runes("b")); m.brag.bragNotice != "Already bragging — see Jobs" || m.mode != ViewBragView {
		t.Fatalf("b while running: notice %q mode %v", m.brag.bragNotice, m.mode)
	}
}

func TestBragEditorScreenShowsTheEditorAndNotice(t *testing.T) {
	m := openBragView(t)
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.brag.bragNotice = "Facts heading missing"
	view := stripANSI(m.View())
	for _, want := range []string{"EDIT BRAG: " + strings.ToUpper(week40().Label()), "## Facts", "Facts heading missing"} {
		if !strings.Contains(view, want) {
			t.Fatalf("editor screen missing %q:\n%s", want, view)
		}
	}
}

func TestBragConfirmTextsForFirstAndRepeatBrags(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	m.brag.bragPeriod = week40()
	m.mode = ViewBragConfirm
	if view := stripANSI(m.View()); !strings.Contains(view, "Brag about "+week40().Label()+"?") || !strings.Contains(view, "Facts are collected and Claude writes the summary in the background.") {
		t.Fatalf("first confirm:\n%s", view)
	}
	m.brag.bragRegenerate = true
	if view := stripANSI(m.View()); !strings.Contains(view, "Brag again about "+week40().Label()+"?") || !strings.Contains(view, "Claude rewrites only the summary from the saved facts.") {
		t.Fatalf("repeat confirm:\n%s", view)
	}
	m.modeStack = []ViewMode{ViewBragList}
	if next, _ := m.cancelBrag(tea.KeyMsg{}); next.(Model).mode != ViewBragList {
		t.Fatal("cancel should return to where the confirm came from")
	}
}

func TestConfirmBragReportsARunAlreadyGoing(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	startBragRun = func(string, brag.Period, bool) error { return brag.ErrBragRunning }
	m.brag.bragPeriod, m.modeStack = week40(), []ViewMode{ViewBragList}
	next, cmd := m.confirmBrag(tea.KeyMsg{})
	if got := settleBackground(t, next.(Model), cmd); got.brag.bragNotice != "Already bragging — see Jobs" || got.mode != ViewBragList {
		t.Fatalf("notice %q mode %v", got.brag.bragNotice, got.mode)
	}
}

func TestBragViewReloadsWhenTheSummaryChanges(t *testing.T) {
	m := openBragView(t)
	m.reloadBragViewIfChanged()
	if m.brag.bragNotice != "" {
		t.Fatalf("an unchanged brag should not reload, notice %q", m.brag.bragNotice)
	}
	updated := *m.brag.bragEntry
	updated.Summary = "- Did more things"
	updated.SummarizedAt = time.Now().Add(time.Minute)
	if err := updated.Save(m.cfg.BragDir()); err != nil {
		t.Fatal(err)
	}
	next, _ := m.handleReviewPoll(reviewPollSnapshot{})
	if got := next.(Model); got.brag.bragNotice != "Summary updated" || !strings.Contains(stripANSI(got.View()), "Did more things") {
		t.Fatalf("notice %q", got.brag.bragNotice)
	}
	m.brag.bragEntry = nil
	m.reloadBragViewIfChanged()
}

func TestShowingAMissingBragKeepsTheListWithTheError(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	m.mode = ViewBragList
	next, _ := m.showBragView(week40())
	if got := next.(Model); got.mode != ViewBragList || got.brag.bragNotice == "" {
		t.Fatalf("mode %v notice %q", got.mode, got.brag.bragNotice)
	}
	if next, cmd := m.editBrag(tea.KeyMsg{}); cmd != nil || next.(Model).mode != ViewBragList {
		t.Fatal("editing without a loaded brag should do nothing")
	}
}

func TestWeekBraggedReadsDiskWithoutAMemo(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	savedWeek40(t, m)
	m.brag.bragSaved = nil
	if !m.weekBragged(week40()) {
		t.Fatal("the saved week should count as bragged")
	}
}

func TestBragRunPreviewWithoutALog(t *testing.T) {
	preview := bragRunPreview(t.TempDir(), brag.Run{Meta: brag.RunMeta{ID: "2026-W40"}, Status: brag.RunRunning})
	if preview.log != "(no log output yet)" || !strings.Contains(preview.heading, "(RUNNING)") {
		t.Fatalf("preview = %+v", preview)
	}
}

func TestStoppingABragRunReportsFailures(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	markBragRunning(t, m, week40().ID())
	previous := stopBragRun
	stopBragRun = func(string, string) error { return errors.New("kill failed") }
	t.Cleanup(func() { stopBragRun = previous })
	next, _ := m.stopOrDismissBragRun(&brag.Run{Meta: brag.RunMeta{ID: week40().ID()}})
	if got := next.(Model); got.mode == ViewPreview {
		t.Fatal("a failed stop should not open a preview")
	}
}

func TestDismissingTheLastBragRunLeavesThePreview(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	m.notes = nil
	writeBragState(t, m, week40().ID(), "brag.exit", "1")
	writeBragState(t, m, week40().ID(), "meta.json", `{"id":"`+week40().ID()+`"}`)
	m.refreshBragRuns()
	m.mode = ViewPreview
	selectNavKind(t, &m, KindBragRun)
	run := *m.allNavItems()[m.selected].BragRun
	next, _ := m.stopOrDismissBragRun(&run)
	if got := next.(Model); got.mode != ViewDashboard {
		t.Fatalf("mode = %v", got.mode)
	}
}

func TestDismissingABragRunKeepsThePreviewOnTheNextRow(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	for _, id := range []string{"2026-W39", "2026-W40"} {
		writeBragState(t, m, id, "brag.exit", "1")
		writeBragState(t, m, id, "meta.json", `{"id":"`+id+`"}`)
	}
	m.refreshBragRuns()
	m.mode = ViewPreview
	selectNavKind(t, &m, KindBragRun)
	run := *m.allNavItems()[m.selected].BragRun
	next, _ := m.stopOrDismissBragRun(&run)
	if got := next.(Model); got.mode != ViewPreview {
		t.Fatalf("mode = %v", got.mode)
	}
}
