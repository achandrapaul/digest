package tui

import "github.com/achandrapaul/digest/pkg/brag"

type bragState struct {
	bragSaved      *bragSavedMemo
	bragStates     map[string]bragRowState
	bragRuns       []brag.Run
	bragSelected   int
	bragExpanded   map[int]bool
	bragPeriod     brag.Period
	bragEntry      *brag.Brag
	bragRegenerate bool
	bragNotice     string
}
