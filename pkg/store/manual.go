package store

import (
	"errors"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

var ErrEmptyNote = errors.New("note is empty")

func (s *NoteStore) AddManualNote(text string, created time.Time) (*model.Note, error) {
	summary, body := model.SplitNoteText(text)
	if summary == "" {
		return nil, ErrEmptyNote
	}
	note := &model.Note{Status: model.StatusActive, Source: model.SourceManual, Created: created, Summary: summary, Body: body}
	return note, s.Save(note)
}
