package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/store"

	tea "github.com/charmbracelet/bubbletea"
)

type appStateSavedMsg struct {
	err error
}

func saveAppStateCmd(root string, state appstate.State) tea.Cmd {
	return func() tea.Msg {
		return appStateSavedMsg{err: appstate.Save(root, state)}
	}
}

func (m *Model) loadAppState() {
	state, found, err := appstate.Load(m.cfg.Root())
	if err != nil {
		m.showError("STATE ERROR", err)
		return
	}
	m.appState, m.appStateKnown = state, found
}

func (m *Model) requireAppState() {
	_, found, err := appstate.Load(m.cfg.Root())
	var advice string
	switch {
	case err != nil:
		advice = fmt.Sprintf("The app state file can't be read: %v\n\nRun digest migrate to rebuild it.", err)
	case found:
		return
	case m.store.HasNotes():
		advice = appstate.Path(m.cfg.Root()) + " is missing.\n\nRun digest migrate to recreate it."
	default:
		advice = "digest isn't set up yet.\n\nRun digest bootstrap."
	}
	m.errorTitle, m.errorLines, m.quitOnErrorDismiss = "APP STATE", strings.Split(advice, "\n"), true
	m.pushMode(ViewError)
}

func (m *Model) appStateAfterSaves(saved []savedNoteResult) tea.Cmd {
	if !m.appStateKnown {
		return nil
	}
	state, changed := m.appState, false
	for _, result := range saved {
		var created, done bool
		if result.before.FilePath == "" {
			state, created = state.NoteCreated(&result.after)
		}
		if !store.IsDonePath(result.before.FilePath) && result.after.Status == model.StatusDone {
			state, done = state.MarkDone(time.Now(), m.cfg.IsWorkDay)
		}
		changed = changed || created || done
	}
	if !changed {
		return nil
	}
	m.appState = state
	return saveAppStateCmd(m.cfg.Root(), state)
}
