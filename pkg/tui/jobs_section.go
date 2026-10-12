package tui

import (
	"fmt"
	"time"

	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/jobs"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/system"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type jobsSection struct{}

type JobDraft struct {
	Name           string
	DryRunCommand  string
	Command        string
	ExitCode       int
	HasRunDryRun   bool
	DryRunInFlight bool
}

type jobLogTickMsg struct {
	jobName      string
	states       jobStates
	stamp        string
	output       jobOutput
	outputLoaded bool
}

const jobLogTailLines = 400

type runStatePollTickMsg struct {
	states jobStates
}

type jobPreviewLoadedMsg struct {
	jobName string
	output  jobOutput
}

type jobAbortedMsg struct {
	jobName string
	err     error
}

func (m Model) getJobDrafts() []*JobDraft {
	if m.cfg == nil || len(m.cfg.JobList()) == 0 {
		return nil
	}

	var drafts []*JobDraft
	for _, j := range m.cfg.JobList() {
		exitCode := 1
		hasRun := false
		if m.jobs.jobDryRunHasRun != nil && m.jobs.jobDryRunHasRun[j.Name] {
			hasRun = true
			exitCode = m.jobs.jobDryRunExitCodes[j.Name]
		}

		drafts = append(drafts, &JobDraft{
			Name:           j.Name,
			DryRunCommand:  j.DryRunCommand,
			Command:        j.Command,
			ExitCode:       exitCode,
			HasRunDryRun:   hasRun,
			DryRunInFlight: m.jobs.dryRunsInFlight[j.Name],
		})
	}
	return drafts
}

func (m Model) tickRunStatePollCmd() tea.Cmd {
	query := m.jobStatesQuery(true)
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return runStatePollTickMsg{states: query.load()}
	})
}

func (m *Model) ensureRunStatePoll() tea.Cmd {
	if m.jobs.runStatePolling {
		return nil
	}
	m.jobs.runStatePolling = true
	return m.tickRunStatePollCmd()
}

func (m Model) tickJobLogCmd() tea.Cmd {
	jobName := m.previewedRunningJob()
	query, knownStamp, dryRunOutput := m.jobStatesQuery(false), m.jobs.jobLogStamp, m.jobDryRunOutputFor(jobName)
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return loadJobLogTick(jobName, query, knownStamp, dryRunOutput)
	})
}

func loadJobLogTick(jobName string, query jobStatesQuery, knownStamp, dryRunOutput string) jobLogTickMsg {
	if jobName == "" {
		return jobLogTickMsg{}
	}
	msg := jobLogTickMsg{jobName: jobName, states: query.load()}
	msg.stamp = jobRuntime().LogStamp(jobName, msg.states.runningPIDs[jobName] > 0)
	if msg.stamp != knownStamp {
		msg.output, msg.outputLoaded = loadJobOutput(jobName, dryRunOutput), true
	}
	return msg
}

func loadJobPreviewCmd(jobName, dryRunOutput string) tea.Cmd {
	return func() tea.Msg {
		return jobPreviewLoadedMsg{jobName: jobName, output: loadJobOutput(jobName, dryRunOutput)}
	}
}

func (m *Model) ensureJobLogRefresh() tea.Cmd {
	if m.jobs.jobLogRunning {
		return nil
	}
	m.jobs.jobLogRunning = true
	return m.tickJobLogCmd()
}

func (m Model) handleJobLogTick(msg jobLogTickMsg) (tea.Model, tea.Cmd) {
	jobName := m.previewedRunningJob()
	if jobName == "" {
		m.jobs.jobLogRunning = false
		return m, nil
	}
	if msg.jobName != jobName {
		return m, m.tickJobLogCmd()
	}
	m.applyJobStates(msg.states)
	running := m.jobRunning(jobName)
	if msg.stamp != m.jobs.jobLogStamp {
		m.jobs.jobLogStamp = msg.stamp
		wasAtBottom := m.previewViewport.AtBottom()
		previousOffset := m.previewViewport.YOffset
		if msg.outputLoaded {
			m.showJobPreview(msg.output)
		} else {
			m.updatePreviewViewport()
		}
		if wasAtBottom {
			m.previewViewport.GotoBottom()
		} else {
			m.previewViewport.SetYOffset(previousOffset)
		}
	}
	if !running {
		m.jobs.jobLogRunning = false
		return m, nil
	}
	return m, m.tickJobLogCmd()
}

