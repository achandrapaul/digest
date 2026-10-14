package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/model"
)

func tagGroupNotes(now time.Time) []*model.Note {
	today := startOfDay(now)
	yesterday := today.AddDate(0, 0, -1)
	return []*model.Note{
		{Summary: "yesterday manual", Status: model.StatusDone, Source: model.SourceManual, Created: yesterday, Updated: yesterday.Add(time.Hour)},
		{Summary: "yesterday review", Status: model.StatusDone, Source: model.SourcePRReview, Created: yesterday, Updated: yesterday.Add(2 * time.Hour)},
		{Summary: "carried manual", Status: model.StatusActive, Source: model.SourceManual, Created: yesterday, Updated: yesterday},
		{Summary: "carried review", Status: model.StatusActive, Source: model.SourcePRReview, Created: yesterday, Updated: yesterday},
		{Summary: "active manual one", Status: model.StatusActive, Source: model.SourceManual, Created: today.Add(time.Minute), Updated: today.Add(time.Minute)},
		{Summary: "active my pr", Status: model.StatusActive, Source: model.SourceMyPR, Created: today.Add(2 * time.Minute), Updated: today.Add(2 * time.Minute)},
		{Summary: "active manual two", Status: model.StatusActive, Source: model.SourceManual, Created: today.Add(3 * time.Minute), Updated: today.Add(3 * time.Minute)},
		{Summary: "closed manual", Status: model.StatusDone, Source: model.SourceManual, Created: today, Updated: today.Add(time.Minute)},
		{Summary: "closed my pr", Status: model.StatusDone, Source: model.SourceMyPR, Created: today, Updated: today.Add(2 * time.Minute)},
	}
}

func headerLines(view string) []string {
	var headers []string
	for _, line := range strings.Split(view, "\n") {
		if trimmed := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "│")); strings.HasPrefix(trimmed, "#") && !strings.Contains(trimmed, " ") {
			headers = append(headers, trimmed)
		}
	}
	return headers
}

func navSummaries(m Model) []string {
	var summaries []string
	for _, item := range m.allNavItems() {
		if item.Note != nil {
			summaries = append(summaries, item.Note.Summary)
		}
	}
	return summaries
}

func TestHTogglesTagGroupsAndRemembersItInAppState(t *testing.T) {
	m := appStateModel(t, &appstate.State{Streak: 2}, tagGroupNotes(time.Now())...)
	selectSummary(t, &m, "active my pr")
	if headers := headerLines(stripANSI(m.View())); len(headers) != 0 {
		t.Fatalf("grouping should start off, headers = %v", headers)
	}
	next, cmd := pressKey(t, m, "h")
	m = applyAllMessages(next, cmd)
	if !m.appState.GroupByTag {
		t.Fatal("h should turn grouping on")
	}
	if item, _ := m.selectedNavItem(); item.Note == nil || item.Note.Summary != "active my pr" {
		t.Errorf("selection should stay on the same note, got %+v", item)
	}
	saved, _, err := appstate.Load(m.cfg.Root())
	if err != nil || !saved.GroupByTag || saved.Streak != 2 {
		t.Fatalf("saved state = %+v, %v", saved, err)
	}
	if reopened := NewModel(m.cfg, nil); !reopened.appState.GroupByTag {
		t.Error("grouping should be on after a restart")
	}
	next, cmd = pressKey(t, m, "h")
	m = applyAllMessages(next, cmd)
	if saved, _, _ := appstate.Load(m.cfg.Root()); saved.GroupByTag || m.appState.GroupByTag {
		t.Error("a second h should turn grouping off")
	}
}

func TestTagGroupsPutAHeaderOverEachTagInTheNoteSections(t *testing.T) {
	m := appStateModel(t, &appstate.State{GroupByTag: true}, tagGroupNotes(time.Now())...)
	view := stripANSI(m.View())
	want := []string{"#manual", "#pr-review", "#pr-review", "#manual", "#manual", "#my-pr", "#manual", "#my-pr"}
	if got := headerLines(view); !slices.Equal(got, want) {
		t.Fatalf("headers = %v, want %v\n%s", got, want, view)
	}
	order := navSummaries(m)
	for index := 1; index < len(order); index++ {
		if strings.Index(view, order[index-1]) > strings.Index(view, order[index]) {
			t.Errorf("%q should render above %q", order[index-1], order[index])
		}
	}
}

func TestActiveTodayGathersTagsOnlyWhenGrouping(t *testing.T) {
	ungrouped := appStateModel(t, &appstate.State{}, tagGroupNotes(time.Now())...)
	if got := groupSummaries(ungrouped.groupNotes().today); !slices.Equal(got, []string{"active manual one", "active my pr", "active manual two"}) {
		t.Errorf("without grouping today = %v", got)
	}
	grouped := appStateModel(t, &appstate.State{GroupByTag: true}, tagGroupNotes(time.Now())...)
	if got := groupSummaries(grouped.groupNotes().today); !slices.Equal(got, []string{"active manual one", "active manual two", "active my pr"}) {
		t.Errorf("with grouping today = %v", got)
	}
}
