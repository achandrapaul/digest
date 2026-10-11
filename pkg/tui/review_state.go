package tui

import (
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/tui/textarea"
)

type reviewState struct {
	reviewReports        *reviewReportMemo
	localReviews         map[string]localReviewState
	previewFindingsCount int
	reviewRuns           []review.ReviewRun
	reviewRunAction      string
	reviewRunTarget      review.PRRef
	reviewCursor         int
	reviewSelected       map[int]bool
	reviewEvent          review.Event
	reviewBody           string
	reviewNotice         string
	rejectInput          *textarea.Model
	contextCache         map[string]string
	historyRequested     map[string]bool
}
