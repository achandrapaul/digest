package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
)

func summariseTestModel(t *testing.T) Model {
	t.Helper()
	m, _ := automationTestModel(t)
	m.cfg.Automations = append(m.cfg.Automations, config.AutomationSpec{Name: "summarise", Type: config.AutomationTypePRSummary, Match: []string{"check pr"}, DraftPrompt: "d"})
	m.notes = append(m.notes, &model.Note{ID: "note-2", Summary: "check pr", Body: "https://github.com/acme/api/pull/3", Created: m.currentDate, Source: model.SourceManual})
	selectNote(t, &m, "note-2")
	return m
}

func TestSummariseShowsInTheMenuOnceGHHostsLoad(t *testing.T) {
	m := summariseTestModel(t)
	if menuHasAutomation(m, "summarise") {
		t.Fatal("summarise should stay hidden until the gh hosts are known")
	}
	updated, _ := m.Update(ghHostsLoadedMsg{hosts: []string{"github.com"}})
	if !menuHasAutomation(updated.(Model), "summarise") {
		t.Error("summarise should show for a PR on a gh host")
	}
	updated, _ = m.Update(ghHostsLoadedMsg{hosts: []string{"code.example.com"}})
	if menuHasAutomation(updated.(Model), "summarise") {
		t.Error("summarise should stay hidden for a PR on another host")
	}
	if spec, found := updated.(Model).noteAutomation(updated.(Model).notes[len(m.notes)-1]); found && spec.Name == "summarise" {
		t.Error("noteAutomation should apply the same host check")
	}
}

func TestLoadGHHostsCmdReportsTheHostsOnce(t *testing.T) {
	calls := 0
	original := loadGHHosts
	loadGHHosts = func(context.Context) ([]string, error) {
		calls++
		return []string{"github.com"}, nil
	}
	t.Cleanup(func() { loadGHHosts = original })
	msg, ok := loadGHHostsCmd()().(ghHostsLoadedMsg)
	if !ok || calls != 1 || len(msg.hosts) != 1 || msg.hosts[0] != "github.com" {
		t.Errorf("msg = %+v, calls = %d", msg, calls)
	}
	loadGHHosts = func(context.Context) ([]string, error) { return nil, errors.New("logged out") }
	m := summariseTestModel(t)
	updated, _ := m.Update(loadGHHostsCmd()())
	if menuHasAutomation(updated.(Model), "summarise") {
		t.Error("a failed lookup should keep summarise hidden")
	}
}

func TestInitSkipsTheGHHostLookupWhenShowGitIsOff(t *testing.T) {
	original := loadGHHosts
	t.Cleanup(func() { loadGHHosts = original })
	loadGHHosts = func(context.Context) ([]string, error) { return []string{"github.com"}, nil }
	m := syncTestModel(t)
	showGit := false
	m.cfg.ShowGit = &showGit
	for _, msg := range collectMsgsWithoutTicks(m.Init()) {
		if _, isHosts := msg.(ghHostsLoadedMsg); isHosts {
			t.Fatal("Init should not look up gh hosts when show_git is off")
		}
	}
}
