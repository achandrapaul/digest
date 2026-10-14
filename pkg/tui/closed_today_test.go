package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

func TestDoneStampsTheViewedDayWithTheCurrentClock(t *testing.T) {
	m := noteSaveModel(t)
	m.currentDate = time.Now().AddDate(0, 0, -2).Add(-3 * time.Hour)
	note := &model.Note{Summary: "task", Status: model.StatusActive}
	before := time.Now()
	m.toggleNoteDone(note)
	after := time.Now()
	if !isSameDay(note.Updated, m.currentDate) {
		t.Fatalf("done day = %v, want the viewed day %v", note.Updated, m.currentDate)
	}
	clock := func(moment time.Time) time.Duration { return moment.Sub(startOfDay(moment)) }
	if clock(note.Updated) < clock(before) || clock(note.Updated) > clock(after) {
		t.Errorf("done clock = %v, want between %v and %v", clock(note.Updated), clock(before), clock(after))
	}
}

func TestClosedTodayGroupsManualThenMyPRsThenReviewsNewestDoneFirst(t *testing.T) {
	today := startOfDay(time.Now())
	at := func(hour int) time.Time { return today.Add(time.Duration(hour) * time.Hour) }
	done := func(summary string, source model.Source, hour int) *model.Note {
		return &model.Note{Summary: summary, Status: model.StatusDone, Source: source, Created: at(0), Updated: at(hour)}
	}
	notes := []*model.Note{
		done("review early", model.SourcePRReview, 1),
		done("janitor", model.SourceJanitor, 9),
		done("manual early", model.SourceManual, 2),
		done("my pr", model.SourceMyPR, 3),
		done("review late", model.SourcePRReview, 8),
		done("unsourced late", "", 7),
		done("standup", model.SourceStandup, 10),
		done("manual mid", model.SourceManual, 5),
	}
	m := noteSaveModel(t, notes...)
	want := []string{"unsourced late", "manual mid", "manual early", "my pr", "review late", "review early", "janitor", "standup"}
	if got := groupSummaries(m.groupNotes().todayDone); !slices.Equal(got, want) {
		t.Fatalf("closed today order = %v, want %v", got, want)
	}
	var navOrder []string
	for _, item := range m.allNavItems() {
		if item.Kind == KindTodayDone {
			navOrder = append(navOrder, item.Note.Summary)
		}
	}
	if !slices.Equal(navOrder, want) {
		t.Errorf("navigation order = %v, want %v", navOrder, want)
	}
	view := stripANSI(m.View())
	for index := 1; index < len(want); index++ {
		if strings.Index(view, want[index-1]) > strings.Index(view, want[index]) {
			t.Errorf("%q should render above %q", want[index-1], want[index])
		}
	}
}
