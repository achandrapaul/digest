package tui

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/jobs"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/tui/textarea"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	bragClock    = time.Now
	startBragRun = brag.StartBackground
)

type bragRow struct {
	year   int
	period brag.Period
}

func (r bragRow) title() string {
	switch period := r.period.(type) {
	case nil:
		return fmt.Sprint(r.year)
	case brag.Week:
		return fmt.Sprintf("Week %02d · %s", period.Number, period.Days())
	default:
		return period.Label()
	}
}

func (m Model) bragRoot() string {
	return m.cfg.BragDir()
}

func (m Model) firstNoteTime() time.Time {
	if m.appStateKnown && !m.appState.FirstNoteCreated.IsZero() {
		return m.appState.FirstNoteCreated
	}
	var first time.Time
	for _, note := range m.notesForBrowsing() {
		if !note.Created.IsZero() && (first.IsZero() || note.Created.Before(first)) {
			first = note.Created
		}
	}
	return first
}

func (m Model) bragYearExpanded(year int) bool {
	if expanded, ok := m.brag.bragExpanded[year]; ok {
		return expanded
	}
	return year == bragClock().Year()
}

func (m Model) bragRows() []bragRow {
	now := bragClock()
	first := m.firstNoteTime()
	weeks := brag.WeeksSince(first, now)
	weeksByMonth := map[string][]brag.Week{}
	for _, week := range weeks {
		monthID := brag.MonthOfWeek(week).ID()
		weeksByMonth[monthID] = append(weeksByMonth[monthID], week)
	}
	newestMonth := brag.MonthOfWeek(weeks[0])
	oldestMonth := brag.MonthOfWeek(weeks[len(weeks)-1])
	if !first.IsZero() {
		oldestMonth = brag.MonthOf(first.In(now.Location()))
	}
	var rows []bragRow
	for month := newestMonth; !month.Start.Before(oldestMonth.Start); month = month.Previous() {
		year := month.Start.Year()
		if len(rows) == 0 || rows[len(rows)-1].year != year {
			rows = append(rows, bragRow{year: year})
			if m.bragYearExpanded(year) {
				rows = append(rows, bragRow{year: year, period: brag.YearOf(month.Start)})
			}
		}
		if !m.bragYearExpanded(year) {
			continue
		}
		rows = append(rows, bragRow{year: year, period: month})
		for _, week := range weeksByMonth[month.ID()] {
			rows = append(rows, bragRow{year: year, period: week})
		}
	}
	return rows
}

type bragRowState struct {
	label      string
	failed     bool
	unreadable bool
}

func (m Model) bragRowStateFor(period brag.Period) bragRowState {
	if state, cached := m.brag.bragStates[period.ID()]; cached {
		return state
	}
	state := m.readBragRowState(period)
	if m.brag.bragStates != nil {
		m.brag.bragStates[period.ID()] = state
	}
	return state
}

func (m Model) bragStateLabel(row bragRow) string {
	if row.period == nil {
		return ""
	}
	return m.bragRowStateFor(row.period).label
}

func (m Model) readBragRowState(period brag.Period) bragRowState {
	root := m.bragRoot()
	switch {
	case brag.IsRunning(root, period.ID()):
		return bragRowState{label: "Bragging..."}
	case brag.Exists(root, period):
		if _, err := brag.Load(root, period); err != nil {
			return bragRowState{label: fmt.Sprintf("Brag about this %s again?", period.Kind()), unreadable: true}
		}
		return bragRowState{label: "View your brag"}
	}
	failed := brag.Braggable(period, bragClock()) && brag.Status(root, period.ID()) == brag.RunFailed
	return bragRowState{label: fmt.Sprintf("Brag about this %s?", period.Kind()), failed: failed}
}

func (m Model) selectedBragRow() (bragRow, bool) {
	rows := m.bragRows()
	if m.brag.bragCursor.selected < 0 || m.brag.bragCursor.selected >= len(rows) {
		return bragRow{}, false
	}
	return rows[m.brag.bragCursor.selected], true
}

