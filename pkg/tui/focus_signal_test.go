package tui

import (
	"testing"

	"github.com/achandrapaul/digest/pkg/focus"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/sourcecontrol"
	"github.com/achandrapaul/digest/pkg/system"

	tea "github.com/charmbracelet/bubbletea"
)

func focusTestPR(owner, repo string, number int, title string) GitPRItem {
	ref := review.PRRef{Host: "github.com", Owner: owner, Repo: repo, Number: number, URL: "https://github.com/" + owner + "/" + repo + "/pull/" + string(rune('0'+number))}
	return sourcecontrol.NewPRItem(review.QueuedPR{Ref: ref, Title: title, CIState: "SUCCESS"}, "Pending Review")
}

func focusTestModel(t *testing.T) Model {
	t.Helper()
	m := selectionTestModel(t)
	m.applyGitPending(gitPendingMsg{generation: m.git.fetchGeneration, pending: []GitPRItem{
		focusTestPR("acme", "control", 1, "fix: header"),
		focusTestPR("acme", "control", 2, "feat: footer: sticky"),
	}})
	selectNavItem(t, &m, "job:janitor")
	return m
}

func requestFocus(t *testing.T, m Model, raw string) Model {
	t.Helper()
	selector, err := focus.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := focus.Write(m.cfg.Root(), selector); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(focusSignalMsg{})
	return next.(Model)
}

func TestFocusSignalSelectsTheMatchingPendingPR(t *testing.T) {
	m := requestFocus(t, focusTestModel(t), "acme:control:feat: footer: sticky")
	if want := "pr:" + focusTestPR("acme", "control", 2, "").URL; selectedKey(m) != want {
		t.Errorf("selected %q, want %q", selectedKey(m), want)
	}
	if system.Exists(focus.RequestPath(m.cfg.Root())) {
		t.Error("focus file should be deleted once focus is achieved")
	}
}

func TestFocusSignalFallsBackToTheReviewRun(t *testing.T) {
	m := focusTestModel(t)
	ref := review.PRRef{Owner: "acme", Repo: "mobile", Number: 9, URL: "https://github.com/acme/mobile/pull/9"}
	m.review.reviewRuns = []review.ReviewRun{{Meta: review.Meta{Ref: ref, Title: "chore: deps"}, Status: review.RunDone}}
	m = requestFocus(t, m, "acme:mobile:chore: deps")
	if selectedKey(m) != "review:"+ref.URL {
		t.Errorf("selected %q", selectedKey(m))
	}
}

func TestFocusSignalWithoutAMatchDeletesTheFile(t *testing.T) {
	m := requestFocus(t, focusTestModel(t), "acme:control:gone")
	if selectedKey(m) != "job:janitor" {
		t.Errorf("selection moved to %q", selectedKey(m))
	}
	if system.Exists(focus.RequestPath(m.cfg.Root())) {
		t.Error("an unmatched focus file should still be deleted")
	}
}

func TestFocusSignalWaitsForTheEditToEnd(t *testing.T) {
	for name, enterEdit := range map[string]func(*Model){
		"inline edit":        func(m *Model) { m.mode = ViewInlineEdit },
		"preview modal edit": func(m *Model) { m.mode = ViewPreview; m.pushMode(ViewEdit) },
	} {
		t.Run(name, func(t *testing.T) {
			m := focusTestModel(t)
			enterEdit(&m)
			m = requestFocus(t, m, "acme:control:fix: header")
			if selectedKey(m) != "job:janitor" {
				t.Fatalf("selection moved during an edit to %q", selectedKey(m))
			}
			if !system.Exists(focus.RequestPath(m.cfg.Root())) {
				t.Fatal("focus file should wait while editing")
			}
			m.mode, m.modeStack = ViewDashboard, nil
			next, _ := m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			m = next.(Model)
			if want := "pr:" + focusTestPR("acme", "control", 1, "").URL; selectedKey(m) != want {
				t.Errorf("after the edit selected %q, want %q", selectedKey(m), want)
			}
			if system.Exists(focus.RequestPath(m.cfg.Root())) {
				t.Error("focus file should be deleted after the edit")
			}
		})
	}
}
