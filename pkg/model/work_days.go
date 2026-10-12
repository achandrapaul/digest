package model

import (
	"slices"
	"time"
)

const WorkDayLayout = "02-01-2006"

func parseWorkDay(value string) (time.Time, bool) {
	day, err := time.ParseInLocation(WorkDayLayout, value, time.Local)
	return day, err == nil
}

func workDayText(day time.Time) string {
	return day.Local().Format(WorkDayLayout)
}

func (n *Note) StartedOn(day time.Time) bool {
	for index := len(n.StartedWorkingOn) - 1; index >= 0; index-- {
		if _, readable := parseWorkDay(n.StartedWorkingOn[index]); readable {
			return n.StartedWorkingOn[index] == workDayText(day)
		}
	}
	return false
}

func (n *Note) WorkedOnDays() []time.Time {
	var days []time.Time
	if !n.Created.IsZero() {
		created := n.Created.Local()
		days = append(days, time.Date(created.Year(), created.Month(), created.Day(), 0, 0, 0, 0, time.Local))
	}
	for _, value := range n.StartedWorkingOn {
		if day, readable := parseWorkDay(value); readable {
			days = append(days, day)
		}
	}
	slices.SortFunc(days, func(left, right time.Time) int { return left.Compare(right) })
	return slices.CompactFunc(days, func(left, right time.Time) bool { return left.Equal(right) })
}

func (n *Note) InvalidWorkDates() []string {
	var invalid []string
	for _, value := range n.StartedWorkingOn {
		if _, readable := parseWorkDay(value); !readable {
			invalid = append(invalid, value)
		}
	}
	return invalid
}

func (n *Note) ToggleWorkingOn(day time.Time) bool {
	text := workDayText(day)
	if !slices.Contains(n.StartedWorkingOn, text) {
		n.StartedWorkingOn = append(n.StartedWorkingOn, text)
		return true
	}
	n.StartedWorkingOn = slices.DeleteFunc(n.StartedWorkingOn, func(value string) bool { return value == text })
	if len(n.StartedWorkingOn) == 0 {
		n.StartedWorkingOn = nil
	}
	return false
}

func (n *Note) RecordWorkedOn(day time.Time) {
	if text := workDayText(day); !slices.Contains(n.StartedWorkingOn, text) {
		n.StartedWorkingOn = append(n.StartedWorkingOn, text)
	}
}