func (m *Model) refreshBragRuns() {
	m.brag.bragStates = nil
	m.applyBragRuns(brag.ListRuns(m.bragRoot()))
}

func (m *Model) applyBragRuns(runs []brag.Run) {
	unchanged := m.brag.bragStates != nil && slices.EqualFunc(m.brag.bragRuns, runs, func(previous, current brag.Run) bool {
		return previous.Meta == current.Meta && previous.Status == current.Status && previous.StartedAt.Equal(current.StartedAt)
	})
	m.brag.bragRuns = runs
	if unchanged {
		return
	}
	m.brag.bragStates = make(map[string]bragRowState)
	m.brag.bragSaved = &bragSavedMemo{}
}

type bragSavedMemo struct {
	weekStart time.Time
	saved     bool
}

func (m Model) weekBragged(week brag.Week) bool {
	if m.brag.bragSaved == nil {
		return brag.Exists(m.bragRoot(), week)
	}
	if !m.brag.bragSaved.weekStart.Equal(week.Start) {
		m.brag.bragSaved.weekStart, m.brag.bragSaved.saved = week.Start, brag.Exists(m.bragRoot(), week)
	}
	return m.brag.bragSaved.saved
}

func (m Model) anyBragRunning() bool {
	for _, run := range m.brag.bragRuns {
		if run.Status == brag.RunRunning {
			return true
		}
	}
	return false
}

func (m Model) openBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = ViewBragList
	m.brag.bragNotice = ""
	if m.unbraggedWeekNotice() != "" {
		m.selectBragWeek(brag.WeekOf(bragClock()).Previous())
	}
	m.brag.bragCursor.Clamp(len(m.bragRows()))
	if m.appStateKnown && !m.appState.FirstNoteCreated.IsZero() {
		return m, nil
	}
	return m, m.browseNotesCmd()
}

func (m *Model) selectBragWeek(week brag.Week) {
	if year := brag.MonthOfWeek(week).Start.Year(); !m.bragYearExpanded(year) {
		expanded := maps.Clone(m.brag.bragExpanded)
		if expanded == nil {
			expanded = map[int]bool{}
		}
		expanded[year] = true
		m.brag.bragExpanded = expanded
	}
	for index, row := range m.bragRows() {
		if row.period != nil && row.period.ID() == week.ID() {
			m.brag.bragCursor.selected = index
			return
		}
	}
}

func (m Model) closeBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = ViewDashboard
	m.brag.bragNotice = ""
	return m, nil
}

func (m Model) bragCursorDown(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.brag.bragCursor.Move(1, len(m.bragRows()))
	return m, nil
}

func (m Model) bragCursorUp(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.brag.bragCursor.Move(-1, len(m.bragRows()))
	return m, nil
}

func (m Model) bragListEnter(tea.KeyMsg) (tea.Model, tea.Cmd) {
	row, ok := m.selectedBragRow()
	if !ok {
		return m, nil
	}
	m.brag.bragNotice = ""
	if row.period == nil {
		expanded := make(map[int]bool, len(m.brag.bragExpanded)+1)
		for year, value := range m.brag.bragExpanded {
			expanded[year] = value
		}
		expanded[row.year] = !m.bragYearExpanded(row.year)
		m.brag.bragExpanded = expanded
		return m, nil
	}
	root := m.bragRoot()
	switch {
	case brag.IsRunning(root, row.period.ID()):
		m.brag.bragNotice = fmt.Sprintf("Already bragging about %s — see Jobs", row.period.Label())
		return m, nil
	case brag.Exists(root, row.period):
		if _, err := brag.Load(root, row.period); err != nil {
			m.brag.bragNotice = err.Error()
			m.brag.bragPeriod, m.brag.bragRegenerate = row.period, false
			m.pushMode(ViewBragConfirm)
			return m, nil
		}
		return m.showBragView(row.period)
	case !brag.Braggable(row.period, bragClock()):
		_, end := row.period.Range()
		m.brag.bragNotice = fmt.Sprintf("%s isn't over yet — brag about it from %s", row.title(), end.Format("Mon 02 Jan"))
		return m, nil
	}
	m.brag.bragPeriod, m.brag.bragRegenerate = row.period, false
	m.pushMode(ViewBragConfirm)
	return m, nil
}

