package tui

import "maps"

type jobsState struct {
	dryRunLogStamps       map[string]string
	jobDryRunOutputs      map[string]string
	jobDryRunExitCodes    map[string]int
	jobDryRunHasRun       map[string]bool
	jobLogRunning         bool
	jobLogStamp           string
	runStatePolling       bool
	runningJobPIDs        map[string]int
	dryRunsInFlight       map[string]bool
	previewJobLogFinished bool
	jobToExecute          string
	jobToAbort            string
	jobRunStopsDryRun     bool
	jobDryRunToStart      string
	staleJobToClear       string
	jobConfirmError       string
}

type dryRunResult struct {
	stamp    string
	output   string
	exitCode int
	found    bool
}

type jobStates struct {
	runningPIDs map[string]int
	inFlight    map[string]bool
	dryRuns     map[string]dryRunResult
	loaded      bool
}

type jobStatesQuery struct {
	jobNames    []string
	knownStamps map[string]string
	dryRuns     bool
}

func (m Model) jobStatesQuery(withDryRuns bool) jobStatesQuery {
	query := jobStatesQuery{dryRuns: withDryRuns && m.cfg != nil}
	if m.cfg != nil {
		for _, job := range m.cfg.JobList() {
			query.jobNames = append(query.jobNames, job.Name)
		}
	}
	if query.dryRuns {
		query.knownStamps = maps.Clone(m.jobs.dryRunLogStamps)
	}
	return query
}

func (query jobStatesQuery) load() jobStates {
	states := jobStates{runningPIDs: map[string]int{}, inFlight: map[string]bool{}, loaded: true}
	for _, jobName := range query.jobNames {
		if pid, running := runningJobPID(jobName); running {
			states.runningPIDs[jobName] = pid
		}
		if isDryRunInFlight(jobName) {
			states.inFlight[jobName] = true
		}
	}
	if !query.dryRuns {
		return states
	}
	states.dryRuns = map[string]dryRunResult{}
	for _, jobName := range query.jobNames {
		if states.inFlight[jobName] {
			continue
		}
		stamp := dryRunStamp(jobName)
		if previous, seen := query.knownStamps[jobName]; seen && previous == stamp {
			continue
		}
		output, exitCode, found := loadDryRunResult(jobName)
		states.dryRuns[jobName] = dryRunResult{stamp: stamp, output: output, exitCode: exitCode, found: found}
	}
	return states
}

func (m *Model) applyJobStates(states jobStates) {
	if !states.loaded {
		return
	}
	m.jobs.runningJobPIDs, m.jobs.dryRunsInFlight = states.runningPIDs, states.inFlight
	if states.dryRuns == nil {
		return
	}
	outputs, exitCodes, hasRun, stamps := maps.Clone(m.jobs.jobDryRunOutputs), maps.Clone(m.jobs.jobDryRunExitCodes), maps.Clone(m.jobs.jobDryRunHasRun), maps.Clone(m.jobs.dryRunLogStamps)
	if outputs == nil {
		outputs, exitCodes, hasRun = map[string]string{}, map[string]int{}, map[string]bool{}
	}
	if stamps == nil {
		stamps = map[string]string{}
	}
	for jobName, result := range states.dryRuns {
		stamps[jobName] = result.stamp
		if !result.found {
			delete(outputs, jobName)
			delete(exitCodes, jobName)
			delete(hasRun, jobName)
			continue
		}
		outputs[jobName], exitCodes[jobName], hasRun[jobName] = result.output, result.exitCode, true
	}
	m.jobs.jobDryRunOutputs, m.jobs.jobDryRunExitCodes, m.jobs.jobDryRunHasRun, m.jobs.dryRunLogStamps = outputs, exitCodes, hasRun, stamps
}

func (m *Model) refreshJobStates() {
	m.applyJobStates(m.jobStatesQuery(false).load())
}

func (m *Model) refreshDryRunResults() {
	m.applyJobStates(m.jobStatesQuery(true).load())
}

type jobOutput struct {
	text     string
	isDryRun bool
}

func loadJobOutput(jobName, dryRunOutput string) jobOutput {
	text, isDryRun := latestJobOutput(jobName, dryRunOutput)
	return jobOutput{text: text, isDryRun: isDryRun}
}
