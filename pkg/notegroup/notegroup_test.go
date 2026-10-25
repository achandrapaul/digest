package notegroup

import (
	"slices"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

func at(viewedDay time.Time, daysOffset, hour int) time.Time {
	return time.Date(viewedDay.Year(), viewedDay.Month(), viewedDay.Day(), hour, 0, 0, 0, time.Local).AddDate(0, 0, daysOffset)
}

func summaries(notes []*model.Note) []string {
	var names []string
	for _, note := range notes {
		names = append(names, note.Summary)
	}
	return names
}

func weekdaysOnly(weekday time.Weekday) bool {
	return weekday != time.Saturday && weekday != time.Sunday
}

func TestPreviousWorkDaySkipsTheWeekend(t *testing.T) {
	monday := time.Date(2026, 10, 19, 10, 0, 0, 0, time.Local)
	if got := PreviousWorkDay(monday, weekdaysOnly); got != time.Date(2026, 10, 16, 0, 0, 0, 0, time.Local) {
		t.Fatalf("previous work day of Monday = %v, want Friday", got)
	}
	tuesday := monday.AddDate(0, 0, 1)
	if got := PreviousWorkDay(tuesday, weekdaysOnly); got != StartOfDay(monday) {
		t.Fatalf("previous work day of Tuesday = %v, want Monday", got)
	}
}

func TestGroupSplitsNotesIntoThePreviousDayPendingTodayAndClosedToday(t *testing.T) {
	viewedDay := time.Date(2026, 10, 19, 15, 0, 0, 0, time.Local)
	previousDay := PreviousWorkDay(viewedDay, weekdaysOnly)
	startedToday := &model.Note{Summary: "started today", Status: model.StatusActive, Created: at(viewedDay, -5, 9), StartedWorkingOn: []string{viewedDay.Format(model.WorkDayLayout)}}
	notes := []*model.Note{
		{Summary: "done friday", Status: model.StatusDone, Source: model.SourceManual, Created: at(viewedDay, -4, 9), Updated: at(viewedDay, -3, 11)},
		{Summary: "done thursday", Status: model.StatusDone, Source: model.SourceManual, Created: at(viewedDay, -5, 9), Updated: at(viewedDay, -4, 11)},
		{Summary: "pending", Status: model.StatusActive, Source: model.SourceManual, Created: at(viewedDay, -2, 9)},
		{Summary: "added today", Status: model.StatusActive, Source: model.SourceManual, Created: at(viewedDay, 0, 9)},
		startedToday,
		{Summary: "pr closed today", Status: model.StatusDone, Source: model.SourceMyPR, Created: at(viewedDay, -1, 9), Updated: at(viewedDay, 0, 12)},
		{Summary: "closed today", Status: model.StatusDone, Source: model.SourceManual, Created: at(viewedDay, -1, 9), Updated: at(viewedDay, 0, 10)},
		{Summary: "archived", Status: model.StatusArchived, Created: at(viewedDay, -3, 9), Updated: at(viewedDay, -3, 10)},
	}
	groups := Group(notes, viewedDay, previousDay)
	checks := map[string][2][]string{
		"previous done": {summaries(groups.PreviousDone), {"done friday"}},
		"carried":       {summaries(groups.Carried), {"pending"}},
		"today":         {summaries(groups.Today), {"added today", "started today"}},
		"today done":    {summaries(groups.TodayDone), {"closed today", "pr closed today"}},
	}
	for name, check := range checks {
		if !slices.Equal(check[0], check[1]) {
			t.Errorf("%s = %v, want %v", name, check[0], check[1])
		}
	}
}

func TestSourceTagDefaultsToManual(t *testing.T) {
	if got := SourceTag(&model.Note{}); got != "#manual" {
		t.Fatalf("SourceTag = %q, want #manual", got)
	}
	if got := SourceTag(&model.Note{Source: model.SourceMyPR}); got != "#my-pr" {
		t.Fatalf("SourceTag = %q, want #my-pr", got)
	}
}

func TestPreviousWorkDayFallsBackToYesterdayWithNoWorkDays(t *testing.T) {
	monday := time.Date(2026, 10, 19, 10, 0, 0, 0, time.Local)
	if got := PreviousWorkDay(monday, func(time.Weekday) bool { return false }); got != time.Date(2026, 10, 18, 0, 0, 0, 0, time.Local) {
		t.Fatalf("previous work day with no work days = %v, want Sunday", got)
	}
}

func TestSameDayComparesTheLocalDate(t *testing.T) {
	morning := time.Date(2026, 10, 19, 0, 1, 0, 0, time.Local)
	if !SameDay(morning, morning.Add(23*time.Hour)) {
		t.Error("two moments on the same local day should match")
	}
	if SameDay(morning, morning.Add(-2*time.Minute)) {
		t.Error("a moment before midnight should be the previous day")
	}
}

func TestGroupSortsCarriedByTagDescendingThenNewestCreated(t *testing.T) {
	viewedDay := time.Date(2026, 10, 19, 15, 0, 0, 0, time.Local)
	notes := []*model.Note{
		{Summary: "old manual", Status: model.StatusActive, Source: model.SourceManual, Created: at(viewedDay, -3, 9)},
		{Summary: "review", Status: model.StatusActive, Source: model.SourcePRReview, Created: at(viewedDay, -5, 9)},
		{Summary: "new manual", Status: model.StatusActive, Source: model.SourceManual, Created: at(viewedDay, -1, 9)},
		{Summary: "my pr", Status: model.StatusActive, Source: model.SourceMyPR, Created: at(viewedDay, -2, 9)},
	}
	groups := Group(notes, viewedDay, PreviousWorkDay(viewedDay, weekdaysOnly))
	if got, want := summaries(groups.Carried), []string{"review", "my pr", "new manual", "old manual"}; !slices.Equal(got, want) {
		t.Errorf("carried = %v, want %v", got, want)
	}
}

func TestGroupSortsPreviousDoneBySourceThenOldestFirst(t *testing.T) {
	viewedDay := time.Date(2026, 10, 19, 15, 0, 0, 0, time.Local)
	notes := []*model.Note{
		{Summary: "review late", Status: model.StatusDone, Source: model.SourcePRReview, Created: at(viewedDay, -5, 9), Updated: at(viewedDay, -3, 16)},
		{Summary: "manual late", Status: model.StatusDone, Source: model.SourceManual, Created: at(viewedDay, -5, 9), Updated: at(viewedDay, -3, 15)},
		{Summary: "review early", Status: model.StatusDone, Source: model.SourcePRReview, Created: at(viewedDay, -5, 9), Updated: at(viewedDay, -3, 10)},
		{Summary: "manual early", Status: model.StatusDone, Source: model.SourceManual, Created: at(viewedDay, -5, 9), Updated: at(viewedDay, -3, 9)},
	}
	groups := Group(notes, viewedDay, PreviousWorkDay(viewedDay, weekdaysOnly))
	if got, want := summaries(groups.PreviousDone), []string{"manual early", "manual late", "review early", "review late"}; !slices.Equal(got, want) {
		t.Errorf("previous done = %v, want %v", got, want)
	}
}