func (m Model) bragModalSize() (int, int) {
	return modalWidthFor(m.width), max(12, m.height*90/100)
}

func (m *Model) loadBragView() error {
	entry, err := brag.Load(m.bragRoot(), m.brag.bragPeriod)
	if err != nil {
		return err
	}
	m.showBragEntry(entry)
	return nil
}

func (m *Model) showBragEntry(entry *brag.Brag) {
	m.brag.bragEntry = entry
	modalWidth, modalHeight := m.bragModalSize()
	innerWidth := modalWidth - 6
	content := renderMarkdown(entry.Body(), innerWidth)
	m.previewViewport = viewport.New(innerWidth, max(3, modalHeight-10))
	m.previewViewport.SetContent(content)
}

func (m *Model) reloadBragViewIfChanged() {
	if m.brag.bragPeriod == nil || m.brag.bragEntry == nil {
		return
	}
	latest, err := brag.Load(m.bragRoot(), m.brag.bragPeriod)
	if err != nil || (latest.Updated.Equal(m.brag.bragEntry.Updated) && latest.SummarizedAt.Equal(m.brag.bragEntry.SummarizedAt)) {
		return
	}
	if err := m.loadBragView(); err == nil {
		m.brag.bragNotice = "Summary updated"
	}
}

func (m Model) showBragView(period brag.Period) (tea.Model, tea.Cmd) {
	m.brag.bragPeriod = period
	if err := m.loadBragView(); err != nil {
		m.brag.bragNotice = err.Error()
		return m, nil
	}
	m.mode = ViewBragView
	return m, nil
}

func (m Model) closeBragView(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = ViewBragList
	m.brag.bragNotice = ""
	return m, nil
}

func (m Model) editBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.brag.bragEntry == nil {
		return m, nil
	}
	m.brag.bragNotice = ""
	m.mode = ViewBragEdit
	m.replaceEditorText(m.brag.bragEntry.Body())
	m.editor.Focus()
	startEditorAtTop(m.editor)
	return m, textarea.Blink
}

func (m Model) saveBragEdit(tea.KeyMsg) (tea.Model, tea.Cmd) {
	parsed, err := brag.ParseBody(m.brag.bragPeriod, m.editor.Value())
	if err != nil {
		m.brag.bragNotice = err.Error()
		return m, nil
	}
	now := time.Now()
	parsed.Created, parsed.SummarizedAt, parsed.Updated = m.brag.bragEntry.Created, m.brag.bragEntry.SummarizedAt, now
	if parsed.Created.IsZero() {
		parsed.Created = now
	}
	if !m.beginLaunch(bragEditLaunchKey) {
		return m, nil
	}
	root, edited := m.bragRoot(), *parsed
	return m, func() tea.Msg {
		msg := bragEditSavedMsg{period: edited.Period, saveErr: edited.Save(root)}
		if msg.saveErr == nil || errors.Is(msg.saveErr, system.ErrLock) {
			msg.entry, msg.loadErr = brag.Load(root, edited.Period)
		}
		return msg
	}
}

const bragEditLaunchKey = "brag-edit"

type bragEditSavedMsg struct {
	period  brag.Period
	saveErr error
	entry   *brag.Brag
	loadErr error
}

