package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/store"
)

func TestApprovalNotesCreatedOnceWithRepoPRID(t *testing.T) {
	noteStore := store.New(t.TempDir())
	approvedAt := time.Date(2026, 10, 1, 15, 30, 0, 0, time.Local)
	existing := &model.Note{Ref: "o/control#5", Summary: "Approved: Old", Status: model.StatusDone, Source: model.SourcePRReview}
	if err := noteStore.Save(existing); err != nil {
		t.Fatal(err)
	}
	approvals := []review.ActivityPR{
		{Number: 5, Title: "Old", URL: "https://github.com/o/control/pull/5", Repository: "control", State: "APPROVED", ReviewedAt: approvedAt},
		{Number: 6, Title: "Fix date picker", URL: "https://github.com/o/control/pull/6", Repository: "control", State: "APPROVED", ReviewedAt: approvedAt},
		{Number: 6, Title: "Fix date picker", URL: "https://github.com/o/control/pull/6", Repository: "control", State: "APPROVED", ReviewedAt: approvedAt},
	}
	known := indexPRNotes([]*model.Note{existing}, "")
	for attempt := range 2 {
		msg := knownReviewNotesCmd(noteStore, known, approvals)()
		if changed, ok := msg.(notesChangedMsg); attempt == 0 && (!ok || changed.err != nil) {
			t.Fatalf("msg = %#v", msg)
		}
	}
	notes, err := noteStore.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("notes = %d, want 2", len(notes))
	}
	var created *model.Note
	for _, note := range notes {
		if note.Ref == "o/control#6" {
			created = note
		}
	}
	if created == nil {
		t.Fatalf("no o/control#6 note")
	}
	if created.ID == "control:6" || !strings.HasPrefix(filepath.Base(created.FilePath), created.ID) {
		t.Errorf("new review note should get a timestamp id that names its file: id=%q file=%q", created.ID, created.FilePath)
	}
	if created.Summary != "Approved:control:6 Fix date picker" || created.Status != model.StatusDone || created.Source != model.SourcePRReview || created.Repo != "control" {
		t.Errorf("note = %+v", created)
	}
	if !created.Updated.Equal(approvedAt) || created.Body != "https://github.com/o/control/pull/6" {
		t.Errorf("updated=%v body=%q", created.Updated, created.Body)
	}
}

func TestApprovalNotesExactIDMatch(t *testing.T) {
	noteStore := store.New(t.TempDir())
	if err := noteStore.Save(&model.Note{ID: "control:19162", Summary: "x", Status: model.StatusDone}); err != nil {
		t.Fatal(err)
	}
	approvals := []review.ActivityPR{{Number: 1, Title: "One", URL: "https://github.com/o/control/pull/1", Repository: "control", State: "APPROVED", ReviewedAt: time.Now()}}
	reviewNotesCmd(noteStore, approvals)()
	notes, _ := noteStore.List()
	if len(notes) != 2 {
		t.Errorf("control:1 was treated as existing; notes = %d", len(notes))
	}
}

func TestApprovalNotesNothingToDo(t *testing.T) {
	if cmd := reviewNotesCmd(store.New(t.TempDir()), nil); cmd != nil {
		t.Errorf("want nil cmd without approvals")
	}
}

func TestReviewNoteSummaryIncludesRepoAndNumber(t *testing.T) {
	cases := map[string]string{
		"APPROVED":          "Approved:control:6 Fix",
		"CHANGES_REQUESTED": "Reviewed: RequestedChanges:control:6 Fix",
		"COMMENTED":         "Reviewed: Commented:control:6 Fix",
	}
	for state, want := range cases {
		if got := reviewNoteSummary(state, "control", 6, "Fix"); got != want {
			t.Errorf("%s: got %q, want %q", state, got, want)
		}
	}
}

func TestReopenApprovedNotesMatchesNewAndLegacySummary(t *testing.T) {
	noteStore := store.New(t.TempDir())
	approvedAt := time.Now().Add(-time.Hour)
	approved := []*model.Note{
		{Ref: prNoteRef("", "control", 7), Summary: "Approved:control:7 New", Status: model.StatusDone, Source: model.SourcePRReview, Updated: approvedAt},
		{Ref: prNoteRef("", "control", 8), Summary: "Approved: Legacy", Status: model.StatusDone, Source: model.SourcePRReview, Updated: approvedAt},
	}
	for _, note := range approved {
		if err := noteStore.Save(note); err != nil {
			t.Fatal(err)
		}
	}
	pending := []GitPRItem{{Number: 7, Repository: "control"}, {Number: 8, Repository: "control"}}
	msg := knownReopenApprovedNotesCmd(noteStore, indexPRNotes(approved, ""), pending, time.Now())()
	loaded, ok := msg.(notesChangedMsg)
	if !ok || loaded.err != nil {
		t.Fatalf("msg = %#v", msg)
	}
	if len(loaded.notes) != 2 {
		t.Fatalf("reopened notes = %d, want 2", len(loaded.notes))
	}
	for _, note := range loaded.notes {
		if note.Status != model.StatusActive {
			t.Errorf("%s status = %s, want active", note.ID, note.Status)
		}
	}
}

func TestReviewNotesUseThePRCreatedTimeAndAnOwnerFallback(t *testing.T) {
	noteStore := store.New(t.TempDir())
	prCreated := time.Date(2026, 9, 20, 11, 0, 0, 0, time.Local)
	reviews := []review.ActivityPR{
		{Number: 3, Title: "A", URL: "https://github.com/acme/web/pull/3", Repository: "web", State: "APPROVED", ReviewedAt: time.Now(), CreatedAt: prCreated},
		{Number: 4, Title: "B", URL: "not a pr link", Repository: "web", State: "APPROVED", ReviewedAt: time.Now()},
	}
	reviewNotesCmd(noteStore, reviews)()
	notes, err := noteStore.List()
	if err != nil {
		t.Fatal(err)
	}
	refs := map[string]*model.Note{}
	for _, note := range notes {
		refs[note.Ref] = note
	}
	if note := refs["acme/web#3"]; note == nil || !note.Created.Equal(prCreated) || note.ID != store.NoteID(prCreated) {
		t.Errorf("review note should take the PR created time: %+v", note)
	}
	if refs["owner/web#4"] == nil {
		t.Errorf("unknown owner should be stored as owner: %v", refs)
	}
}
