package model_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"

	"gopkg.in/yaml.v3"
)

func localDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func TestStartedWorkingOnRoundTripsAsDayMonthYear(t *testing.T) {
	note := model.Note{Summary: "s", StartedWorkingOn: []string{"10-10-2026", "12-10-2026"}}
	encoded, err := yaml.Marshal(note)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "started_working_on:") || !strings.Contains(string(encoded), "12-10-2026") {
		t.Fatalf("encoded = %s", encoded)
	}
	var decoded model.Note
	if err := yaml.Unmarshal([]byte("started_working_on: [10-10-2026, 12-10-2026]\n"), &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.StartedWorkingOn, note.StartedWorkingOn) {
		t.Errorf("decoded = %v", decoded.StartedWorkingOn)
	}
	bare, _ := yaml.Marshal(model.Note{Summary: "bare"})
	if strings.Contains(string(bare), "started_working_on") {
		t.Errorf("empty list should be omitted: %s", bare)
	}
}

func TestStartedOnCountsOnlyTheLastReadableDate(t *testing.T) {
	note := model.Note{StartedWorkingOn: []string{"10-10-2026", "12-10-2026", "not-a-date"}}
	if !note.StartedOn(localDay(2026, 10, 12).Add(15 * time.Hour)) {
		t.Error("last readable date should count")
	}
	if note.StartedOn(localDay(2026, 10, 10)) {
		t.Error("an earlier date should not count")
	}
	if (&model.Note{}).StartedOn(localDay(2026, 10, 12)) {
		t.Error("no dates should never count")
	}
}

func TestWorkedOnDaysIncludeCreatedDaySortedAndDistinct(t *testing.T) {
	note := model.Note{
		Created:          localDay(2026, 10, 3).Add(9 * time.Hour),
		StartedWorkingOn: []string{"12-10-2026", "03-10-2026", "10-10-2026", "12-10-2026", "31-02-2026"},
	}
	want := []time.Time{localDay(2026, 10, 3), localDay(2026, 10, 10), localDay(2026, 10, 12)}
	if got := note.WorkedOnDays(); !reflect.DeepEqual(got, want) {
		t.Errorf("worked on days = %v, want %v", got, want)
	}
	if got := note.InvalidWorkDates(); !reflect.DeepEqual(got, []string{"31-02-2026"}) {
		t.Errorf("invalid dates = %v", got)
	}
}

func TestToggleWorkingOnAddsThenRemovesTheDay(t *testing.T) {
	note := model.Note{StartedWorkingOn: []string{"10-10-2026"}}
	today := localDay(2026, 10, 12).Add(11 * time.Hour)
	if started := note.ToggleWorkingOn(today); !started || !reflect.DeepEqual(note.StartedWorkingOn, []string{"10-10-2026", "12-10-2026"}) {
		t.Fatalf("started = %v, dates = %v", started, note.StartedWorkingOn)
	}
	if started := note.ToggleWorkingOn(today); started || !reflect.DeepEqual(note.StartedWorkingOn, []string{"10-10-2026"}) {
		t.Fatalf("started = %v, dates = %v", started, note.StartedWorkingOn)
	}
	single := model.Note{StartedWorkingOn: []string{"12-10-2026"}}
	single.ToggleWorkingOn(today)
	if single.StartedWorkingOn != nil {
		t.Errorf("removing the only date should leave no list, got %v", single.StartedWorkingOn)
	}
}

func TestRecordWorkedOnAppendsOnlyMissingDays(t *testing.T) {
	note := model.Note{StartedWorkingOn: []string{"12-10-2026"}}
	note.RecordWorkedOn(localDay(2026, 10, 12))
	note.RecordWorkedOn(localDay(2026, 10, 13).Add(18 * time.Hour))
	if !reflect.DeepEqual(note.StartedWorkingOn, []string{"12-10-2026", "13-10-2026"}) {
		t.Errorf("dates = %v", note.StartedWorkingOn)
	}
}