func (m Model) applyBragEditSaved(msg bragEditSavedMsg) (tea.Model, tea.Cmd) {
	m.finishLaunch(bragEditLaunchKey)
	if msg.saveErr != nil && !errors.Is(msg.saveErr, system.ErrLock) {
		m.brag.bragNotice = msg.saveErr.Error()
		return m, nil
	}
	if m.mode != ViewBragEdit || m.brag.bragPeriod.ID() != msg.period.ID() {
		return m, nil
	}
	m.editor.Blur()
	switch {
	case msg.loadErr != nil:
		m.brag.bragNotice = msg.loadErr.Error()
	case msg.saveErr != nil:
		m.showBragEntry(msg.entry)
		m.brag.bragNotice = "Saved but not locked"
	default:
		m.showBragEntry(msg.entry)
		m.brag.bragNotice = "Saved"
	}
	m.mode = ViewBragView
	return m, nil
}

func (m Model) cancelBragEdit(tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editorChanged() {
		return m.beginDiscardConfirm(), nil
	}
	m.editor.Blur()
	m.brag.bragNotice = ""
	m.mode = ViewBragView
	return m, nil
}

func (m Model) copyBragEditor(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.copyText(strings.TrimSpace(m.editor.Value()))
	return m, nil
}

func (m Model) copyBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.brag.bragEntry != nil {
		m.copyText(strings.TrimSpace(m.brag.bragEntry.Body()))
	}
	return m, nil
}

func (m Model) bragAgain(tea.KeyMsg) (tea.Model, tea.Cmd) {
	if brag.IsRunning(m.bragRoot(), m.brag.bragPeriod.ID()) {
		m.brag.bragNotice = "Already bragging — see Jobs"
		return m, nil
	}
	m.brag.bragRegenerate = true
	m.pushMode(ViewBragConfirm)
	return m, nil
}

type bragStartedMsg struct {
	period brag.Period
	retry  bool
	err    error
}

func bragLaunchKey(period brag.Period) string {
	return "brag:" + period.ID()
}

func (m *Model) startBragCmd(period brag.Period, regenerate, retry bool) tea.Cmd {
	if !m.beginLaunch(bragLaunchKey(period)) {
		return nil
	}
	root := m.bragRoot()
	return func() tea.Msg {
		return bragStartedMsg{period: period, retry: retry, err: startBragRun(root, period, regenerate)}
	}
}

func (m Model) confirmBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = m.popMode()
	return m, m.startBragCmd(m.brag.bragPeriod, m.brag.bragRegenerate, false)
}

func (m Model) applyBragStarted(msg bragStartedMsg) (tea.Model, tea.Cmd) {
	m.finishLaunch(bragLaunchKey(msg.period))
	if msg.retry {
		if msg.err != nil {
			m.showError("BRAG ERROR", msg.err)
			return m, nil
		}
		m.refreshBragRuns()
		return m.afterRetry()
	}
	if msg.err != nil {
		if errors.Is(msg.err, brag.ErrBragRunning) {
			m.brag.bragNotice = "Already bragging — see Jobs"
		} else {
			m.brag.bragNotice = msg.err.Error()
		}
		return m, nil
	}
	m.brag.bragNotice = fmt.Sprintf("Bragging about %s in the background", msg.period.Label())
	m.refreshBragRuns()
	return m, tea.Batch(m.ensureReviewPoll(), m.ensureSyncPulse())
}

func (m Model) cancelBrag(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = m.popMode()
	return m, nil
}

func (m Model) unbraggedWeekNotice() string {
	now := bragClock()
	lastWeek := brag.WeekOf(now).Previous()
	first := m.firstNoteTime()
	if first.IsZero() || brag.WeekOf(first.In(now.Location())).Start.After(lastWeek.Start) {
		return ""
	}
	if m.weekBragged(lastWeek) {
		return ""
	}
	return fmt.Sprintf("Last week (W%02d) isn't bragged — press b", lastWeek.Number)
}

