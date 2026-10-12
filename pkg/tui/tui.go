package tui

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/jobs"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/sourcecontrol"
	"github.com/achandrapaul/digest/pkg/store"
	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/system/clipboard"
	"github.com/achandrapaul/digest/pkg/system/process"
	"github.com/achandrapaul/digest/pkg/tui/textarea"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewEdit
	ViewSearch
	ViewDeleteConfirm
	ViewPreview
	ViewInlineEdit
	ViewGitDetails
	ViewArchived
	ViewError
	ViewReviewConfirm
	ViewReviewRunConfirm
	ViewRejectComment
	ViewHelp
	ViewBragList
	ViewBragView
	ViewBragConfirm
	ViewBragEdit
	ViewAutomationEdit
	ViewAutomationConfirm
	ViewActionMenu
	ViewLinkMenu
	ViewNotifyInput
	ViewSetup
	ViewSetupDiscard
	ViewRecreateConfirm
	ViewRecreateRow
)

type NavItemKind int

const (
	KindGitRepo NavItemKind = iota
	KindYesterdayDone
	KindTodayNote
	KindTodayDone
	KindPendingGit
	KindCarriedNote
	KindJobDraft
	KindReviewRun
	KindBragRun
	KindAutomationRun
	KindMyPR
)

type GitPRItem = sourcecontrol.PRItem

type GitRepoStat struct {
	Name     string
	Path     string
	Commits  int
	Reviewed int
	Assigned int
	Items    []GitPRItem
}

type NavItem struct {
	Kind          NavItemKind
	Note          *model.Note
	GitRepo       *GitRepoStat
	Draft         *JobDraft
	PendingGitPR  *GitPRItem
	ReviewRun     *review.ReviewRun
	BragRun       *brag.Run
	AutomationRun *automation.Run
	MyPR          *review.QueuedPR
}

type bannerWaveTickMsg struct{}
type syncPulseTickMsg struct{}
type ctrlCResetMsg struct{ pressSequence int }
type localReviewState struct {
	status         review.RunStatus
	finishedAt     time.Time
	finished       bool
	pid            int
	recommendation string
	cloned         bool
	logTail        string
}

func safeRepeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}

var openURL = func(url string) error {
	if url == "" {
		return nil
	}
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	_, err := startReaped(exec.Command(opener, url))
	return err
}

func (m *Model) openLink(url string) {
	if err := openURL(url); err != nil {
		m.showError("BROWSER", err)
	}
}

func (m *Model) copyText(text string) {
	if text == "" {
		return
	}
	if err := copyToClipboard(text); err != nil {
		m.postMessage(messageSourceClipboard, messageError, err.Error())
		return
	}
	m.postMessage(messageSourceClipboard, messageSuccess, "copied")
}

func startReaped(cmd *exec.Cmd) (<-chan error, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return exited, nil
}

var copyToClipboard = clipboard.Copy

func daysAgo(from, to time.Time) int {
	y1, m1, d1 := from.Local().Date()
	y2, m2, d2 := to.Local().Date()
	t1 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.Local)
	t2 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.Local)
	days := int(t2.Sub(t1).Hours() / 24)
	if days < 1 {
		days = 1
	}
	return days
}

