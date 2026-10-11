package tui

import (
	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/review"

	tea "github.com/charmbracelet/bubbletea"
)

type backgroundRun interface {
	Label(m Model) string
	Running(m Model) bool
	Preview(m Model, width int) string
	Stop(m Model) (tea.Model, tea.Cmd)
	Dismiss(m Model) (tea.Model, tea.Cmd)
	Retry(m Model) (tea.Model, tea.Cmd)
}

func backgroundRunFor(item NavItem) backgroundRun {
	switch {
	case item.Kind == KindReviewRun && item.ReviewRun != nil:
		return reviewBackgroundRun{run: item.ReviewRun}
	case item.Kind == KindBragRun && item.BragRun != nil:
		return bragBackgroundRun{run: item.BragRun}
	case item.Kind == KindAutomationRun && item.AutomationRun != nil:
		return automationBackgroundRun{run: item.AutomationRun}
	}
	return nil
}

type reviewBackgroundRun struct{ run *review.ReviewRun }

func (r reviewBackgroundRun) Label(Model) string { return "review " + reviewRunLabel(*r.run) }

func (r reviewBackgroundRun) Running(Model) bool { return r.run.Status == review.RunRunning }

func (r reviewBackgroundRun) Preview(m Model, width int) string {
	return reviewRunPreview(m.reviewRoot(), *r.run).render(width)
}

func (r reviewBackgroundRun) Stop(m Model) (tea.Model, tea.Cmd) {
	if review.Status(review.StateDir(m.reviewRoot(), r.run.Meta.Ref)) != review.RunRunning {
		return r.Dismiss(m)
	}
	return m.beginReviewRunConfirm(reviewActionStop, r.run.Meta.Ref)
}

func (r reviewBackgroundRun) Dismiss(m Model) (tea.Model, tea.Cmd) {
	return m.dismissReviewRun(r.run.Meta.Ref)
}

func (r reviewBackgroundRun) Retry(m Model) (tea.Model, tea.Cmd) {
	meta := r.run.Meta
	return m, m.startReviewCmd(review.QueuedPR{Ref: meta.Ref, Title: meta.Title, HeadSHA: meta.HeadSHA}, true)
}

type bragBackgroundRun struct{ run *brag.Run }

func (r bragBackgroundRun) Label(Model) string { return bragRunLabel(*r.run) }

func (r bragBackgroundRun) Running(Model) bool { return r.run.Status == brag.RunRunning }

func (r bragBackgroundRun) Preview(m Model, width int) string {
	return bragRunPreview(m.bragRoot(), *r.run).render(width)
}

func (r bragBackgroundRun) Stop(m Model) (tea.Model, tea.Cmd) {
	if !brag.IsRunning(m.bragRoot(), r.run.Meta.ID) {
		return r.Dismiss(m)
	}
	run := *r.run
	m.bragRunToStop = &run
	return m.beginStopConfirm(), nil
}

func (r bragBackgroundRun) Dismiss(m Model) (tea.Model, tea.Cmd) {
	return m.stopOrDismissBragRun(r.run)
}

func (r bragBackgroundRun) Retry(m Model) (tea.Model, tea.Cmd) {
	period, err := brag.ParsePeriod(r.run.Meta.ID, bragClock().Location())
	if err != nil {
		m.showError("BRAG ERROR", err)
		return m, nil
	}
	return m, m.startBragCmd(period, r.run.Meta.Regenerate, true)
}

type automationBackgroundRun struct{ run *automation.Run }

func (r automationBackgroundRun) Label(m Model) string { return m.automationJobLabel(*r.run) }

func (r automationBackgroundRun) Running(m Model) bool { return m.automationJobRunning(r.run) }

func (r automationBackgroundRun) Preview(m Model, width int) string {
	return m.automationRunPreview(*r.run).render(width)
}

func (r automationBackgroundRun) Stop(m Model) (tea.Model, tea.Cmd) {
	if !m.automationJobRunning(r.run) {
		return r.Dismiss(m)
	}
	run := *r.run
	m.automationRunToStop = &run
	return m.beginStopConfirm(), nil
}

func (r automationBackgroundRun) Dismiss(m Model) (tea.Model, tea.Cmd) {
	return m.dismissAutomationRun(r.run)
}

func (r automationBackgroundRun) Retry(m Model) (tea.Model, tea.Cmd) {
	meta := r.run.Meta
	m.automation.automationNoteID, m.automation.automationName, m.automation.automationPhase = meta.NoteID, meta.Automation, meta.Phase
	m.pushMode(ViewAutomationConfirm)
	return m.confirmAutomation(tea.KeyMsg{})
}
