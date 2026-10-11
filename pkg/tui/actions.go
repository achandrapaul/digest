package tui

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/notify"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	actionNameNotify    = "notify"
	actionNameNotifyOff = "notify:off"
)

var typedIntervalPattern = regexp.MustCompile(`^[0-9]*[mhd]?$`)

type noteAction struct {
	name       string
	automation bool
}

type notifyEntriesMsg struct {
	entries map[string]notify.Entry
	err     error
}

func noteIsActive(n *model.Note) bool {
	return n != nil && n.ID != "" && n.Status != model.StatusDone && n.Status != model.StatusArchived
}

func (m Model) actionTargetNote() (*model.Note, bool) {
	note, found := m.previewNote()
	return note, found && noteIsActive(note)
}

func (m Model) noteActions(n *model.Note) []noteAction {
	if !noteIsActive(n) {
		return nil
	}
	_, reminding := m.notesUI.notifyEntries[n.ID]
	var actions []noteAction
	if !reminding {
		actions = append(actions, noteAction{name: actionNameNotify})
	}
	if !m.automatedKindKnown(n.Automated) {
		for _, spec := range automation.MatchAll(m.cfg.AutomationList(), n) {
			actions = append(actions, noteAction{name: spec.Name, automation: true})
		}
	}
	savedOrder := m.appState.ActionMenuOrder
	rank := func(name string) int {
		if position := slices.Index(savedOrder, name); position >= 0 {
			return position
		}
		return len(savedOrder)
	}
	slices.SortStableFunc(actions, func(left, right noteAction) int { return rank(left.name) - rank(right.name) })
	if reminding {
		actions = slices.Insert(actions, 0, noteAction{name: actionNameNotifyOff}, noteAction{name: actionNameNotify})
	}
	return actions
}

func (m Model) openActionMenu(tea.KeyMsg) (tea.Model, tea.Cmd) {
	note, found := m.actionTargetNote()
	if !found {
		return m, nil
	}
	m.notesUI.actionMenuNoteID = note.ID
	m.notesUI.actionMenuItems = m.noteActions(note)
	m.notesUI.actionMenuCursor.selected = 0
	m.pushMode(ViewActionMenu)
	return m, nil
}

func (m Model) actionMenuDown(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notesUI.actionMenuCursor.Move(1, len(m.notesUI.actionMenuItems))
	return m, nil
}

func (m Model) actionMenuUp(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notesUI.actionMenuCursor.Move(-1, len(m.notesUI.actionMenuItems))
	return m, nil
}

func (m Model) closeActionMenu(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = m.popMode()
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
	return m, nil
}

func (m Model) chooseAction(tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.notesUI.actionMenuCursor.selected >= len(m.notesUI.actionMenuItems) {
		return m, nil
	}
	chosen := m.notesUI.actionMenuItems[m.notesUI.actionMenuCursor.selected]
	if chosen.name == actionNameNotifyOff {
		return m.removeReminder()
	}
	switch {
	case !chosen.automation:
		m.notesUI.notifyNotice = ""
		m.notesUI.notifyInput.SetValue(m.notesUI.notifyEntries[m.notesUI.actionMenuNoteID].Interval)
		m.notesUI.notifyInput.CursorEnd()
		m.notesUI.notifyInput.Focus()
		m.mode = ViewNotifyInput
	default:
		m.mode = m.popMode()
		m = m.askAutomationConfirm(m.notesUI.actionMenuNoteID, chosen.name, automation.PhaseDraft)
	}
	return m, nil
}

func (m *Model) leaveNotifyInput() {
	m.notesUI.notifyInput.Blur()
	m.mode = m.popMode()
	if m.mode == ViewPreview {
		m.updatePreviewViewport()
	}
}

const messageSourceNotify = "notify"

func (m Model) confirmNotify(tea.KeyMsg) (tea.Model, tea.Cmd) {
	interval, err := notify.ParseInterval(m.notesUI.notifyInput.Value())
	if err != nil {
		m.notesUI.notifyNotice = err.Error()
		return m, nil
	}
	note := m.noteByID(m.notesUI.actionMenuNoteID)
	if note == nil {
		m.notesUI.notifyInput.Blur()
		m.mode = ViewDashboard
		m.postMessage(messageSourceNotify, messageError, "note no longer exists")
		return m, nil
	}
	entry := notify.Entry{NoteID: note.ID, Summary: note.Summary, Interval: notify.IntervalLabel(interval), NotifiedAt: time.Now(), OpenURL: notePullRequestURL(note)}
	if entry.OpenURL == "" {
		entry.Terminal = os.Getenv("__CFBundleIdentifier")
	}
	m.notesUI.notifyNotice = ""
	root := m.cfg.Root()
	return m, func() tea.Msg {
		if err := notify.Save(root, entry); err != nil {
			return reminderSavedMsg{noteID: entry.NoteID, err: err}
		}
		return reminderSavedMsg{noteID: entry.NoteID, listed: listNotifyEntries(root)}
	}
}

type reminderSavedMsg struct {
	noteID string
	listed notifyEntriesMsg
	err    error
}

