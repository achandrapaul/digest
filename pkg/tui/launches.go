package tui

import "maps"

func (m *Model) beginLaunch(key string) bool {
	if m.launchesPending[key] {
		return false
	}
	pending := maps.Clone(m.launchesPending)
	if pending == nil {
		pending = map[string]bool{}
	}
	pending[key] = true
	m.launchesPending = pending
	return true
}

func (m *Model) finishLaunch(key string) {
	if !m.launchesPending[key] {
		return
	}
	pending := maps.Clone(m.launchesPending)
	delete(pending, key)
	m.launchesPending = pending
}