type Model struct {
	git             gitState
	notesUI         notesState
	review          reviewState
	brag            bragState
	automation      automationState
	jobs            jobsState
	mode            ViewMode
	modeStack       []ViewMode
	launchesPending map[string]bool
	cfg             *config.Config
	store           *store.NoteStore
	notes           []*model.Note

	sessionCtx         context.Context
	appState           appstate.State
	appStateKnown      bool
	cancelSession      context.CancelFunc
	searchCache        *searchMemo
	contentVersion     int
	scrollPending      bool
	frames             *dashboardFrameCache
	selected           int
	scrollOffset       int
	ctrlCCount         int
	ctrlCPressSequence int
	currentDate        time.Time

	bragRunToStop       *brag.Run
	automationRunToStop *automation.Run
	runToRetry          *NavItem
	draftToDelete       string
	discardingEdit      bool
	previewNotice       string
	editorStartText     string
	editorNotice        string
	confirmTitle        string
	confirmPrompt       string
	restoreOnConfirm    bool

	syncPulseFrame   int
	bannerWaveActive bool
	bannerWaveFrame  int

	syncPulseRunning bool
	previewPollStamp previewStamp

	initialSelectionPending bool
	tagCells                *rowTagCellCache
	noteRows                *noteRowCache
	popupMemo               *framedPopupMemo
	hintGeneration          int
	hintVisible             bool
	editorRevision          int
	editorCache             *editorViewCache
	prActionFromDashboard   bool
	setup                   *setupState
	setupInput              *textinput.Model
	configPath              string

	previewViewport  viewport.Model
	previewPlainText string

	editor      *textarea.Model
	searchInput *textinput.Model

	searchCursor     listCursor
	searchPreviewID  string
	searchPreviewing bool
	searchNotice     string

	previewTab    int
	reviewPolling bool

	errorTitle           string
	messages             []appMessage
	messageExpiryPending bool
	errorLines           []string
	quitOnErrorDismiss   bool
	messageLogExpanded   bool
	messageLogViewport   viewport.Model
	screenError          string
	screenErrorMode      ViewMode

	width  int
	height int
}

func NewModel(cfg *config.Config, startupErr error) Model {
	ti := textinput.New()
	ti.Placeholder = "Search notes… (tag:pr-review, date:7d, date:2w, date:3m, date:24-12-2026)"

	ii := textinput.New()
	ii.Prompt = ""

	notifyInput := textinput.New()
	notifyInput.Prompt = "@notify:"
	notifyInput.Placeholder = "1h"
	notifyInput.CharLimit = 16
	notifyInput.Width = lipgloss.Width(notifyInput.Prompt + notifyInput.Placeholder)

	ta := textarea.New()
	ta.Placeholder = "First line: Summary\n\nRemaining lines: Body..."
	ta.ShowLineNumbers = false
	ta.MaxHeight = 0
	ta.Cursor.SetMode(cursor.CursorStatic)

	textStyle := lipgloss.NewStyle().Foreground(colourText).UnsetBackground()
	placeholderStyle := lipgloss.NewStyle().UnsetBackground()

	ta.FocusedStyle.Base = textStyle
	ta.FocusedStyle.Text = textStyle
	ta.FocusedStyle.Placeholder = placeholderStyle
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLineNumber = lipgloss.NewStyle()
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle()
	ta.FocusedStyle.EndOfBuffer = lipgloss.NewStyle()

	ta.BlurredStyle = ta.FocusedStyle
	disableTextareaDeleteShortcuts(&ta)
	for _, input := range []*textinput.Model{&ti, &ii, &notifyInput} {
		disableInputDeleteShortcuts(input)
	}

	storePath := cfg.NotesDir()
	setDigestRoot(cfg.Root())

	rejectArea := textarea.New()
	rejectArea.Placeholder = "Explain what needs to change..."
	rejectArea.ShowLineNumbers = false
	rejectArea.MaxHeight = 0
	rejectArea.FocusedStyle = ta.FocusedStyle
	rejectArea.BlurredStyle = ta.FocusedStyle
	disableTextareaDeleteShortcuts(&rejectArea)

	m := Model{
		initialSelectionPending: true,
		mode:                    ViewDashboard,
		cfg:                     cfg,
		store:                   store.New(storePath),
		searchInput:             &ti,
		frames:                  newDashboardFrameCache(),
		noteRows:                newNoteRowCache(),
		popupMemo:               &framedPopupMemo{},
		editor:                  &ta,
		editorCache:             &editorViewCache{},
		searchCache:             &searchMemo{},
		review:                  reviewState{rejectInput: &rejectArea, reviewSelected: make(map[int]bool), contextCache: make(map[string]string), reviewReports: &reviewReportMemo{}},
		notesUI:                 notesState{inlineInput: &ii, notifyInput: &notifyInput, archivedSelectedMap: make(map[int]bool)},
		jobs:                    jobsState{jobDryRunOutputs: make(map[string]string), jobDryRunExitCodes: make(map[string]int), jobDryRunHasRun: make(map[string]bool)},
		git:                     gitState{loadingCommits: cfg != nil && cfg.DailyCommitsEnabled()},
		currentDate:             time.Now(),
		bannerWaveActive:        true,
	}
	m.sessionCtx, m.cancelSession = context.WithCancel(context.Background())
	m.notes, m.notesUI.startupNotesErr = m.store.ListDashboard(m.currentDate, m.previousNoteDay())
	m.loadAppState()
	m.notesUI.notifyEntries = listNotifyEntries(cfg.Root()).entries
	m.git.fetchedPreviousDay = m.previousNoteDay()
	m.git.loadingGit = false
	if cfg.GitEnabled() {
		m.loadGitOnStartup()
	}
	m.refreshDryRunResults()
	m.refreshReviewRuns()
	m.git.commitsCtx, m.git.commitsCancel = context.WithCancel(context.Background())
	m.refreshBragRuns()
	m.applyAutomationRuns(automation.ListRuns(m.automationRoot()))
	m.reviewPolling = m.anyReviewRunning() || m.anyBragRunning() || m.anyAutomationRunning()
	m.syncPulseRunning = m.headerAnimating()
	m.jobs.runStatePolling = m.isAnyDryRunInFlight() || m.isAnyJobRunning()
	if startupErr != nil {
		m.showError("CONFIG ERROR", startupErr)
	}
	if cfg != nil && len(cfg.UnknownKeys) > 0 {
		m.postMessage("config.yaml", messageError, "unknown keys "+strings.Join(cfg.UnknownKeys, ", ")+" — run digest doctor")
	}
	return m
}

