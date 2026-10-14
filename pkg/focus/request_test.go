package focus

import (
	"testing"

	"github.com/achandrapaul/digest/pkg/system"
)

func TestWriteKeepsOnlyTheLatestSelector(t *testing.T) {
	root := t.TempDir()
	first := Selector{Org: "o", Repo: "r", Title: "first"}
	second := Selector{Org: "o", Repo: "r", Title: "second"}
	if err := Write(root, first); err != nil {
		t.Fatal(err)
	}
	if err := Write(root, second); err != nil {
		t.Fatal(err)
	}
	taken, found, err := Take(root)
	if err != nil || !found || taken != second {
		t.Fatalf("Take = %+v, %v, %v", taken, found, err)
	}
	if system.Exists(RequestPath(root)) {
		t.Error("Take should delete the focus file")
	}
}

func TestTakeWithoutAFileFindsNothing(t *testing.T) {
	if _, found, err := Take(t.TempDir()); found || err != nil {
		t.Errorf("Take = %v, %v", found, err)
	}
}

func TestTakeDeletesAnUnreadableFile(t *testing.T) {
	root := t.TempDir()
	if err := system.Write(RequestPath(root), []byte("{")); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Take(root); found || err == nil {
		t.Errorf("Take = %v, %v", found, err)
	}
	if system.Exists(RequestPath(root)) {
		t.Error("an unreadable focus file should be deleted")
	}
}
