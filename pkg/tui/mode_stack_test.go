package tui

import "testing"

func TestPopModeReturnsTheDashboardWhenEmpty(t *testing.T) {
	var m Model
	if got := m.popMode(); got != ViewDashboard {
		t.Fatalf("popMode on an empty stack = %v", got)
	}
}

func TestModeStackReturnsInPushOrder(t *testing.T) {
	m := Model{mode: ViewPreview}
	m.pushMode(ViewHelp)
	m.pushMode(ViewError)
	if m.mode != ViewError || m.returnMode() != ViewHelp {
		t.Fatalf("mode %v return %v", m.mode, m.returnMode())
	}
	if first, second := m.popMode(), m.popMode(); first != ViewHelp || second != ViewPreview {
		t.Fatalf("pops = %v, %v", first, second)
	}
}

func TestModeStackCopiesDoNotShareTheirPushes(t *testing.T) {
	base := Model{mode: ViewHelp, modeStack: append(make([]ViewMode, 0, 8), ViewPreview)}
	fromHelp, fromArchive := base, base
	fromHelp.mode = ViewHelp
	fromHelp.pushMode(ViewError)
	fromArchive.mode = ViewArchived
	fromArchive.pushMode(ViewError)
	if got := fromHelp.popMode(); got != ViewHelp {
		t.Fatalf("a sibling copy overwrote the stack: %v", got)
	}
	if got := base.returnMode(); got != ViewPreview || len(base.modeStack) != 1 {
		t.Fatalf("the original stack changed: %v", base.modeStack)
	}
}