type rootRelockedMsg struct {
	err error
}

func relockRootCmd(root string) tea.Cmd {
	if !system.Protected(root) {
		return nil
	}
	return func() tea.Msg {
		_, err := system.LockTree(root)
		return rootRelockedMsg{err: err}
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.startupNotesCmd(), tickBannerWaveCmd(), m.loadCommitsCmd(), m.startupHintCmd(), relockRootCmd(m.cfg.Root())}
	if m.syncPulseRunning {
		cmds = append(cmds, tickSyncPulseCmd())
	}
	if m.git.syncOnLoad {
		cmds = append(cmds, m.gitFetchCmd())
	} else if m.git.loadingMyPRs {
		cmds = append(cmds, m.myPRsFetchCmd())
	}
	if m.jobs.runStatePolling {
		cmds = append(cmds, m.tickRunStatePollCmd())
	}
	if m.reviewPolling {
		cmds = append(cmds, m.tickReviewPollCmd())
	}
	return tea.Batch(cmds...)
}

func (m *Model) showError(title string, errs ...error) {
	var lines []string
	for _, err := range errs {
		if err != nil {
			lines = append(lines, err.Error())
		}
	}
	if len(lines) == 0 {
		return
	}
	m.postMessage(title, messageError, strings.Join(lines, "\n"))
}

func navItemKey(item NavItem) string {
	return kindHandlerFor(item.Kind).key(item)
}

func (m Model) selectedNavKey() (string, int) {
	items := m.allNavItems()
	if m.selected < 0 || m.selected >= len(items) {
		return "", 0
	}
	key := navItemKey(items[m.selected])
	occurrence := 0
	for _, item := range items[:m.selected] {
		if navItemKey(item) == key {
			occurrence++
		}
	}
	return key, occurrence
}

var launchKinds = map[string][]NavItemKind{
	config.SelectionNotesToday:     {KindTodayNote, KindCarriedNote},
	config.SelectionNotesYesterday: {KindYesterdayDone},
}