func (m Model) applyReminderSaved(msg reminderSavedMsg) (tea.Model, tea.Cmd) {
	inputOpen := m.mode == ViewNotifyInput && m.notesUI.actionMenuNoteID == msg.noteID
	if msg.err != nil {
		if inputOpen {
			m.notesUI.notifyNotice = msg.err.Error()
		} else {
			m.showError("NOTIFY ERROR", msg.err)
		}
		return m, nil
	}
	if msg.listed.err != nil {
		if inputOpen {
			m.leaveNotifyInput()
		}
		m.showError("NOTIFY ERROR", msg.listed.err)
		return m, nil
	}
	m.notesUI.notifyEntries = msg.listed.entries
	if inputOpen {
		m.leaveNotifyInput()
	}
	return m, nil
}

func (m Model) removeReminder() (tea.Model, tea.Cmd) {
	noteID, root := m.notesUI.actionMenuNoteID, m.cfg.Root()
	entries := maps.Clone(m.notesUI.notifyEntries)
	delete(entries, noteID)
	m.notesUI.notifyEntries = entries
	m.leaveNotifyInput()
	return m, func() tea.Msg {
		if err := notify.Remove(root, noteID); err != nil {
			return notifyEntriesMsg{err: err}
		}
		return listNotifyEntries(root)
	}
}

func notifyInputAccepts(current string, msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return true
	}
	candidate := current + strings.ToLower(string(msg.Runes))
	if !typedIntervalPattern.MatchString(candidate) || strings.ContainsAny(candidate[:1], "mhd") {
		return false
	}
	amount, err := strconv.Atoi(strings.TrimRight(candidate, "mhd"))
	if err != nil || amount <= 0 {
		return false
	}
	if strings.ContainsAny(candidate, "mhd") {
		_, err := notify.ParseInterval(candidate)
		return err == nil
	}
	return time.Duration(amount)*time.Minute <= notify.MaxInterval
}

func (m Model) notifyInfoPill() string {
	return renderHintPill([]keyHint{{icon: "ⓘ", label: "1m–3d · whole numbers · m h d"}})
}

func (m Model) cancelNotify(tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.leaveNotifyInput()
	return m, nil
}

func (m Model) noteByID(noteID string) *model.Note {
	for _, note := range m.notes {
		if note.ID == noteID {
			return note
		}
	}
	return nil
}

func listNotifyEntries(root string) notifyEntriesMsg {
	entries, err := notify.List(root)
	byNoteID := make(map[string]notify.Entry, len(entries))
	for _, entry := range entries {
		byNoteID[entry.NoteID] = entry
	}
	return notifyEntriesMsg{entries: byNoteID, err: err}
}

func refreshNotifyCmd(root string, notes []*model.Note) tea.Cmd {
	activeNoteIDs := map[string]bool{}
	for _, note := range notes {
		if noteIsActive(note) {
			activeNoteIDs[note.ID] = true
		}
	}
	return func() tea.Msg {
		entries, err := notify.List(root)
		if err != nil {
			return notifyEntriesMsg{err: err}
		}
		for _, entry := range entries {
			if !activeNoteIDs[entry.NoteID] {
				if err := notify.Remove(root, entry.NoteID); err != nil {
					return notifyEntriesMsg{err: err}
				}
			}
		}
		return listNotifyEntries(root)
	}
}

func (m Model) notifyTag(n *model.Note) string {
	entry, found := m.notesUI.notifyEntries[n.ID]
	if !found || !noteIsActive(n) {
		return ""
	}
	return yellowBadgeStyle.Render("@notify:" + entry.Interval)
}

func (m Model) renderActionDropdown() string {
	labelWidth := 0
	for _, action := range m.notesUI.actionMenuItems {
		labelWidth = max(labelWidth, lipgloss.Width("@"+action.name))
	}
	var rows []string
	for index, action := range m.notesUI.actionMenuItems {
		label := fmt.Sprintf("%-*s", labelWidth, "@"+action.name)
		if index == m.notesUI.actionMenuCursor.selected {
			rows = append(rows, hintKeyStyle.Render("› "+label))
		} else {
			rows = append(rows, hintTextStyle.Render("  "+label))
		}
	}
	return hintPillStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderActionMenu() string {
	dropdown := m.renderActionDropdown()
	return m.overlayUnderSelectedRow(dropdown, m.rightAlignedColumn(dropdown))
}

func actionMenuBindings() []keyBinding {
	return []keyBinding{
		newKeyBinding(actionChooseAction, []string{"enter", "y", "Y"}, "y|enter", "choose"),
		newKeyBinding(actionActionMenuDown, []string{"j", "down"}, "j|k", "nav"),
		hiddenKeyBinding(actionActionMenuUp, "k", "up"),
		newKeyBinding(actionCloseActionMenu, []string{"esc", "n", "N"}, "esc", "cancel"),
	}
}

func notifyInputBindings() []keyBinding {
	return []keyBinding{
		newKeyBinding(actionConfirmNotify, []string{"enter", "y", "Y"}, "y|enter", "confirm"),
		newKeyBinding(actionCancelNotify, []string{"esc"}, "esc", "cancel"),
	}
}
