package appstate

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/paths"
)

func weekdaysOnly(day time.Weekday) bool {
	return day != time.Saturday && day != time.Sunday
}

var friday = time.Date(2026, 10, 9, 15, 0, 0, 0, time.Local)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, found, err := Load(root); found || err != nil {
		t.Fatalf("missing file = %v, %v", found, err)
	}
	state := State{FirstNoteCreated: friday.AddDate(-1, 0, 0), Streak: 4, LastDoneDay: "2026-10-09"}
	if err := Save(root, state); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := Load(root)
	if err != nil || !found || !loaded.FirstNoteCreated.Equal(state.FirstNoteCreated) || loaded.Streak != 4 || loaded.LastDoneDay != "2026-10-09" {
		t.Fatalf("loaded = %+v, %v, %v", loaded, found, err)
	}
	if info, err := os.Stat(Path(root)); err != nil || info.Mode().Perm() != paths.PrivateFileMode {
		t.Errorf("state file should be owner only: %v", err)
	}
}

func TestCorruptFileIsAnError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(Path(root), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(root); err == nil {
		t.Error("a corrupt state file should be reported")
	}
}

func TestCurrentStreak(t *testing.T) {
	cases := []struct {
		name     string
		lastDone string
		now      time.Time
		want     int
	}{
		{"done today", "2026-10-09", friday, 3},
		{"done on the previous work day", "2026-10-08", friday, 3},
		{"weekend keeps friday's streak", "2026-10-09", friday.AddDate(0, 0, 2), 3},
		{"monday still counts friday", "2026-10-09", friday.AddDate(0, 0, 3), 3},
		{"a missed work day ends it", "2026-10-07", friday, 0},
		{"never done", "", friday, 0},
	}
	for _, tc := range cases {
		state := State{Streak: 3, LastDoneDay: tc.lastDone}
		if got := state.CurrentStreak(tc.now, weekdaysOnly); got != tc.want {
			t.Errorf("%s: streak = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestMarkDone(t *testing.T) {
	state := State{Streak: 3, LastDoneDay: "2026-10-08"}
	next, changed := state.MarkDone(friday, weekdaysOnly)
	if !changed || next.Streak != 4 || next.LastDoneDay != "2026-10-09" {
		t.Fatalf("first done today = %+v, %v", next, changed)
	}
	if again, changed := next.MarkDone(friday.Add(time.Hour), weekdaysOnly); changed || !reflect.DeepEqual(again, next) {
		t.Errorf("a second done the same day should change nothing: %+v, %v", again, changed)
	}
	if weekend, changed := next.MarkDone(friday.AddDate(0, 0, 1), weekdaysOnly); changed || !reflect.DeepEqual(weekend, next) {
		t.Errorf("a done on a non-work day should change nothing: %+v, %v", weekend, changed)
	}
	missed := State{Streak: 3, LastDoneDay: "2026-10-06"}
	if reset, changed := missed.MarkDone(friday, weekdaysOnly); !changed || reset.Streak != 1 {
		t.Errorf("after a missed work day the streak restarts at 1: %+v", reset)
	}
}

func TestFromNotes(t *testing.T) {
	notes := []*model.Note{
		{Status: model.StatusActive, Created: friday.AddDate(0, -2, 0)},
		{Status: model.StatusDone, Created: friday.AddDate(0, 0, -3), Updated: friday.AddDate(0, 0, -1)},
		{Status: model.StatusDone, Created: friday.AddDate(0, 0, -3), Updated: friday.AddDate(0, 0, -2)},
		{Status: model.StatusDone, Created: friday.AddDate(0, 0, -3), Updated: friday.AddDate(0, 0, -6)},
	}
	state := FromNotes(notes, friday, weekdaysOnly)
	if !state.FirstNoteCreated.IsZero() || state.Streak != 2 || state.LastDoneDay != "2026-10-08" {
		t.Errorf("state = %+v", state)
	}
}

func TestZeroFirstNoteCreatedIsSavedAsNull(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, State{}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(Path(root))
	if !strings.Contains(string(data), `"first_note_created": null`) {
		t.Errorf("an empty first note date should be null: %s", data)
	}
	loaded, found, err := Load(root)
	if err != nil || !found || !loaded.FirstNoteCreated.IsZero() || loaded.Streak != 0 {
		t.Errorf("loaded = %+v, %v, %v", loaded, found, err)
	}
}

func TestNoteCreatedSetsTheDateOnceForAManualNote(t *testing.T) {
	first := friday.AddDate(0, -1, 0)
	for _, source := range []model.Source{model.SourceMyPR, model.SourcePRReview, ""} {
		if _, changed := (State{}).NoteCreated(&model.Note{Source: source, Created: first}); changed {
			t.Errorf("a %q note should never set the first note date", source)
		}
	}
	if _, changed := (State{}).NoteCreated(&model.Note{Source: model.SourceManual}); changed {
		t.Error("a note without a created time should not set the date")
	}
	state, changed := State{Streak: 3}.NoteCreated(&model.Note{Source: model.SourceManual, Created: first})
	if !changed || !state.FirstNoteCreated.Equal(first) || state.Streak != 3 {
		t.Fatalf("the first manual note should set the date: %+v, %v", state, changed)
	}
	if again, changed := state.NoteCreated(&model.Note{Source: model.SourceManual, Created: first.AddDate(-1, 0, 0)}); changed || !again.FirstNoteCreated.Equal(first) {
		t.Errorf("an existing first note date should never move: %+v, %v", again, changed)
	}
}

func TestFirstManualNoteCreatedIgnoresOtherSources(t *testing.T) {
	earliestManual := friday.AddDate(0, -2, 0)
	notes := []*model.Note{
		{Source: model.SourceMyPR, Created: friday.AddDate(-1, 0, 0)},
		{Source: model.SourcePRReview, Created: friday.AddDate(-1, 0, 0)},
		{Source: model.SourceManual, Created: friday},
		{Source: model.SourceManual, Created: earliestManual},
		{Source: model.SourceManual},
	}
	if got := FirstManualNoteCreated(notes); !got.Equal(earliestManual) {
		t.Errorf("first manual note = %v, want %v", got, earliestManual)
	}
	if got := FirstManualNoteCreated(notes[:2]); !got.IsZero() {
		t.Errorf("no manual notes should give no date: %v", got)
	}
}

func TestActionMenuOrderRoundTripsAndStaysOptional(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, State{ActionMenuOrder: []string{"jira", "notify"}}); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := Load(root)
	if err != nil || len(loaded.ActionMenuOrder) != 2 || loaded.ActionMenuOrder[0] != "jira" {
		t.Fatalf("loaded = %+v, %v", loaded, err)
	}
	if err := Save(root, State{Streak: 1}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(Path(root)); strings.Contains(string(data), "action_menu_order") {
		t.Errorf("an empty order should not be written: %s", data)
	}
}

func TestGroupByTagRoundTripsAndStaysOptional(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, State{GroupByTag: true}); err != nil {
		t.Fatal(err)
	}
	if loaded, _, err := Load(root); err != nil || !loaded.GroupByTag {
		t.Fatalf("loaded = %+v, %v", loaded, err)
	}
	if data, _ := os.ReadFile(Path(root)); !strings.Contains(string(data), `"group_by_tag": true`) && !strings.Contains(string(data), `"group_by_tag":true`) {
		t.Errorf("group_by_tag should be written: %s", data)
	}
	if err := Save(root, State{Streak: 1}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(Path(root)); strings.Contains(string(data), "group_by_tag") {
		t.Errorf("grouping off should not be written: %s", data)
	}
}
