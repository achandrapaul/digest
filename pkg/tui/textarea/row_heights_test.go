package textarea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func expectedHeights(m Model) []int {
	expected := make([]int, len(m.value))
	for row, runes := range m.value {
		expected[row] = len(wrap(runes, m.width))
	}
	return expected
}

func assertHeightsMatch(t *testing.T, step string, m Model) {
	t.Helper()
	actual := m.heights()
	expected := expectedHeights(m)
	if len(actual) != len(expected) {
		t.Fatalf("%s: %d heights for %d rows", step, len(actual), len(expected))
	}
	for row := range expected {
		if actual[row] != expected[row] {
			t.Fatalf("%s: row %d height %d, want %d", step, row, actual[row], expected[row])
		}
	}
}

func sendKey(m Model, keyType tea.KeyType) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated
}

func TestRowHeightsFollowEveryKindOfEdit(t *testing.T) {
	m := newTextArea()
	m.SetWidth(30)
	m.Focus()
	m.SetValue(strings.Repeat("word ", 30) + "\nshort\n" + strings.Repeat("x", 80))
	assertHeightsMatch(t, "set value", m)

	m.moveToBegin()
	m = sendString(m, strings.Repeat("abc ", 20))
	assertHeightsMatch(t, "typing grows a row", m)

	m = sendKey(m, tea.KeyEnter)
	assertHeightsMatch(t, "enter splits a row", m)

	m = sendKey(m, tea.KeyBackspace)
	assertHeightsMatch(t, "backspace merges rows", m)

	m.moveToBegin()
	m.CursorEnd()
	m = sendKey(m, tea.KeyDelete)
	assertHeightsMatch(t, "delete merges the row below", m)

	m.InsertString("first\nsecond line that is long enough to wrap around the edge\nthird")
	assertHeightsMatch(t, "multi-line paste", m)

	m.SetWidth(12)
	assertHeightsMatch(t, "narrower width", m)
	m.SetWidth(80)
	assertHeightsMatch(t, "wider width", m)

	m.SetValue("one\ntwo")
	assertHeightsMatch(t, "set value replaces", m)
	m.Reset()
	assertHeightsMatch(t, "reset", m)
	m.SetValue(strings.Repeat("y", 100))
	assertHeightsMatch(t, "set value after reset", m)
}

func TestSingleRowReplacementWithSameRowCountRefreshesHeights(t *testing.T) {
	m := newTextArea()
	m.SetWidth(20)
	m.SetValue(strings.Repeat("long ", 20))
	assertHeightsMatch(t, "long row", m)
	m.SetValue("tiny")
	assertHeightsMatch(t, "replaced by a tiny row", m)
	if got := m.heights()[0]; got != 1 {
		t.Errorf("tiny row height = %d, want 1", got)
	}
}

func TestShortRowsSkipWrappingButKeepTheirHeight(t *testing.T) {
	m := newTextArea()
	m.SetWidth(40)
	for _, text := range []string{"", "a", "short line", strings.Repeat("w", 18), strings.Repeat("w", 19), strings.Repeat("w", 20), strings.Repeat("宽", 10), strings.Repeat("宽", 25)} {
		if got, want := m.heightOf([]rune(text)), len(wrap([]rune(text), m.width)); got != want {
			t.Errorf("%q: height %d, want %d", text, got, want)
		}
	}
}
