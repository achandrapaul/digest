package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/store"

	tea "github.com/charmbracelet/bubbletea"
)

func workOnNotes(now time.Time) []*model.Note {
	return []*model.Note{
		{Summary: "old task", Status: model.StatusActive, Source: model.SourceManual, Created: now.AddDate(0, 0, -3), Updated: now.AddDate(0, 0, -3)},
		{Summary: "fresh task", Status: model.StatusActive, Source: model.SourceManual, Created: now, Updated: now},
	}
}

func groupSummaries(notes []*model.Note) []string {
	var summaries []string
	for _, note := range notes {
		summaries = append(summaries, note.Summary)
	}
	return summaries
}

func pressWorkOn(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	if binding, found := m.resolveKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}); !found || binding.action != actionWorkOn {
		t.Fatalf("f should resolve to the work-on action, got %+v %v", binding, found)
	}
	next, cmd := m.toggleWorkOn(tea.KeyMsg{})
	return next.(Model), cmd
}

func lastMessageText(m Model) string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1].text
}

func TestFPullsAnEarlierNoteIntoTodayAndBack(t *testing.T) {
	now := time.Now()
	m := noteSaveModel(t, workOnNotes(now)...)
	selectSummary(t, &m, "old task")
	m, cmd := pressWorkOn(t, m)
	if groups := m.groupNotes(); !slices.Contains(groupSummaries(groups.today), "old task") || slices.Contains(groupSummaries(groups.carried), "old task") {
		t.Fatalf("today = %v carried = %v", groupSummaries(groups.today), groupSummaries(groups.carried))
	}
	if lastMessageText(m) != "started today" {
		t.Errorf("message = %q", lastMessageText(m))
	}
	if item, _ := m.selectedNavItem(); item.Note == nil || item.Note.Summary != "old task" {
		t.Errorf("selection should follow the note, got %+v", item)
	}
	messages := savedMessages(cmd)
	if len(messages) != 1 {
		t.Fatalf("f should only save, got %#v", messages)
	}
	if _, single := messages[0].(notesSavedMsg); !single {
		t.Fatalf("f should save and reread only that note, got %#v", messages[0])
	}
	m, _ = applyNoteMessages(t, m, cmd)
	onDisk, err := store.Load(noteBySummary(t, m, "old task").FilePath)
	if err != nil || !slices.Equal(onDisk.StartedWorkingOn, []string{now.Format(model.WorkDayLayout)}) {
		t.Fatalf("on disk = %v, %v", onDisk, err)
	}
	if !strings.Contains(stripANSI(m.View()), "Active items for today") {
		t.Errorf("dashboard should show the renamed section")
	}

	m, cmd = pressWorkOn(t, m)
	m, _ = applyNoteMessages(t, m, cmd)
	if groups := m.groupNotes(); !slices.Contains(groupSummaries(groups.carried), "old task") {
		t.Fatalf("second f should move it back, carried = %v", groupSummaries(groups.carried))
	}
	if lastMessageText(m) != "marked pending" {
		t.Errorf("message = %q", lastMessageText(m))
	}
	if onDisk, _ := store.Load(noteBySummary(t, m, "old task").FilePath); len(onDisk.StartedWorkingOn) != 0 {
		t.Errorf("dates left on disk: %v", onDisk.StartedWorkingOn)
	}
}

func TestOnlyTheLastStartedDateCountsForTheViewedDay(t *testing.T) {
	now := time.Now()
	notes := workOnNotes(now)
	notes[0].StartedWorkingOn = []string{now.AddDate(0, 0, -1).Format(model.WorkDayLayout), now.Format(model.WorkDayLayout)}
	m := noteSaveModel(t, notes...)
	if groups := m.groupNotes(); !slices.Contains(groupSummaries(groups.today), "old task") {
		t.Fatalf("today = %v", groupSummaries(groups.today))
	}
	m.currentDate = now.AddDate(0, 0, -1)
	if groups := m.groupNotes(); !slices.Contains(groupSummaries(groups.carried), "old task") {
		t.Fatalf("an earlier started day should not count, carried = %v", groupSummaries(groups.carried))
	}
	if age, _ := m.renderNoteTagCells(notes[0]); !strings.Contains(stripANSI(age), "2d ago") {
		t.Errorf("age should still count from created, got %q", stripANSI(age))
	}
}

func TestFOnlyWorksOnActiveEarlierNotesWhileViewingToday(t *testing.T) {
	now := time.Now()
	notes := append(workOnNotes(now), &model.Note{Summary: "closed", Status: model.StatusDone, Source: model.SourceManual, Created: now.AddDate(0, 0, -4), Updated: now})
	m := noteSaveModel(t, notes...)
	for _, summary := range []string{"fresh task", "closed"} {
		selectSummary(t, &m, summary)
		if next, cmd := pressKey(t, m, "f"); cmd != nil || len(noteBySummary(t, next, summary).StartedWorkingOn) != 0 {
			t.Errorf("f should do nothing on %q", summary)
		}
	}
	selectSummary(t, &m, "old task")
	if !slices.ContainsFunc(m.selectedRowHints(), func(hint keyHint) bool { return hint.key == "f" && hint.label == "start today" }) {
		t.Errorf("row hints should offer f on an earlier note: %v", m.selectedRowHints())
	}
	m.currentDate = now.AddDate(0, 0, -1)
	m.contentVersion++
	selectSummary(t, &m, "old task")
	if _, cmd := pressKey(t, m, "f"); cmd != nil {
		t.Error("f should do nothing while viewing another day")
	}
}