var (
	bragYearStyle  = lipgloss.NewStyle().Foreground(colourBlue).Bold(true)
	bragMonthStyle = lipgloss.NewStyle().Foreground(colourMauve).Bold(true)
	bragDoneStyle  = lipgloss.NewStyle().Foreground(colourGreen)
)

func (m Model) renderBragRow(row bragRow, selected bool, width int) string {
	indent, titleStyle := "     ", itemStyle
	switch row.period.(type) {
	case nil:
		marker := "▸"
		if m.bragYearExpanded(row.year) {
			marker = "▾"
		}
		indent, titleStyle = bragYearStyle.Render(marker+" "), bragYearStyle
	case brag.Month, brag.Year:
		indent, titleStyle = "   ", bragMonthStyle
	}
	if selected {
		titleStyle = titleStyle.Bold(true)
	}
	left := indent + underlinedWhen(selected, titleStyle.Render(row.title()))
	return alignRight(left, []string{m.renderBragStatus(row, selected)}, width, false)
}

func (m Model) renderBragStatus(row bragRow, selected bool) string {
	label := m.bragStateLabel(row)
	if label == "" {
		return ""
	}
	return underlinedWhen(selected, m.bragStatusCell(row, label))
}

func (m Model) bragStatusCell(row bragRow, label string) string {
	if label == "Bragging..." {
		return m.renderPulseIndicator(label)
	}
	state := m.bragRowStateFor(row.period)
	failed := state.failed
	switch {
	case state.unreadable:
		return yellowBadgeStyle.Render(label) + stateStyle(review.StateFailed).Render(" · unreadable")
	case label == "View your brag":
		return bragDoneStyle.Render(label)
	case !brag.Braggable(row.period, bragClock()):
		return mutedStyle.Render(label)
	case failed:
		return yellowBadgeStyle.Render(label) + stateStyle(review.StateFailed).Render(" · last run failed")
	default:
		return yellowBadgeStyle.Render(label)
	}
}

func (m Model) renderBragList() string {
	modalWidth, modalHeight := m.bragModalSize()
	innerWidth := modalWidth - 6
	title := modalTitleStyle.Render(" BRAG ")
	footer := renderModalFooter(footerItemsFrom(bragListBindings()), innerWidth)
	below := []string{""}
	if m.brag.bragNotice != "" {
		below = append(below, yellowBadgeStyle.Render(m.brag.bragNotice), "")
	}
	below = append(below, footer)
	fixed := lipgloss.Height(title) + 1 + lipgloss.Height(strings.Join(below, "\n"))
	rowsAvailable := max(1, modalHeight-modalStyle.GetVerticalFrameSize()-fixed)
	rows := m.bragRows()
	first, last := visibleGitRows(m.brag.bragCursor.selected, len(rows), rowsAvailable)
	var lines []string
	for index := first; index < last; index++ {
		lines = append(lines, m.renderBragRow(rows[index], index == m.brag.bragCursor.selected, innerWidth))
	}
	for len(lines) < rowsAvailable {
		lines = append(lines, "")
	}
	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, "", strings.Join(lines, "\n")}, below...)...)
	return m.placeBragModal(content, modalWidth)
}

func (m Model) placeBragModal(content string, modalWidth int) string {
	return m.framedPopup(content, modalWidth)
}

func (m Model) renderBragView() string {
	modalWidth, modalHeight := m.bragModalSize()
	innerWidth := modalWidth - 6
	title := modalTitleStyle.Render(fmt.Sprintf(" BRAG: %s ", strings.ToUpper(m.brag.bragPeriod.Label())))
	if m.bragRowStateFor(m.brag.bragPeriod).label == "Bragging..." {
		title += "  " + m.renderPulseIndicator("Bragging...")
	}
	below := []string{""}
	if m.brag.bragNotice != "" {
		below = append(below, yellowBadgeStyle.Render(m.brag.bragNotice), "")
	}
	below = append(below, renderModalFooter(footerItemsFrom(bragViewBindings()), innerWidth))
	fixed := lipgloss.Height(title) + 1 + lipgloss.Height(strings.Join(below, "\n"))
	m.previewViewport.Height = max(3, modalHeight-modalStyle.GetVerticalFrameSize()-fixed)
	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, "", m.previewViewport.View()}, below...)...)
	return m.placeBragModal(content, modalWidth)
}

