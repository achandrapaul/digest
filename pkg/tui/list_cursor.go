package tui

type listCursor struct {
	selected int
	scroll   int
}

func (cursor *listCursor) Move(delta, count int) {
	cursor.selected += delta
	cursor.Clamp(count)
}

func (cursor *listCursor) Clamp(count int) {
	cursor.selected = min(max(cursor.selected, 0), max(count-1, 0))
}

func (cursor *listCursor) Follow(heights []int, available int) (first, last int) {
	first, last = searchWindow(heights, cursor.selected, cursor.scroll, available)
	cursor.scroll = first
	return first, last
}