func (m Model) runSelectedJob(tea.KeyMsg) (tea.Model, tea.Cmd) {
	item, found := m.selectedNavItem()
	if !found {
		return m, nil
	}
	if m.runStopLabel(item) == "dismiss" {
		retried := item
		m.runToRetry = &retried
		return m.beginStopConfirm(), nil
	}
	if item.Kind != KindJobDraft || item.Draft == nil || jobRuntime().IsRunning(item.Draft.Name) {
		return m, nil
	}
	m.jobs.jobToExecute = item.Draft.Name
	m.jobs.jobRunStopsDryRun = jobRuntime().DryRunInFlight(item.Draft.Name)
	m.jobs.jobConfirmError = ""
	return m.beginStopConfirm(), nil
}

func (m Model) dryRunSelectedJob(tea.KeyMsg) (tea.Model, tea.Cmd) {
	item, found := m.selectedNavItem()
	if !found || item.Kind != KindJobDraft || item.Draft == nil || jobRuntime().IsRunning(item.Draft.Name) || !jobs.HasDryRun(item.Draft.DryRunCommand) || m.jobs.dryRunsInFlight[item.Draft.Name] {
		return m, nil
	}
	m.jobs.jobDryRunToStart = item.Draft.Name
	return m.beginStopConfirm(), nil
}

func (m Model) stopSelectedItem(tea.KeyMsg) (tea.Model, tea.Cmd) {
	item, found := m.selectedNavItem()
	if !found {
		return m, nil
	}
	if run := backgroundRunFor(item); run != nil {
		return run.Stop(m)
	}
	switch {
	case item.Kind == KindJobDraft && item.Draft != nil && jobRuntime().IsRunning(item.Draft.Name):
		m.jobs.jobToAbort = item.Draft.Name
		return m.beginStopConfirm(), nil
	case item.Kind == KindJobDraft && item.Draft != nil && m.jobRunning(item.Draft.Name):
		m.jobs.staleJobToClear = item.Draft.Name
		return m.beginStopConfirm(), nil
	}
	return m, nil
}

func (m Model) renderDraftRow(draft *JobDraft, selected bool, width int) string {
	icon := amberDiamond.Render()
	if draft.HasRunDryRun && draft.ExitCode == 0 {
		icon = checkDone.Render()
	}
	return renderJobStyleRow(icon, draft.Name, m.draftRowTag(draft), selected, width)
}

func (m Model) draftRowTag(draft *JobDraft) string {
	if m.jobRunning(draft.Name) {
		return fmt.Sprintf("%s   %s", jobActiveTagStyle.Render("#job"), m.renderJobRunningIndicator())
	}
	if draft.DryRunInFlight {
		return fmt.Sprintf("%s   %s", jobActiveTagStyle.Render("#job"), m.renderDryRunIndicator())
	}
	statusText := "act"
	if draft.HasRunDryRun && draft.ExitCode == 0 {
		statusText = "done"
	}
	return fmt.Sprintf("%s   %s", dimBlueText.Render("#job"), mutedStyle.Render(statusText))
}

func (m Model) renderReviewRunRow(run review.ReviewRun, selected bool, width int) string {
	return renderJobStyleRow(amberDiamond.Render(), reviewRunLabel(run), m.reviewRunRowTag(run), selected, width)
}

func (m Model) reviewRunRowTag(run review.ReviewRun) string {
	if run.Status == review.RunRunning {
		return m.renderReviewRunningIndicator()
	}
	return stateStyle(review.StateFailed).Render("failed")
}

func (m Model) renderBragRunRow(run brag.Run, selected bool, width int) string {
	return renderJobStyleRow(amberDiamond.Render(), bragRunLabel(run), m.bragRunRowTag(run), selected, width)
}

func (m Model) bragRunRowTag(run brag.Run) string {
	if run.Status == brag.RunRunning {
		return m.renderPulseIndicator("bragging...")
	}
	return stateStyle(review.StateFailed).Render("failed")
}

func (m Model) renderAutomationRunRow(run automation.Run, selected bool, width int) string {
	return renderJobStyleRow(amberDiamond.Render(), m.automationJobLabel(run), m.automationRunRowTag(run), selected, width)
}

func (m Model) automationRunRowTag(run automation.Run) string {
	switch {
	case run.Status == automation.RunFailed:
		return stateStyle(review.StateFailed).Render("failed")
	case run.Status == automation.RunNeedsReauth:
		return stateStyle(review.StateFailed).Render("needs re-auth")
	case run.Meta.Phase == automation.PhaseDraft:
		return m.renderPulseIndicator("drafting...")
	default:
		return m.renderPulseIndicator("running...")
	}
}

