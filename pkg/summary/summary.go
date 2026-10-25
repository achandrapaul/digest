package summary

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/notegroup"
	"github.com/achandrapaul/digest/pkg/store"
)

type Note struct {
	ID      string    `json:"id"`
	Summary string    `json:"summary"`
	Source  string    `json:"source"`
	Status  string    `json:"status"`
	Repo    string    `json:"repo,omitempty"`
	Ref     string    `json:"ref,omitempty"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type Section struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Notes []Note `json:"notes"`
}

type Report struct {
	Date        string    `json:"date"`
	PreviousDay string    `json:"previous_day"`
	Sections    []Section `json:"sections"`
}

var exclusions = map[string][]model.Source{
	"pr": {model.SourceMyPR, model.SourcePRReview},
}

func rawSource(note *model.Note) string {
	return strings.TrimPrefix(string(cmp.Or(note.Source, model.SourceManual)), "#")
}

func reportNotes(notes []*model.Note) []Note {
	converted := make([]Note, 0, len(notes))
	for _, note := range notes {
		converted = append(converted, Note{
			ID:      note.ID,
			Summary: note.Summary,
			Source:  rawSource(note),
			Status:  string(note.Status),
			Repo:    note.Repo,
			Ref:     note.Ref,
			Created: note.Created,
			Updated: note.Updated,
		})
	}
	return converted
}

func ExcludedSources(exclude string) ([]model.Source, error) {
	var sources []model.Source
	for name := range strings.SplitSeq(exclude, ",") {
		if name = strings.TrimSpace(name); name == "" {
			continue
		}
		matched, known := exclusions[name]
		if !known {
			return nil, fmt.Errorf("unknown --exclude value %q (known: %s)", name, strings.Join(slices.Sorted(maps.Keys(exclusions)), ", "))
		}
		for _, source := range matched {
			if !slices.Contains(sources, source) {
				sources = append(sources, source)
			}
		}
	}
	return sources, nil
}

func Build(notesDir string, now time.Time, isWorkDay func(time.Weekday) bool, exclude string) (Report, error) {
	skippedSources, err := ExcludedSources(exclude)
	if err != nil {
		return Report{}, err
	}
	previousDay := notegroup.PreviousWorkDay(now, isWorkDay)
	notes, listErr := store.New(notesDir).ListDashboard(now, previousDay)
	var skipped *store.SkippedNotesError
	if listErr != nil && !errors.As(listErr, &skipped) {
		return Report{}, listErr
	}
	notes = slices.DeleteFunc(notes, func(note *model.Note) bool { return slices.Contains(skippedSources, note.Source) })
	groups := notegroup.Group(notes, now, previousDay)
	return Report{
		Date:        now.Format(time.DateOnly),
		PreviousDay: previousDay.Format(time.DateOnly),
		Sections: []Section{
			{Key: "previous_done", Title: "Done " + previousDay.Format("Monday"), Notes: reportNotes(groups.PreviousDone)},
			{Key: "pending", Title: "Pending", Notes: reportNotes(groups.Carried)},
			{Key: "today", Title: "Today", Notes: reportNotes(groups.Today)},
			{Key: "today_done", Title: "Closed today", Notes: reportNotes(groups.TodayDone)},
		},
	}, listErr
}

func WriteText(out io.Writer, report Report) {
	for sectionIndex, section := range report.Sections {
		if sectionIndex > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "%s (%d)\n", section.Title, len(section.Notes))
		if len(section.Notes) == 0 {
			fmt.Fprintln(out, "  none")
		}
		for _, note := range section.Notes {
			fmt.Fprintf(out, "  - %s #%s\n", note.Summary, note.Source)
		}
	}
}
