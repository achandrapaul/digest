package tui

import (
	"testing"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) loadReviewPollSnapshot() reviewPollSnapshot {
	return loadReviewPollSnapshot(m.reviewRoot(), m.bragRoot(), m.listedReviewRefs(), m.review.localReviews)
}

func (m Model) renderPRTag(item *GitPRItem, selected bool) string {
	return underlinedWhen(selected, joinTags(m.prTags(item, selected)))
}

func (m Model) reviewFooterItems(item *GitPRItem) []footerItem {
	return footerItemsFrom(m.prPreviewBindings(item))
}

func reviewRunFooterItems(running bool) []footerItem {
	status := review.RunFailed
	if running {
		status = review.RunRunning
	}
	item := NavItem{Kind: KindReviewRun, ReviewRun: &review.ReviewRun{Status: status}}
	return footerItemsFrom(Model{}.runPreviewBindings(item))
}

func (m Model) getYesterdayDoneNotes() []*model.Note {
	return m.groupNotes().previousDone
}

func (m Model) renderDashboardBody() string {
	if m.scrollPending {
		m.settleScroll()
	}
	return m.renderFrameBody(m.currentDashboardFrame())
}

func runJobProcess(cfg *config.Config, jobName string) error {
	return realStartJobProcess(jobLaunch{cfg: cfg, spec: findJobSpec(cfg, jobName)})
}

func dryRunJobProcess(spec config.JobSpec) error {
	return realStartJobProcess(jobLaunch{spec: spec, dryRun: true})
}

func (m Model) loadRunStatePoll() runStatePollTickMsg {
	return runStatePollTickMsg{states: m.jobStatesQuery(true).load()}
}

func (m Model) loadJobLogTick() jobLogTickMsg {
	jobName := m.previewedRunningJob()
	return loadJobLogTick(jobName, m.jobStatesQuery(false), m.jobs.jobLogStamp, m.jobDryRunOutputFor(jobName))
}

func settleBackground(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for _, msg := range collectMsgsWithoutTicks(cmd) {
		switch msg.(type) {
		case jobLaunchedMsg, reviewStartedMsg, bragStartedMsg, bragEditSavedMsg, automationStartedMsg, automationDraftSavedMsg, jobPreviewLoadedMsg:
			next, followUp := m.Update(msg)
			m = settleBackground(t, next.(Model), followUp)
		}
	}
	return m
}

func pressSettled(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, cmd := m.Update(msg)
	return settleBackground(t, next.(Model), cmd)
}