func renderJobStyleRow(icon, label, rightBlock string, selected bool, width int) string {
	label = ansi.Truncate(label, max(width-lipgloss.Width(rightBlock)-6, 5), "…")
	labelText := itemStyle.Render(label)
	if selected {
		labelText = selectedTitle(label)
	}
	return alignRight("   "+icon+" "+labelText, []string{rightBlock}, width, selected) + "\n"
}

func (jobsSection) ApplyMessage(m Model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case jobLogTickMsg:
		return messageHandled(m.handleJobLogTick(msg))

	case jobPreviewLoadedMsg:
		if m.mode == ViewPreview && m.previewedJobState().name == msg.jobName {
			m.showJobPreview(msg.output)
		}
		return messageHandled(m, nil)

	case jobLaunchedMsg:
		return messageHandled(m.applyJobLaunch(msg))

	case reviewStartedMsg:
		return messageHandled(m.applyReviewStarted(msg))

	case bragStartedMsg:
		return messageHandled(m.applyBragStarted(msg))

	case bragEditSavedMsg:
		return messageHandled(m.applyBragEditSaved(msg))

	case automationStartedMsg:
		return messageHandled(m.applyAutomationStarted(msg))

	case automationDraftSavedMsg:
		return messageHandled(m.applyAutomationDraftSaved(msg))

	case runStatePollTickMsg:
		previousJobState := m.previewedJobState()
		m.applyJobStates(msg.states)
		stillBusy := m.isAnyDryRunInFlight() || m.isAnyJobRunning()
		var previewCmd tea.Cmd
		if current := m.previewedJobState(); m.mode == ViewPreview && !m.jobs.jobLogRunning && current != previousJobState {
			previewCmd = loadJobPreviewCmd(current.name, m.jobDryRunOutputFor(current.name))
		}
		if stillBusy {
			return messageHandled(m, tea.Batch(previewCmd, m.tickRunStatePollCmd()))
		}
		m.jobs.runStatePolling = false
		return messageHandled(m, previewCmd)

	case jobAbortedMsg:
		m.refreshJobStates()
		if m.mode == ViewPreview {
			m.updatePreviewViewport()
		}
		if msg.err != nil {
			m.showError("JOB ERROR", msg.err)
		}
		return messageHandled(m, nil)

	case reviewPollTickMsg:
		return messageHandled(m.handleReviewPoll(msg.snapshot))
	}
	return m, nil, false
}

var jobsKeystrokes sectionKeystrokes

func init() {
	jobsKeystrokes = sectionKeystrokes{
		actionRunSelectedJob:    Model.runSelectedJob,
		actionDryRunSelectedJob: Model.dryRunSelectedJob,
		actionStopSelectedItem:  Model.stopSelectedItem,
	}
}

func (jobsSection) ApplyKeystrokes(m Model, binding keyBinding, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return jobsKeystrokes.apply(m, binding, msg)
}

func (jobsSection) Render(m Model, builder *dashboardBuilder, data dashboardData) {
	builder.board.WriteString(sectionGap + "  " + m.renderSubSection("Jobs", 0, false, builder.selectedWithin(data.pendingEnd, data.jobsEnd)) + "\n")
	if data.jobsCount == 0 {
		builder.board.WriteString(mutedStyle.Render("   (no jobs configured)\n"))
	}
	for _, draft := range data.drafts {
		builder.emitRow(func(selected bool) string { return m.renderDraftRow(draft, selected, builder.innerWidth) })
	}
	for _, run := range m.review.reviewRuns {
		builder.emitRow(func(selected bool) string { return m.renderReviewRunRow(run, selected, builder.innerWidth) })
	}
	for _, run := range m.brag.bragRuns {
		builder.emitRow(func(selected bool) string { return m.renderBragRunRow(run, selected, builder.innerWidth) })
	}
	for _, run := range data.automationRuns {
		builder.emitRow(func(selected bool) string { return m.renderAutomationRunRow(run, selected, builder.innerWidth) })
	}
}

func (jobsSection) appendNavItems(m Model, items []NavItem, data dashboardData) []NavItem {
	for _, d := range data.drafts {
		items = append(items, NavItem{Kind: KindJobDraft, Draft: d})
	}
	for i := range m.review.reviewRuns {
		items = append(items, NavItem{Kind: KindReviewRun, ReviewRun: &m.review.reviewRuns[i]})
	}
	for i := range m.brag.bragRuns {
		items = append(items, NavItem{Kind: KindBragRun, BragRun: &m.brag.bragRuns[i]})
	}
	for i := range data.automationRuns {
		items = append(items, NavItem{Kind: KindAutomationRun, AutomationRun: &data.automationRuns[i]})
	}
	return items
}

