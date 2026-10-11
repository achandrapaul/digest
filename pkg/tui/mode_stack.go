package tui

func (m *Model) pushMode(next ViewMode) {
	m.modeStack = append(m.modeStack[:len(m.modeStack):len(m.modeStack)], m.mode)
	m.mode = next
}

func (m *Model) popMode() ViewMode {
	if len(m.modeStack) == 0 {
		return ViewDashboard
	}
	top := len(m.modeStack) - 1
	returnMode := m.modeStack[top]
	m.modeStack = m.modeStack[:top:top]
	return returnMode
}

func (m Model) returnMode() ViewMode {
	if len(m.modeStack) == 0 {
		return ViewDashboard
	}
	return m.modeStack[len(m.modeStack)-1]
}
