package summary

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/store"
)

func weekdaysOnly(weekday time.Weekday) bool {
	return weekday != time.Saturday && weekday != time.Sunday
}

func TestExcludedSources(t *testing.T) {
	prSources := []model.Source{model.SourceMyPR, model.SourcePRReview}
	cases := map[string]struct {
		exclude string
		want    []model.Source
		fails   bool
	}{
		"empty":            {exclude: "", want: nil},
		"only commas":      {exclude: " , ,", want: nil},
		"pr":               {exclude: "pr", want: prSources},
		"spaces":           {exclude: " pr , ", want: prSources},
		"repeated":         {exclude: "pr,pr", want: prSources},
		"unknown":          {exclude: "bogus", fails: true},
		"unknown after pr": {exclude: "pr,bogus", fails: true},
	}
	for name, testCase := range cases {
		got, err := ExcludedSources(testCase.exclude)
		if testCase.fails {
			if err == nil || !strings.Contains(err.Error(), "known: pr") {
				t.Errorf("%s: err = %v, want an unknown value error listing pr", name, err)
			}
			continue
		}
		if err != nil || !slices.Equal(got, testCase.want) {
			t.Errorf("%s: got %v, %v; want %v", name, got, err, testCase.want)
		}
	}
}

func saveNotes(t *testing.T, notesDir string, notes ...*model.Note) {
	t.Helper()
	noteStore := store.New(notesDir)
	for _, note := range notes {
		if err := noteStore.Save(note); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBuildPutsTheRawSourceInTheReportAndTheTagInTheText(t *testing.T) {
	notesDir := t.TempDir()
	monday := time.Date(2026, 10, 19, 12, 0, 0, 0, time.Local)
	friday := time.Date(2026, 10, 16, 15, 0, 0, 0, time.Local)
	saveNotes(t, notesDir,
		&model.Note{Summary: "friday work", Status: model.StatusDone, Source: model.SourceManual, Created: friday, Updated: friday},
		&model.Note{Summary: "Own:OPEN:o:r:my pr", Status: model.StatusActive, Source: model.SourceMyPR, Created: monday, Updated: monday},
	)
	report, err := Build(notesDir, monday, weekdaysOnly, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.PreviousDay != "2026-10-16" || report.Sections[0].Title != "Done Friday" || len(report.Sections[0].Notes) != 1 {
		t.Fatalf("previous done section = %+v", report.Sections[0])
	}
	if source := report.Sections[2].Notes[0].Source; source != "my-pr" {
		t.Errorf("json source = %q, want the raw model source", source)
	}
	var text strings.Builder
	WriteText(&text, report)
	if !strings.Contains(text.String(), "Done Friday (1)\n  - friday work #manual") || !strings.Contains(text.String(), "  - Own:OPEN:o:r:my pr #my-pr") {
		t.Errorf("text =\n%s", text.String())
	}
}

func TestBuildKeepsTheReadableNotesWhenOneFileIsCorrupt(t *testing.T) {
	notesDir := t.TempDir()
	monday := time.Date(2026, 10, 19, 12, 0, 0, 0, time.Local)
	saveNotes(t, notesDir, &model.Note{Summary: "today item", Status: model.StatusActive, Source: model.SourceManual, Created: monday, Updated: monday})
	corruptPath := filepath.Join(notesDir, "2026", "10", "broken.md")
	if err := os.MkdirAll(filepath.Dir(corruptPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corruptPath, []byte("---\nnot: [valid\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Build(notesDir, monday, weekdaysOnly, "")
	var skipped *store.SkippedNotesError
	if !errors.As(err, &skipped) {
		t.Fatalf("err = %v, want *store.SkippedNotesError", err)
	}
	if len(report.Sections) != 4 || len(report.Sections[2].Notes) != 1 || report.Sections[2].Notes[0].Summary != "today item" {
		t.Errorf("report should keep the readable notes: %+v", report)
	}
}