func (m Model) dismissReviewRun(ref review.PRRef) (tea.Model, tea.Cmd) {
	if err := review.Dismiss(m.reviewRoot(), ref); err != nil {
		m.showError("REVIEW ERROR", err)
		return m, nil
	}
	selectedKey, selectedOccurrence := m.selectedNavKey()
	m.refreshReviewRuns()
	m.restoreSelection(selectedKey, selectedOccurrence)
	if m.mode == ViewPreview {
		if _, ok := m.selectedNavItem(); ok {
			m.updatePreviewViewport()
		} else {
			m.mode = ViewDashboard
		}
	}
	m.clampScreenSelection()
	return m, nil
}

func (m Model) jobConfirmPrompt() (title, prompt string, found bool) {
	switch {
	case m.jobs.jobToAbort != "":
		return deleteTitleStyle.Render(" ABORT JOB "), fmt.Sprintf("Are you sure you want to abort running job '%s'?", m.jobs.jobToAbort), true
	case m.jobs.jobToExecute != "" && m.jobs.jobRunStopsDryRun:
		prompt = fmt.Sprintf("Dry run in progress. Stop it and run the job?\n\nThe dry run of '%s' is killed first.", m.jobs.jobToExecute)
		return modalTitleStyle.Render(" EXECUTE JOB "), m.withJobConfirmError(prompt), true
	case m.jobs.jobToExecute != "":
		return modalTitleStyle.Render(" EXECUTE JOB "), m.withJobConfirmError(fmt.Sprintf("Are you sure you want to run '%s'?", m.jobs.jobToExecute)), true
	case m.jobs.jobDryRunToStart != "":
		return modalTitleStyle.Render(" DRY RUN "), fmt.Sprintf("Dry run '%s'?", m.jobs.jobDryRunToStart), true
	case m.jobs.staleJobToClear != "":
		return deleteTitleStyle.Render(" STALE JOB "), fmt.Sprintf("'%s' is no longer running.\n\nClear its stale state?", m.jobs.staleJobToClear), true
	case m.runToRetry != nil:
		return modalTitleStyle.Render(" RETRY JOB "), fmt.Sprintf("Run again?\n\n%s", m.runLabel(*m.runToRetry)), true
	case m.draftToDelete != "":
		return deleteTitleStyle.Render(" DELETE DRAFT "), fmt.Sprintf("Delete the %s draft for this note?", m.automation.automationRuns[m.draftToDelete].Meta.Automation), true
	case m.discardingEdit:
		return deleteTitleStyle.Render(" DISCARD "), "Discard changes?", true
	}
	if target := m.stopTargetName(); target != "" {
		return deleteTitleStyle.Render(" STOP JOB "), fmt.Sprintf("Are you sure you want to stop '%s'?", target), true
	}
	return "", "", false
}

func (m Model) withJobConfirmError(prompt string) string {
	if m.jobs.jobConfirmError == "" {
		return prompt
	}
	return prompt + "\n\n" + stateStyle(review.StateFailed).Render(m.jobs.jobConfirmError)
}

func (m Model) runLabel(item NavItem) string {
	if run := backgroundRunFor(item); run != nil {
		return run.Label(m)
	}
	return ""
}

func (m Model) confirmJobPrompt() (tea.Model, tea.Cmd, bool) {
	switch {
	case m.discardingEdit:
		next, cmd := m.confirmDiscardEdit()
		return next, cmd, true
	case m.jobs.jobToAbort != "":
		jobName := m.jobs.jobToAbort
		m.jobs.jobToAbort = ""
		m.returnFromJobConfirm()
		return m, abortJobCmd(jobName), true
	case m.jobs.jobToExecute != "":
		next, cmd := m.confirmRunJob()
		return next, cmd, true
	case m.jobs.jobDryRunToStart != "":
		jobName := m.jobs.jobDryRunToStart
		m.jobs.jobDryRunToStart = ""
		m.returnFromJobConfirm()
		return m, m.startJobDryRunCmd(jobName), true
	case m.jobs.staleJobToClear != "":
		jobName := m.jobs.staleJobToClear
		m.jobs.staleJobToClear = ""
		clearStaleJobPID(jobName)
		m.refreshJobStates()
		m.returnFromJobConfirm()
		return m, nil, true
	case m.runToRetry != nil:
		item := *m.runToRetry
		m.runToRetry = nil
		m.returnFromJobConfirm()
		next, cmd := m.retryRun(item)
		return next, cmd, true
	case m.draftToDelete != "":
		noteID := m.draftToDelete
		m.draftToDelete = ""
		m.mode = m.popMode()
		next, cmd := m.deleteDraft(noteID)
		return next, cmd, true
	case m.bragRunToStop != nil || m.automationRunToStop != nil:
		next, cmd := m.confirmStopRun()
		return next, cmd, true
	}
	return m, nil, false
}

