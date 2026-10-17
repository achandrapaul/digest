package notegroup

import (
	"slices"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

type Groups struct {
	PreviousDay  time.Time
	PreviousDone []*model.Note
	Carried      []*model.Note
	Today        []*model.Note
	TodayDone    []*model.Note
}

func StartOfDay(moment time.Time) time.Time {
	return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, moment.Location())
}

func SameDay(first, second time.Time) bool {
	firstYear, firstMonth, firstDay := first.Local().Date()
	secondYear, secondMonth, secondDay := second.Local().Date()
	return firstYear == secondYear && firstMonth == secondMonth && firstDay == secondDay
}

func PreviousWorkDay(viewedDay time.Time, isWorkDay func(time.Weekday) bool) time.Time {
	viewedDayStart := StartOfDay(viewedDay)
	for daysBack := 1; daysBack <= 7; daysBack++ {
		if candidate := viewedDayStart.AddDate(0, 0, -daysBack); isWorkDay(candidate.Weekday()) {
			return candidate
		}
	}
	return viewedDayStart.AddDate(0, 0, -1)
}

func SourceTag(n *model.Note) string {
	sourceText := string(n.Source)
	if sourceText == "" {
		sourceText = "manual"
	}
	if !strings.HasPrefix(sourceText, "#") {
		sourceText = "#" + sourceText
	}
	return sourceText
}

func closedSourceRank(n *model.Note) int {
	switch n.Source {
	case model.SourceManual, "":
		return 0
	case model.SourceMyPR:
		return 1
	case model.SourcePRReview:
		return 2
	}
	return 3
}

func Group(notes []*model.Note, viewedDay, previousDay time.Time) Groups {
	groups := Groups{PreviousDay: previousDay}
	for _, n := range notes {
		switch n.Status {
		case model.StatusArchived:
		case model.StatusDone:
			if !n.Updated.Before(previousDay) && n.Updated.Before(previousDay.AddDate(0, 0, 1)) {
				groups.PreviousDone = append(groups.PreviousDone, n)
			}
			if SameDay(n.Updated, viewedDay) {
				groups.TodayDone = append(groups.TodayDone, n)
			}
		default:
			if SameDay(n.Created, viewedDay) || (n.Created.Before(viewedDay) && n.StartedOn(viewedDay)) {
				groups.Today = append(groups.Today, n)
			} else if n.Created.Before(viewedDay) {
				groups.Carried = append(groups.Carried, n)
			}
		}
	}
	slices.SortStableFunc(groups.PreviousDone, func(a, b *model.Note) int {
		if bySource := strings.Compare(string(a.Source), string(b.Source)); bySource != 0 {
			return bySource
		}
		return a.Updated.Compare(b.Updated)
	})
	slices.SortStableFunc(groups.TodayDone, func(a, b *model.Note) int {
		if byRank := closedSourceRank(a) - closedSourceRank(b); byRank != 0 {
			return byRank
		}
		if bySource := strings.Compare(SourceTag(a), SourceTag(b)); bySource != 0 {
			return bySource
		}
		return b.Updated.Compare(a.Updated)
	})
	slices.SortStableFunc(groups.Carried, func(a, b *model.Note) int {
		if bySourceTag := strings.Compare(SourceTag(b), SourceTag(a)); bySourceTag != 0 {
			return bySourceTag
		}
		return b.Created.Compare(a.Created)
	})
	return groups
}