func TestPreviewListsTheWorkedOnDates(t *testing.T) {
	created := time.Date(2026, 10, 3, 9, 0, 0, 0, time.Local)
	note := &model.Note{Summary: "s", Created: created, StartedWorkingOn: []string{"10-10-2026", "12-10-2026"}}
	if got := workedOnLine(note); got != "You have tried to work on this on 3 dates: Sat 03 Oct (created), Sat 10 Oct, Mon 12 Oct" {
		t.Errorf("line = %q", got)
	}
	if got := workedOnLine(&model.Note{Created: created}); got != "" {
		t.Errorf("a note never picked up again should show no line, got %q", got)
	}
	now := time.Now()
	notes := workOnNotes(now)
	notes[0].StartedWorkingOn = []string{now.Format(model.WorkDayLayout)}
	m := noteSaveModel(t, notes...)
	selectSummary(t, &m, "old task")
	m, _ = pressKey(t, m, "tab")
	lines := strings.Split(stripANSI(m.View()), "\n")
	headerRow := slices.IndexFunc(lines, func(line string) bool { return strings.Contains(line, "PREVIEW NOTE") })
	if headerRow < 0 || headerRow+1 >= len(lines) || !strings.Contains(lines[headerRow+1], "You have tried to work on this on 2 dates") || strings.Contains(strings.Join(lines, "\n"), "#general") {
		t.Errorf("the dates should sit right below the preview header:\n%s", strings.Join(lines, "\n"))
	}
	if strings.Contains(stripANSI(m.previewViewport.View()), "You have tried") {
		t.Errorf("the dates should not scroll with the body:\n%s", stripANSI(m.previewViewport.View()))
	}
}

func TestDoneRecordsTheDoneDayOnceAndActiveKeepsIt(t *testing.T) {
	now := time.Now()
	m := noteSaveModel(t, workOnNotes(now)...)
	today := now.Format(model.WorkDayLayout)
	selectSummary(t, &m, "old task")
	m, cmd := pressKey(t, m, " ")
	m, _ = applyNoteMessages(t, m, cmd)
	note := noteBySummary(t, m, "old task")
	if !slices.Equal(note.StartedWorkingOn, []string{today}) {
		t.Fatalf("done should record today, got %v", note.StartedWorkingOn)
	}
	selectSummary(t, &m, "old task")
	m, cmd = pressKey(t, m, " ")
	m, _ = applyNoteMessages(t, m, cmd)
	selectSummary(t, &m, "old task")
	m, cmd = pressKey(t, m, " ")
	m, _ = applyNoteMessages(t, m, cmd)
	if onDisk, _ := store.Load(noteBySummary(t, m, "old task").FilePath); !slices.Equal(onDisk.StartedWorkingOn, []string{today}) {
		t.Errorf("active then done again should keep one date, got %v", onDisk.StartedWorkingOn)
	}
}

func TestFThenSpaceBeforeTheSaveReturnsKeepsBoth(t *testing.T) {
	now := time.Now()
	m := noteSaveModel(t, workOnNotes(now)...)
	selectSummary(t, &m, "old task")
	m, workCmd := pressWorkOn(t, m)
	m, doneCmd := pressKey(t, m, " ")
	m, _ = applyNoteMessages(t, m, workCmd)
	m, _ = applyNoteMessages(t, m, doneCmd)
	note := noteBySummary(t, m, "old task")
	onDisk, err := store.Load(note.FilePath)
	if err != nil || onDisk.Status != model.StatusDone || !slices.Equal(onDisk.StartedWorkingOn, []string{now.Format(model.WorkDayLayout)}) {
		t.Fatalf("on disk = %+v, %v", onDisk, err)
	}
}

func TestUnreadableStartedDatesShowAsAHeaderError(t *testing.T) {
	now := time.Now()
	notes := workOnNotes(now)
	notes[0].StartedWorkingOn = []string{"2026-10-12"}
	m := noteSaveModel(t, notes...)
	m, _ = applyNoteMessages(t, m, m.Init())
	if text := lastMessageText(m); text != `old task: started_working_on "2026-10-12" is not DD-MM-YYYY` {
		t.Errorf("message = %q", text)
	}
	if groups := m.groupNotes(); !slices.Contains(groupSummaries(groups.carried), "old task") {
		t.Errorf("an unreadable date should be ignored, carried = %v", groupSummaries(groups.carried))
	}
	m.messages = nil
	next, _ := m.Update(notesChangedMsg{notes: []model.Note{*noteBySummary(t, m, "old task")}})
	if text := lastMessageText(next.(Model)); !strings.Contains(text, "is not DD-MM-YYYY") {
		t.Errorf("a background change should report it too, got %q", text)
	}
}