func (m *Model) returnFromJobConfirm() {
	m.mode = m.popMode()
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
}

func (m Model) confirmRunJob() (tea.Model, tea.Cmd) {
	jobName := m.jobs.jobToExecute
	if !m.beginLaunch(jobLaunchKey(jobName, false)) {
		return m, nil
	}
	return m, launchJobCmd(jobLaunch{cfg: m.cfg, spec: jobs.FindSpec(m.cfg, jobName), stopDryRunFirst: m.jobs.jobRunStopsDryRun, states: m.jobStatesQuery(m.jobs.jobRunStopsDryRun)})
}

func (m Model) applyJobLaunch(msg jobLaunchedMsg) (tea.Model, tea.Cmd) {
	jobName := msg.launch.spec.Name
	m.finishLaunch(jobLaunchKey(jobName, msg.launch.dryRun))
	m.applyJobStates(msg.states)
	if msg.launch.dryRun {
		if m.mode == ViewPreview {
			m.updatePreviewViewport()
		}
		if msg.err != nil {
			m.showError("JOB ERROR", msg.err)
			return m, nil
		}
		return m, tea.Batch(m.ensureRunStatePoll(), m.ensureSyncPulse())
	}
	confirming := m.mode == ViewDeleteConfirm && m.jobs.jobToExecute == jobName
	if msg.dryRunStopped && confirming {
		m.jobs.jobRunStopsDryRun = false
	}
	if msg.err != nil {
		if !confirming {
			m.showError("JOB ERROR", msg.err)
			return m, nil
		}
		return m.failJobConfirm(msg.err), nil
	}
	if !confirming {
		return m, tea.Batch(m.ensureSyncPulse(), m.ensureRunStatePoll())
	}
	m.jobs.jobToExecute, m.jobs.jobConfirmError = "", ""
	if m.returnMode() == ViewPreview {
		m.previewNotice = ""
	}
	m.returnFromJobConfirm()
	return m, tea.Batch(m.ensureJobLogRefresh(), m.ensureSyncPulse(), m.ensureRunStatePoll())
}

func (m Model) failJobConfirm(err error) Model {
	m.showError("JOB ERROR", err)
	m.screenError = ""
	m.jobs.jobConfirmError = err.Error()
	if m.returnMode() == ViewPreview {
		m.previewNotice = "JOB ERROR: " + err.Error()
	}
	return m
}

func clearStaleJobPID(jobName string) {
	if runtime := jobRuntime(); !runtime.IsRunning(jobName) {
		_ = system.Remove(runtime.PIDPath(jobName))
	}
}

func (m Model) retryRun(item NavItem) (tea.Model, tea.Cmd) {
	if run := backgroundRunFor(item); run != nil {
		return run.Retry(m)
	}
	return m, nil
}

func (m Model) afterRetry() (tea.Model, tea.Cmd) {
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
	return m, tea.Batch(m.ensureReviewPoll(), m.ensureSyncPulse())
}

func (m Model) landPreviewOnSelection() (tea.Model, tea.Cmd) {
	m.updateScrollOffset()
	m.previewNotice = ""
	item, found := m.selectedNavItem()
	if found && item.Kind == KindGitRepo && item.GitRepo != nil {
		return m.openPreview(item)
	}
	m.resetReviewView()
	m.updatePreviewViewport()
	m.mode = ViewPreview
	cmds := []tea.Cmd{m.changesSinceReviewCmd(m.currentPRItem())}
	if found && item.Kind == KindJobDraft && item.Draft != nil && m.jobRunning(item.Draft.Name) {
		m.jobs.jobLogStamp = jobRuntime().LogStamp(item.Draft.Name, true)
		cmds = append(cmds, m.ensureJobLogRefresh())
	}
	return m, tea.Batch(cmds...)
}
