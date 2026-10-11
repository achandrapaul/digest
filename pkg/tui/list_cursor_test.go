package tui

import "testing"

func TestListCursorMoveStaysInsideTheList(t *testing.T) {
	cursor := listCursor{}
	cursor.Move(-1, 3)
	if cursor.selected != 0 {
		t.Fatalf("moving up from the top = %d", cursor.selected)
	}
	cursor.Move(5, 3)
	if cursor.selected != 2 {
		t.Fatalf("moving past the end = %d", cursor.selected)
	}
	cursor.Move(-1, 3)
	if cursor.selected != 1 {
		t.Fatalf("moving up one = %d", cursor.selected)
	}
	cursor.Move(1, 0)
	if cursor.selected != 0 {
		t.Fatalf("an empty list selects 0, got %d", cursor.selected)
	}
}

func TestListCursorClamp(t *testing.T) {
	for _, tc := range []struct{ selected, count, want int }{{5, 3, 2}, {-2, 3, 0}, {1, 3, 1}, {4, 0, 0}} {
		cursor := listCursor{selected: tc.selected}
		cursor.Clamp(tc.count)
		if cursor.selected != tc.want {
			t.Errorf("Clamp(%d) from %d = %d, want %d", tc.count, tc.selected, cursor.selected, tc.want)
		}
	}
}

func TestListCursorFollowKeepsTheSelectionVisible(t *testing.T) {
	heights := []int{2, 2, 2, 2, 2}
	cursor := listCursor{selected: 4}
	first, last := cursor.Follow(heights, 4)
	if cursor.scroll != 3 || first != 3 || last != 5 {
		t.Fatalf("scroll %d window %d..%d, want 3 and 3..5", cursor.scroll, first, last)
	}
	cursor.selected = 0
	if first, _ := cursor.Follow(heights, 4); first != 0 || cursor.scroll != 0 {
		t.Fatalf("scrolling back to the top: first %d scroll %d", first, cursor.scroll)
	}
}
