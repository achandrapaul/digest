package tui

import (
	"sync/atomic"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/jobs"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/system"

	tea "github.com/charmbracelet/bubbletea"
)

var digestRoot atomic.Value

func init() {
	setDigestRoot((*config.Config)(nil).Root())
}

func setDigestRoot(root string) {
	digestRoot.Store(root)
}

func currentDigestRoot() string {
	return digestRoot.Load().(string)
}

func jobRuntime() jobs.Runtime {
	return jobs.NewRuntime(currentDigestRoot())
}

var loadDryRunResult = func(jobName string) (string, int, bool) {
	return jobRuntime().LoadDryRunResult(jobName)
}

func (m Model) jobRunning(jobName string) bool {
	return m.jobs.runningJobPIDs[jobName] > 0
}

func (m Model) isAnyDryRunInFlight() bool {
	return len(m.jobs.dryRunsInFlight) > 0
}

type previewedJob struct {
	name                   string
	inFlight, hasRun       bool
	running                bool
	exitCode, outputLength int
}

func (m Model) previewedJobState() previewedJob {
	item, found := m.selectedNavItem()
	if !found || item.Kind != KindJobDraft || item.Draft == nil {
		return previewedJob{}
	}
	name := item.Draft.Name
	_, running := m.jobs.runningJobPIDs[name]
	return previewedJob{name: name, inFlight: m.jobs.dryRunsInFlight[name], hasRun: m.jobs.jobDryRunHasRun[name], running: running, exitCode: m.jobs.jobDryRunExitCodes[name], outputLength: len(m.jobs.jobDryRunOutputs[name])}
}

func (m Model) isAnyJobRunning() bool {
	return len(m.jobs.runningJobPIDs) > 0
}

type jobLaunch struct {
	cfg             *config.Config
	spec            config.JobSpec
	dryRun          bool
	stopDryRunFirst bool
	states          jobStatesQuery
}

type jobLaunchedMsg struct {
	launch        jobLaunch
	dryRunStopped bool
	err           error
	states        jobStates
}

func jobLaunchKey(jobName string, dryRun bool) string {
	if dryRun {
		return "dry-run:" + jobName
	}
	return "job:" + jobName
}

func launchJobCmd(launch jobLaunch) tea.Cmd {
	return func() tea.Msg {
		msg := jobLaunchedMsg{launch: launch}
		if launch.stopDryRunFirst {
			msg.err = stopJobDryRun(launch.spec.Name)
			msg.dryRunStopped = msg.err == nil
		}
		if msg.err == nil {
			msg.err = startJobProcess(launch)
		}
		msg.states = launch.states.load()
		return msg
	}
}

var startJobProcess = func(launch jobLaunch) error {
	return jobRuntime().Start(launch.cfg, launch.spec, launch.dryRun)
}

var stopJobDryRun = func(jobName string) error {
	return jobRuntime().StopDryRun(jobName)
}

func abortJobCmd(jobName string) tea.Cmd {
	return func() tea.Msg {
		return jobAbortedMsg{jobName: jobName, err: jobRuntime().Abort(jobName)}
	}
}

func readFileTail(path string, maxBytes int64) string {
	return review.PlainText(system.ReadTail(path, maxBytes))
}

func (m Model) jobDryRunOutputFor(jobName string) string {
	if m.jobs.jobDryRunHasRun == nil || !m.jobs.jobDryRunHasRun[jobName] {
		return ""
	}
	return m.jobs.jobDryRunOutputs[jobName]
}

func tickCtrlCResetCmd(pressSequence int) tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return ctrlCResetMsg{pressSequence: pressSequence}
	})
}

func (m Model) previewedRunningJob() string {
	if m.mode != ViewPreview {
		return ""
	}
	navItems := m.allNavItems()
	if m.selected >= len(navItems) {
		return ""
	}
	if item := navItems[m.selected]; item.Kind == KindJobDraft && item.Draft != nil {
		return item.Draft.Name
	}
	return ""
}

func (m *Model) startJobDryRunCmd(jobName string) tea.Cmd {
	if m.cfg == nil || m.jobs.dryRunsInFlight[jobName] || !m.beginLaunch(jobLaunchKey(jobName, true)) {
		return nil
	}
	return launchJobCmd(jobLaunch{cfg: m.cfg, spec: jobs.FindSpec(m.cfg, jobName), dryRun: true, states: m.jobStatesQuery(true)})
}
