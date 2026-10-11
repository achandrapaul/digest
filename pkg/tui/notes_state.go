package tui

import (
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/notify"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type notesState struct {
	currentNote             *model.Note
	archivedSelected        int
	archivedSelectedMap     map[int]bool
	archivedViewport        viewport.Model
	deleteTargetNotes       []*model.Note
	selectAfterReload       string
	awaitingNewNoteSave     bool
	browsedNotes            []*model.Note
	closedThisWeekElsewhere int
	missingSave             *failedNoteSave
	actionMenuNoteID        string
	actionMenuItems         []noteAction
	actionMenuSelected      int
	notifyInput             *textinput.Model
	notifyNotice            string
	notifyEntries           map[string]notify.Entry
	linkMenuItems           []string
	linkMenuSelected        int
	inlineInput             *textinput.Model
	startupNotesErr         error
}
