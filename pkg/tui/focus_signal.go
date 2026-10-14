package tui

import (
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/achandrapaul/digest/pkg/focus"

	tea "github.com/charmbracelet/bubbletea"
)

type focusSignalMsg struct{}

var editingModes = []ViewMode{ViewEdit, ViewInlineEdit, ViewBragEdit, ViewAutomationEdit, ViewNotifyInput, ViewRejectComment, ViewSetup, ViewSetupDiscard}

func listenFocusSignal(program *tea.Program) func() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-signals:
				program.Send(focusSignalMsg{})
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(signals)
		close(done)
	}
}

func (m Model) editing() bool {
	return slices.Contains(editingModes, m.mode) || slices.ContainsFunc(m.modeStack, func(mode ViewMode) bool { return slices.Contains(editingModes, mode) })
}

func (m *Model) applyFocusRequest() {
	selector, found, err := focus.Take(m.cfg.Root())
	if err != nil {
		m.showError("FOCUS", err)
	}
	if !found {
		return
	}
	items := m.allNavItems()
	index := slices.IndexFunc(items, func(item NavItem) bool {
		return item.Kind == KindPendingGit && item.PendingGitPR != nil && item.PendingGitPR.PR != nil &&
			focusMatches(selector, item.PendingGitPR.PR.Ref.Owner, item.PendingGitPR.PR.Ref.Repo, item.PendingGitPR.PR.Title)
	})
	if index < 0 {
		index = slices.IndexFunc(items, func(item NavItem) bool {
			return item.Kind == KindReviewRun && item.ReviewRun != nil &&
				focusMatches(selector, item.ReviewRun.Meta.Ref.Owner, item.ReviewRun.Meta.Ref.Repo, item.ReviewRun.Meta.Title)
		})
	}
	if index < 0 {
		return
	}
	m.selected = index
	m.updateScrollOffset()
}

func focusMatches(selector focus.Selector, owner, repo, title string) bool {
	return strings.EqualFold(selector.Org, owner) && strings.EqualFold(selector.Repo, repo) && selector.Title == title
}
