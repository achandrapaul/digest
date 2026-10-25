package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/model"
)

func carriedReviewNotes(now time.Time) []*model.Note {
	today := startOfDay(now)
	yesterday := today.AddDate(0, 0, -1)
	return append(tagGroupNotes(now),
		&model.Note{Summary: "carried review two", Status: model.StatusActive, Source: model.SourcePRReview, Created: yesterday.Add(time.Minute), Updated: yesterday.Add(time.Minute)},
		&model.Note{Summary: "active review", Status: model.StatusActive, Source: model.SourcePRReview, Created: today.Add(4 * time.Minute), Updated: today.Add(4 * time.Minute)},
		&model.Note{Summary: "closed review", Status: model.StatusDone, Source: model.SourcePRReview, Created: today, Updated: today.Add(3 * time.Minute)},
	)
}

func carriedHeader(t *testing.T, view string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Pending, Carried Over") {
			return line
		}
	}
	t.Fatalf("no carried over header in\n%s", view)
	return ""
}

func TestVTogglesCarriedPRReviewsAndRemembersItInAppState(t *testing.T) {
	m := appStateModel(t, &appstate.State{Streak: 2}, carriedReviewNotes(time.Now())...)
	selectSummary(t, &m, "active my pr")
	if got := groupSummaries(m.groupNotes().carried); !slices.Contains(got, "carried review") || !slices.Contains(got, "carried review two") {
		t.Fatalf("pr reviews should start shown, carried = %v", got)
	}
	if header := carriedHeader(t, stripANSI(m.View())); !strings.Contains(header, "v "+eyeIcon+" #pr-review") || strings.Contains(header, "#pr-review (") {
		t.Errorf("header should hint v with the eye and no count while shown: %q", header)
	}

	next, cmd := pressKey(t, m, "v")
	m = applyAllMessages(next, cmd)
	if got := groupSummaries(m.groupNotes().carried); !slices.Equal(got, []string{"carried manual"}) {
		t.Errorf("v should hide carried pr reviews, carried = %v", got)
	}
	if header := carriedHeader(t, stripANSI(m.View())); !strings.Contains(header, "(1)") || !strings.Contains(header, "v "+eyeOffIcon+" #pr-review (2)") {
		t.Errorf("header should count visible notes and hint the hidden ones: %q", header)
	}
	if item, _ := m.selectedNavItem(); item.Note == nil || item.Note.Summary != "active my pr" {
		t.Errorf("selection should stay on the same note, got %+v", item)
	}
	saved, _, err := appstate.Load(m.cfg.Root())
	if err != nil || !saved.HideCarriedPRReviews || saved.Streak != 2 {
		t.Fatalf("saved state = %+v, %v", saved, err)
	}
	if reopened := NewModel(m.cfg, nil); slices.Contains(groupSummaries(reopened.groupNotes().carried), "carried review") {
		t.Error("pr reviews should stay hidden after a restart")
	}

	next, cmd = pressKey(t, m, "v")
	m = applyAllMessages(next, cmd)
	if saved, _, _ := appstate.Load(m.cfg.Root()); saved.HideCarriedPRReviews {
		t.Errorf("a second v should save false, got %v", saved.HideCarriedPRReviews)
	}
	if got := groupSummaries(m.groupNotes().carried); !slices.Contains(got, "carried review") {
		t.Errorf("a second v should show pr reviews again, carried = %v", got)
	}
}

func TestHidingCarriedPRReviewsLeavesOtherListsAlone(t *testing.T) {
	m := appStateModel(t, &appstate.State{HideCarriedPRReviews: true}, carriedReviewNotes(time.Now())...)
	groups := m.groupNotes()
	if !slices.Contains(groupSummaries(groups.today), "active review") {
		t.Errorf("active pr reviews should stay, today = %v", groupSummaries(groups.today))
	}
	if !slices.Contains(groupSummaries(groups.todayDone), "closed review") {
		t.Errorf("closed pr reviews should stay, closed = %v", groupSummaries(groups.todayDone))
	}
	if !slices.Contains(groupSummaries(groups.previousDone), "yesterday review") {
		t.Errorf("yesterday's pr reviews should stay, yesterday = %v", groupSummaries(groups.previousDone))
	}
	if navigable := navSummaries(m); slices.Contains(navigable, "carried review") {
		t.Errorf("hidden notes should not be navigable: %v", navigable)
	}
}

func TestHidingCarriedPRReviewsOrGroupingTagsDropsANoteRange(t *testing.T) {
	for _, key := range []string{"v", "h"} {
		m := appStateModel(t, &appstate.State{}, carriedReviewNotes(time.Now())...)
		selectSummary(t, &m, "carried manual")
		next, _ := pressKey(t, m, "J")
		if !next.notesUI.rangeActive {
			t.Fatalf("%s: J should start a range", key)
		}
		next, _ = pressKey(t, next, key)
		if next.notesUI.rangeActive {
			t.Errorf("%s should drop the range so x can't act on shifted rows", key)
		}
	}
}
