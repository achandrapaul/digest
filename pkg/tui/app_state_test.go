package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/store"

	tea "github.com/charmbracelet/bubbletea"
)

var everyDay = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func appStateModel(t *testing.T, state *appstate.State, notes ...*model.Note) Model {
	t.Helper()
	showGit := false
	cfg := &config.Config{DigestRoot: t.TempDir(), GreenOnly: true, ShowGit: &showGit, WorkDays: everyDay}
	noteStore := store.New(cfg.NotesDir())
	for _, note := range notes {
		if err := noteStore.Save(note); err != nil {
			t.Fatal(err)
		}
	}
	if state != nil {
		if err := appstate.Save(cfg.Root(), *state); err != nil {
			t.Fatal(err)
		}
	}
	m := NewModel(cfg, nil)
	m.width, m.height = 120, 50
	return m
}

func applyAllMessages(m Model, cmd tea.Cmd) Model {
	for _, msg := range collectMsgs(cmd) {
		switch msg.(type) {
		case notesSavedMsg, appStateSavedMsg, loadNotesMsg:
			next, followUp := m.Update(msg)
			m = applyAllMessages(next.(Model), followUp)
		}
	}
	return m
}

func markDone(t *testing.T, m Model, summary string) Model {
	t.Helper()
	selectSummary(t, &m, summary)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	return applyAllMessages(next.(Model), cmd)
}

func twoActiveNotes(now time.Time) []*model.Note {
	return []*model.Note{
		{Summary: "first", Status: model.StatusActive, Source: model.SourceManual, Created: now.Add(-time.Hour), Updated: now.Add(-time.Hour)},
		{Summary: "second", Status: model.StatusActive, Source: model.SourceManual, Created: now.Add(-time.Minute), Updated: now.Add(-time.Minute)},
	}
}

func TestFirstDoneOfTheDayWritesTheStateOnce(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	m := appStateModel(t, &appstate.State{FirstNoteCreated: now.AddDate(-1, 0, 0), Streak: 2, LastDoneDay: yesterday}, twoActiveNotes(now)...)
	m = markDone(t, m, "first")
	saved, found, err := appstate.Load(m.cfg.Root())
	if err != nil || !found || saved.Streak != 3 || saved.LastDoneDay != now.Format("2006-01-02") {
		t.Fatalf("state after the first done = %+v, %v, %v", saved, found, err)
	}
	if header := stripANSI(headerSection{}.Render(m)); !strings.Contains(header, "3-day streak") {
		t.Errorf("header should show the stored streak: %q", header)
	}
	if err := os.Remove(appstate.Path(m.cfg.Root())); err != nil {
		t.Fatal(err)
	}
	m = markDone(t, m, "second")
	if _, found, _ := appstate.Load(m.cfg.Root()); found {
		t.Error("a second done the same day should not write the state file")
	}
	if header := stripANSI(headerSection{}.Render(m)); !strings.Contains(header, "3-day streak") {
		t.Errorf("streak should stay at 3: %q", header)
	}
}

func TestHeaderStreakComesFromTheStateFile(t *testing.T) {
	now := time.Now()
	m := appStateModel(t, &appstate.State{FirstNoteCreated: now.AddDate(-1, 0, 0), Streak: 7, LastDoneDay: now.Format("2006-01-02")}, twoActiveNotes(now)...)
	if header := stripANSI(headerSection{}.Render(m)); !strings.Contains(header, "7-day streak") {
		t.Errorf("header should show the stored streak before notes load: %q", header)
	}
}

func assertStateModalQuits(t *testing.T, m Model, advice string) {
	t.Helper()
	m.requireAppState()
	if m.mode != ViewError || !strings.Contains(plainText(m.View()), advice) {
		t.Fatalf("mode %v, the state modal should advise %q:\n%s", m.mode, advice, plainText(m.View()))
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyEnter}} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%s should quit", key)
		}
		if _, quits := cmd().(tea.QuitMsg); !quits {
			t.Errorf("%s should quit the app", key)
		}
	}
}

func TestMissingStateWithNotesAdvisesMigrateThenQuits(t *testing.T) {
	m := appStateModel(t, nil, dashboardNotes(time.Now())...)
	m = applyAllMessages(m, m.Init())
	if _, found, _ := appstate.Load(m.cfg.Root()); found {
		t.Fatal("the TUI should never rebuild the state file")
	}
	assertStateModalQuits(t, m, "digest migrate")
}

func TestMissingStateWithoutNotesAdvisesBootstrapThenQuits(t *testing.T) {
	assertStateModalQuits(t, appStateModel(t, nil), "digest bootstrap")
}

