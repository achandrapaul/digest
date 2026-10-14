package store

import (
	"errors"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

func TestAddManualNoteSavesAnActiveManualNote(t *testing.T) {
	created := time.Now()
	note, err := New(t.TempDir()).AddManualNote("Call the bank\nabout the card", created)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(note.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary != "Call the bank" || loaded.Body != "about the card" || loaded.Status != model.StatusActive || loaded.Source != model.SourceManual || loaded.Created.Unix() != created.Unix() {
		t.Errorf("loaded = %+v", loaded)
	}
}

func TestAddManualNoteRefusesEmptyText(t *testing.T) {
	noteStore := New(t.TempDir())
	if _, err := noteStore.AddManualNote("  \n ", time.Now()); !errors.Is(err, ErrEmptyNote) {
		t.Fatalf("err = %v, want ErrEmptyNote", err)
	}
	if noteStore.HasNotes() {
		t.Error("an empty note was written")
	}
}