func (m Model) renderBragEdit() string {
	modalWidth, _ := m.bragModalSize()
	innerWidth := modalWidth - 6
	title := modalTitleStyle.Render(fmt.Sprintf(" EDIT BRAG: %s ", strings.ToUpper(m.brag.bragPeriod.Label())))
	parts := []string{title, "", m.editorView(), ""}
	if m.brag.bragNotice != "" {
		parts = append(parts, yellowBadgeStyle.Render(m.brag.bragNotice), "")
	}
	parts = append(parts, renderModalFooter(footerItemsFrom(bragEditBindings()), innerWidth))
	return m.placeBragModal(lipgloss.JoinVertical(lipgloss.Left, parts...), modalWidth)
}

func (m Model) renderBragConfirm(modalWidth int) string {
	prompt := fmt.Sprintf("Brag about %s?\n\nFacts are collected and Claude writes the summary in the background.", m.brag.bragPeriod.Label())
	if m.brag.bragRegenerate {
		prompt = fmt.Sprintf("Brag again about %s?\n\nClaude rewrites only the summary from the saved facts.", m.brag.bragPeriod.Label())
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render(" BRAG "),
		"",
		prompt,
		"",
		renderModalFooter(footerItemsFrom(bragConfirmBindings()), modalWidth-6),
	)
	return m.placeBragModal(content, modalWidth)
}

func bragRunLabel(run brag.Run) string {
	return "brag " + run.Meta.ID
}

func bragRunPreview(root string, run brag.Run) runPreview {
	status := "FAILED"
	if run.Status == brag.RunRunning {
		status = "RUNNING"
	}
	logText := strings.TrimSpace(readFileTail(filepath.Join(brag.StateDir(root, run.Meta.ID), brag.RunLogFile), jobs.LogTailBytes))
	if logText == "" {
		logText = "(no log output yet)"
	}
	return runPreview{heading: fmt.Sprintf("# Brag: %s (%s)", run.Meta.ID, status), log: logText}
}

var stopBragRun = brag.Stop

func (m Model) stopRunningBragRuns() (tea.Model, tea.Cmd) {
	root := m.bragRoot()
	var failures []error
	for _, run := range brag.ListRuns(root) {
		if run.Status != brag.RunRunning {
			continue
		}
		if err := stopBragRun(root, run.Meta.ID); err != nil {
			failures = append(failures, err)
		}
	}
	m.showError("BRAG ERROR", failures...)
	selectedKey, selectedOccurrence := m.selectedNavKey()
	m.refreshBragRuns()
	m.restoreSelection(selectedKey, selectedOccurrence)
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
	m.clampScreenSelection()
	return m, nil
}

func (m Model) stopOrDismissBragRun(run *brag.Run) (tea.Model, tea.Cmd) {
	root := m.bragRoot()
	var err error
	if brag.IsRunning(root, run.Meta.ID) {
		err = stopBragRun(root, run.Meta.ID)
	} else {
		err = brag.Dismiss(root, run.Meta.ID)
	}
	if err != nil {
		m.showError("BRAG ERROR", err)
		return m, nil
	}
	selectedKey, selectedOccurrence := m.selectedNavKey()
	m.refreshBragRuns()
	m.restoreSelection(selectedKey, selectedOccurrence)
	if m.mode == ViewPreview {
		if _, ok := m.selectedNavItem(); ok {
			m.updatePreviewViewport()
		} else {
			m.mode = ViewDashboard
		}
	}
	return m, nil
}