func (m *Model) selectLaunchItem() {
	items := m.allNavItems()
	for _, preferredKind := range launchKinds[m.cfg.SelectionDefault] {
		if index := slices.IndexFunc(items, func(item NavItem) bool { return item.Kind == preferredKind }); index >= 0 {
			m.selected = index
			return
		}
	}
	m.selected = 0
}

func (m *Model) selectNoteByID(noteID string) {
	items := m.allNavItems()
	if index := slices.IndexFunc(items, func(item NavItem) bool { return item.Note != nil && item.Note.ID == noteID }); index >= 0 {
		m.selected = index
	}
}

func (m *Model) restoreSelection(key string, occurrence int) {
	items := m.allNavItems()
	if key != "" {
		seen := 0
		for index, item := range items {
			if navItemKey(item) != key {
				continue
			}
			if seen == occurrence {
				m.selected = index
				return
			}
			seen++
		}
	}
	if m.selected >= len(items) {
		m.selected = max(len(items)-1, 0)
	}
}

func (m Model) allNavItems() []NavItem {
	data := m.dashboardData()
	items := make([]NavItem, 0, data.jobsEnd)
	items = notesSection{}.appendNavItems(m, items, data)
	items = gitSection{}.appendNavItems(m, items, data)
	return jobsSection{}.appendNavItems(m, items, data)
}

func (m *Model) updatePreviewViewport() {
	item, found := m.selectedNavItem()
	if !found {
		return
	}
	_, innerWidth, innerHeight := previewModalSize(m.width, m.height)
	kindHandlerFor(item.Kind).preview(m, item, innerWidth, innerHeight)
}

func (m *Model) setPreviewContent(item NavItem, mdContent string, innerWidth, innerHeight int) {
	m.previewViewport = viewport.New(innerWidth, innerHeight)
	m.previewViewport.SetContent(mdContent)
	m.previewPlainText = plainText(mdContent)
	if kindHandlerFor(item.Kind).followsLog {
		m.previewViewport.GotoBottom()
	}
}

func (m *Model) jobPreviewContent(draft *JobDraft, output jobOutput, width int) string {
	running := m.jobRunning(draft.Name)
	switch {
	case !running && draft.DryRunInFlight:
		return renderMarkdown(fmt.Sprintf("# Job: %s (DRY RUN IN PROGRESS)\n\nDry run in progress...", draft.Name), width)
	case output.text != "":
		statusHeader := "FINISHED / LOG OUTPUT"
		if running {
			statusHeader = "LIVE EXECUTION LOG"
		} else if output.isDryRun {
			statusHeader = "DRY RUN ANALYSIS"
		}
		return runPreview{heading: fmt.Sprintf("# Job: %s (%s)", draft.Name, statusHeader), log: output.text}.render(width)
	}
	keyGuide := "(r) run"
	if jobs.HasDryRun(draft.DryRunCommand) {
		keyGuide = "(d) dry-run " + keyGuide
	}
	return renderMarkdown(fmt.Sprintf("# Job: %s\n\nNo dry-run analysis or log output available.\n%s", draft.Name, keyGuide), width)
}

func (m *Model) showJobPreview(output jobOutput) {
	item, found := m.selectedNavItem()
	if !found || item.Kind != KindJobDraft || item.Draft == nil {
		return
	}
	_, innerWidth, innerHeight := previewModalSize(m.width, m.height)
	m.setPreviewContent(item, m.jobPreviewContent(item.Draft, output, innerWidth), innerWidth, innerHeight)
}

type runPreview struct {
	heading string
	log     string
}

