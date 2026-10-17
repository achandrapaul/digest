package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/notegroup"
	"github.com/achandrapaul/digest/pkg/store"
)

type summaryNote struct {
	ID      string    `json:"id"`
	Summary string    `json:"summary"`
	Source  string    `json:"source"`
	Status  string    `json:"status"`
	Repo    string    `json:"repo,omitempty"`
	Ref     string    `json:"ref,omitempty"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type summarySection struct {
	Key   string        `json:"key"`
	Title string        `json:"title"`
	Notes []summaryNote `json:"notes"`
}

type summaryReport struct {
	Date        string           `json:"date"`
	PreviousDay string           `json:"previous_day"`
	Sections    []summarySection `json:"sections"`
}

func summaryNotes(notes []*model.Note) []summaryNote {
	converted := make([]summaryNote, 0, len(notes))
	for _, n := range notes {
		converted = append(converted, summaryNote{
			ID:      n.ID,
			Summary: n.Summary,
			Source:  notegroup.SourceTag(n),
			Status:  string(n.Status),
			Repo:    n.Repo,
			Ref:     n.Ref,
			Created: n.Created,
			Updated: n.Updated,
		})
	}
	return converted
}

func buildSummary(cfg *config.Config, now time.Time) (summaryReport, error) {
	previousDay := notegroup.PreviousWorkDay(now, cfg.IsWorkDay)
	notes, err := store.New(cfg.NotesDir()).ListDashboard(now, previousDay)
	if err != nil {
		return summaryReport{}, err
	}
	groups := notegroup.Group(notes, now, previousDay)
	return summaryReport{
		Date:        now.Format(time.DateOnly),
		PreviousDay: previousDay.Format(time.DateOnly),
		Sections: []summarySection{
			{Key: "previous_done", Title: "Done " + previousDay.Format("Monday"), Notes: summaryNotes(groups.PreviousDone)},
			{Key: "pending", Title: "Pending", Notes: summaryNotes(groups.Carried)},
			{Key: "today", Title: "Today", Notes: summaryNotes(groups.Today)},
			{Key: "today_done", Title: "Closed today", Notes: summaryNotes(groups.TodayDone)},
		},
	}, nil
}

func writeSummaryText(out io.Writer, report summaryReport) {
	for sectionIndex, section := range report.Sections {
		if sectionIndex > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "%s (%d)\n", section.Title, len(section.Notes))
		if len(section.Notes) == 0 {
			fmt.Fprintln(out, "  none")
		}
		for _, note := range section.Notes {
			fmt.Fprintf(out, "  - %s %s\n", note.Summary, note.Source)
		}
	}
}

func runSummaryCommand(cfg *config.Config, asJSON bool, out, errOut io.Writer, now time.Time) int {
	report, err := buildSummary(cfg, now)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return 1
	}
	if !asJSON {
		writeSummaryText(out, report)
		return 0
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return 1
	}
	return 0
}
