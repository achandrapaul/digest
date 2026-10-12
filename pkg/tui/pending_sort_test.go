package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

func TestPendingSortsBySourceTagThenNewestCreated(t *testing.T) {
	now := time.Now()
	daysBack := func(days int) time.Time { return now.AddDate(0, 0, -days) }
	notes := []*model.Note{
		{Summary: "manual old", Status: model.StatusActive, Source: model.SourceManual, Created: daysBack(5), Updated: daysBack(5)},
		{Summary: "review new", Status: model.StatusActive, Source: model.SourcePRReview, Created: daysBack(1), Updated: daysBack(1)},
		{Summary: "manual new", Status: model.StatusActive, Source: model.SourceManual, Created: daysBack(2), Updated: daysBack(2)},
		{Summary: "review old", Status: model.StatusActive, Source: model.SourcePRReview, Created: daysBack(4), Updated: daysBack(4)},
		{Summary: "unsourced", Status: model.StatusActive, Created: daysBack(3), Updated: daysBack(3)},
	}
	m := noteSaveModel(t, notes...)
	want := []string{"review new", "review old", "manual new", "unsourced", "manual old"}
	if got := groupSummaries(m.groupNotes().carried); !slices.Equal(got, want) {
		t.Fatalf("pending order = %v, want %v", got, want)
	}
	var navOrder []string
	for _, item := range m.allNavItems() {
		if item.Kind == KindCarriedNote {
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