func (preview runPreview) render(width int) string {
	logTail := lipgloss.NewStyle().Width(width).Render(lastLines(strings.TrimSpace(preview.log), jobLogTailLines))
	return renderMarkdown(preview.heading, width) + "\n\n" + logTail
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !isAnimationTick(msg) {
		m.contentVersion++
	}
	next, cmd := m.handleMsg(msg)
	updated, isModel := next.(Model)
	if !isModel {
		return next, cmd
	}
	updated.clearScreenErrorOffScreen()
	if updated.mode == ViewDashboard {
		updated.modeStack = nil
	}
	if updated.scrollPending {
		updated.settleScroll()
	}
	if updated.notesUI.browsedNotes != nil && !updated.browsing() {
		updated.notesUI.browsedNotes = nil
		cmd = tea.Batch(cmd, updated.closedThisWeekCmd())
	}
	if updated.git.prAlertsDue {
		updated.git.prAlertsDue = false
		cmd = tea.Batch(cmd, updated.prAlertsCmd())
	}
	if !updated.messageExpiryPending {
		if expiry := updated.messageExpiryCmd(); expiry != nil {
			updated.messageExpiryPending = true
			cmd = tea.Batch(cmd, expiry)
		}
	}
	return updated, cmd
}

func isAnimationTick(msg tea.Msg) bool {
	switch msg.(type) {
	case bannerWaveTickMsg, syncPulseTickMsg:
		return true
	}
	return false
}

type messageApplier func(Model, tea.Msg) (tea.Model, tea.Cmd, bool)

var messageAppliers []messageApplier

func init() {
	messageAppliers = []messageApplier{
		headerSection{}.ApplyMessage,
		notesSection{}.ApplyMessage,
		gitSection{}.ApplyMessage,
		jobsSection{}.ApplyMessage,
	}
}

func messageHandled(next tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd, bool) {
	return next, cmd, true
}

func (m Model) handleMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	for _, applyMessage := range messageAppliers {
		if next, cmd, handled := applyMessage(m, msg); handled {
			return next, cmd
		}
	}
	switch msg := msg.(type) {
	case searchExportedMsg:
		return m.handleSearchExported(msg)

	case ctrlCResetMsg:
		if msg.pressSequence == m.ctrlCPressSequence {
			m.ctrlCCount = 0
		}
		return m, nil

	case setupSavedMsg:
		return m.applySetupSaved(msg)

	case spinner.TickMsg:
		return m.tickSetupSpinner(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		_, editorWidth, editorHeight := previewModalSize(msg.Width, msg.Height)
		m.editor.SetWidth(editorWidth)
		m.editor.SetHeight(editorHeight)
		if m.mode == ViewInlineEdit && m.notesUI.currentNote != nil {
			m.notesUI.inlineInput.Width = m.inlineEditWidth(m.notesUI.currentNote)
		}
		m.updateScrollOffset()
		if m.mode == ViewPreview {
			m.updatePreviewViewport()
		}

	case loadNotesMsg:
		notesSection{}.storeLoadedNotes(&m, msg)
		refetchPreviousDay := gitSection{}.refreshPreviousDay(&m)
		return notesSection{}.finishLoadedNotes(m, msg, refetchPreviousDay)

	case tea.KeyMsg:
		next, cmd := m.handleKey(msg)
		updated, isModel := next.(Model)
		if !isModel {
			return next, cmd
		}
		return updated, tea.Batch(cmd, updated.restartHintTimer())

	case hintIdleMsg:
		m.hintVisible = msg.generation == m.hintGeneration && m.mode == ViewDashboard && m.keyHintPill() != ""
		return m, nil
	}

	return m, nil
}

func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Local().Date()
	y2, m2, d2 := t2.Local().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func (m Model) renderSubSection(title string, count int, showCount bool, isActive bool) string {
	baseStyle := subSectionStyle
	if isActive {
		baseStyle = baseStyle.Underline(true)
	}
	if showCount {
		countStr := mutedStyle.Render(fmt.Sprintf("(%d)", count))
		return fmt.Sprintf("%s %s", baseStyle.Render("/ "+title), countStr)
	}
	return baseStyle.Render("/ " + title)
}

