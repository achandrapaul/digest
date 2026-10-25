package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/store"
	"github.com/achandrapaul/digest/pkg/summary"
)

func runSummaryCommand(cfg *config.Config, asJSON bool, exclude string, out, errOut io.Writer, now time.Time) int {
	report, err := summary.Build(cfg.NotesDir(), now, cfg.IsWorkDay, exclude)
	var skipped *store.SkippedNotesError
	if err != nil && !errors.As(err, &skipped) {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return 1
	}
	if skipped != nil {
		defer fmt.Fprintf(errOut, "Warning: %v\n", skipped)
	}
	if !asJSON {
		summary.WriteText(out, report)
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
