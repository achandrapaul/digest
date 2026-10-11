package tui

import (
	"testing"

	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/review"
)

func TestBackgroundRunForCoversOnlyRunRows(t *testing.T) {
	for _, item := range []NavItem{
		{Kind: KindReviewRun, ReviewRun: &review.ReviewRun{}},
		{Kind: KindBragRun, BragRun: &brag.Run{}},
		{Kind: KindAutomationRun, AutomationRun: &automation.Run{}},
	} {
		if backgroundRunFor(item) == nil {
			t.Errorf("kind %v should be a background run", item.Kind)
		}
	}
	for _, item := range []NavItem{{Kind: KindJobDraft, Draft: &JobDraft{}}, {Kind: KindReviewRun}, {Kind: KindTodayNote}} {
		if backgroundRunFor(item) != nil {
			t.Errorf("item %+v is not a background run", item)
		}
	}
}

func TestBackgroundRunLabelsFollowTheRunState(t *testing.T) {
	var m Model
	running := NavItem{Kind: KindBragRun, BragRun: &brag.Run{Meta: brag.RunMeta{ID: "2026-W40"}, Status: brag.RunRunning}}
	failed := NavItem{Kind: KindBragRun, BragRun: &brag.Run{Meta: brag.RunMeta{ID: "2026-W40"}, Status: brag.RunFailed}}
	if m.runStopLabel(running) != "stop" || m.runStopLabel(failed) != "dismiss" || m.runStopLabel(NavItem{Kind: KindTodayNote}) != "" {
		t.Fatalf("labels = %q %q", m.runStopLabel(running), m.runStopLabel(failed))
	}
}