func TestCorruptStateAdvisesMigrateThenQuits(t *testing.T) {
	m := appStateModel(t, nil, dashboardNotes(time.Now())...)
	if err := os.WriteFile(appstate.Path(m.cfg.Root()), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = applyAllMessages(m, m.Init())
	assertStateModalQuits(t, m, "digest migrate")
	if data, _ := os.ReadFile(appstate.Path(m.cfg.Root())); string(data) != "{not json" {
		t.Errorf("the TUI should not overwrite a corrupt state file: %s", data)
	}
}

func TestAReadableStateNeedsNoModal(t *testing.T) {
	m := appStateModel(t, &appstate.State{Streak: 1})
	m.requireAppState()
	if m.mode == ViewError {
		t.Errorf("a readable state file should not show the modal: %s", plainText(m.View()))
	}
}

func TestNewPRNotesNeverSetTheFirstNoteDate(t *testing.T) {
	m := appStateModel(t, &appstate.State{})
	old := time.Now().AddDate(-1, 0, 0)
	m = applyAllMessages(m, m.saveNotesCmd(
		&model.Note{Summary: "my pr", Status: model.StatusActive, Source: model.SourceMyPR, Created: old},
		&model.Note{Summary: "review", Status: model.StatusActive, Source: model.SourcePRReview, Created: old},
	))
	if saved, _, _ := appstate.Load(m.cfg.Root()); !saved.FirstNoteCreated.IsZero() || !m.appState.FirstNoteCreated.IsZero() {
		t.Errorf("PR notes should never set first_note_created: %v", saved.FirstNoteCreated)
	}
}

func TestSavingAnExistingManualNoteNeverSetsTheFirstNoteDate(t *testing.T) {
	now := time.Now()
	m := appStateModel(t, &appstate.State{}, twoActiveNotes(now)...)
	note := noteBySummary(t, m, "first")
	note.Summary = "first, edited"
	m = applyAllMessages(m, m.saveNotesCmd(note))
	if saved, _, _ := appstate.Load(m.cfg.Root()); !saved.FirstNoteCreated.IsZero() {
		t.Errorf("editing a note should never set first_note_created: %v", saved.FirstNoteCreated)
	}
	stored := now.AddDate(0, -1, 0)
	m = appStateModel(t, &appstate.State{FirstNoteCreated: stored}, twoActiveNotes(now)...)
	older := noteBySummary(t, m, "first")
	older.Created = stored.AddDate(-1, 0, 0)
	m = applyAllMessages(m, m.saveNotesCmd(older))
	if saved, _, _ := appstate.Load(m.cfg.Root()); !saved.FirstNoteCreated.Equal(stored) {
		t.Errorf("first_note_created should never move: %v, want %v", saved.FirstNoteCreated, stored)
	}
}

func TestTheFirstNewManualNoteSetsTheFirstNoteDate(t *testing.T) {
	m := appStateModel(t, &appstate.State{Streak: 2})
	created := time.Now().Add(-time.Minute)
	m = applyAllMessages(m, m.saveNotesCmd(&model.Note{Summary: "hello", Status: model.StatusActive, Source: model.SourceManual, Created: created}))
	saved, _, _ := appstate.Load(m.cfg.Root())
	if !saved.FirstNoteCreated.Equal(created) || saved.Streak != 2 {
		t.Fatalf("the first manual note should set first_note_created: %+v", saved)
	}
	m = applyAllMessages(m, m.saveNotesCmd(&model.Note{Summary: "older", Status: model.StatusActive, Source: model.SourceManual, Created: created.AddDate(-1, 0, 0)}))
	if saved, _, _ := appstate.Load(m.cfg.Root()); !saved.FirstNoteCreated.Equal(created) {
		t.Errorf("a later manual note should not move first_note_created: %v", saved.FirstNoteCreated)
	}
}

func TestBragListsWeeksFromTheStoredFirstNoteDate(t *testing.T) {
	now := time.Now()
	first := time.Date(now.Year()-2, time.March, 4, 9, 0, 0, 0, time.Local)
	m := appStateModel(t, &appstate.State{FirstNoteCreated: first}, twoActiveNotes(now)...)
	if !m.firstNoteTime().Equal(first) {
		t.Fatalf("first note time = %v, want %v", m.firstNoteTime(), first)
	}
	rows := m.bragRows()
	if rows[len(rows)-1].year != first.Year() {
		t.Errorf("brag should reach back to %d before notes finish loading, last row year %d", first.Year(), rows[len(rows)-1].year)
	}
}

func TestAppStateSavesKeepFieldsOtherProcessesWrote(t *testing.T) {
	now := time.Now()
	m := appStateModel(t, &appstate.State{Streak: 1}, twoActiveNotes(now)...)
	location := appstate.TUILocation{PID: 7, TmuxPane: "%9"}
	if err := appstate.Update(m.cfg.Root(), func(state *appstate.State) { state.TUILocation = &location }); err != nil {
		t.Fatal(err)
	}
	m = markDone(t, m, "first")
	next, cmd := m.toggleTagGroups(tea.KeyMsg{})
	applyAllMessages(next.(Model), cmd)
	saved, _, err := appstate.Load(m.cfg.Root())
	if err != nil || saved.TUILocation == nil || *saved.TUILocation != location || !saved.GroupByTag || saved.LastDoneDay != now.Format("2006-01-02") {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
}
