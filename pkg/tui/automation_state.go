package tui

import "github.com/achandrapaul/digest/pkg/automation"

type automationState struct {
	automationRuns   map[string]automation.Run
	automationNoteID string
	automationName   string
	automationPhase  automation.Phase
	automationNotice string
	ghHosts          []string
}