func (m Model) openPreview(item NavItem) (tea.Model, tea.Cmd) {
	m.previewNotice = ""
	if item.Kind == KindGitRepo && item.GitRepo != nil {
		m.git.gitPopupRepo = item.GitRepo
		m.git.gitPopupTab = 0
		m.git.gitPopupCursor.selected = 0
		m.mode = ViewGitDetails
		return m, nil
	}
	m.searchPreviewing = false
	m.resetReviewView()
	m.updatePreviewViewport()
	m.mode = ViewPreview
	if item.PendingGitPR != nil {
		cmds := []tea.Cmd{m.relatedHistoryCmd(item.PendingGitPR), m.changesSinceReviewCmd(item.PendingGitPR)}
		if m.anyReviewRunning() {
			cmds = append(cmds, m.ensureReviewPoll())
		}
		return m, tea.Batch(cmds...)
	}
	if item.Kind == KindJobDraft && item.Draft != nil && m.jobRunning(item.Draft.Name) {
		m.jobs.jobLogStamp = jobRuntime().LogStamp(item.Draft.Name, true)
		return m, m.ensureJobLogRefresh()
	}
	return m, nil
}

const sectionGap = "\n\n"

var dashboardContentBuilder = Model.buildDashboardContent

func (m Model) dashboardContent() (content string, selectedLine int) {
	return dashboardContentBuilder(m)
}

func (m Model) buildDashboardContent() (content string, selectedLine int) {
	builder := &dashboardBuilder{selected: m.selected, innerWidth: m.width - 4, rowWidth: m.width - 3}
	if m.frames != nil {
		builder.board.Grow(m.frames.lastContentLength + 1024)
	}
	m.tagCells = &rowTagCellCache{prs: map[*GitPRItem][3]string{}}
	data := m.dashboardData()

	notesSection{}.Render(m, builder, data)
	if m.cfg.GitEnabled() {
		gitSection{}.Render(m, builder, data)
	}
	jobsSection{}.Render(m, builder, data)

	if m.frames != nil {
		m.frames.lastContentLength = builder.board.Len()
	}
	return builder.board.String(), builder.selectedLine
}

func (m Model) visibleScrollOffset(frame *dashboardFrame) int {
	bodyHeight := m.frameBodyHeight(frame)
	lineCount := strings.Count(frame.content, "\n") + 1
	scrollOffset := max(m.scrollOffset, 0)
	if scrollOffset > lineCount-bodyHeight && lineCount > bodyHeight {
		scrollOffset = lineCount - bodyHeight
	}
	return scrollOffset
}

func (m Model) renderFrameBody(frame *dashboardFrame) string {
	bodyHeight := m.frameBodyHeight(frame)
	lines := strings.Split(frame.content, "\n")
	scrollOffset := m.visibleScrollOffset(frame)

	endIdx := scrollOffset + bodyHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	visibleLines := lines[scrollOffset:endIdx]
	for i := len(visibleLines); i < bodyHeight; i++ {
		visibleLines = append(visibleLines, "")
	}

	var framed []string
	for _, l := range visibleLines[:bodyHeight] {
		pad := m.width - lipgloss.Width(l) - 2
		framed = append(framed, fmt.Sprintf("│%s%s│", l, safeRepeat(" ", pad)))
	}

	return strings.Join(framed, "\n")
}

func Run(cfg *config.Config, startupErr error, configPath string, startInSetup bool) error {
	model := NewModel(cfg, startupErr)
	model.configPath = configPath
	if startInSetup {
		model = model.startSetup(configPath)
	}
	model.requireAppState()
	release, err := process.Mark(cfg.TUIMarkerPath())
	if err != nil {
		return err
	}
	defer release()
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

var noteLinkPattern = regexp.MustCompile(`https?://[^\s<>()"']+`)
var pullRequestPathPattern = regexp.MustCompile(`/[^/\s]+/pull/\d+`)

func prReviewNoteURL(note *model.Note) string {
	if note == nil || note.Source != model.SourcePRReview {
		return ""
	}
	return notePullRequestURL(note)
}

func notePullRequestURL(note *model.Note) string {
	links := noteLinkPattern.FindAllString(note.Body, -1)
	for i := len(links) - 1; i >= 0; i-- {
		if pullRequestPathPattern.MatchString(links[i]) {
			return links[i]
		}
	}
	return ""
}
