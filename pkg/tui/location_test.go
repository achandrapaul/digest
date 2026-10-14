package tui

import (
	"os"
	"testing"

	"github.com/achandrapaul/digest/pkg/appstate"
)

func TestRecordLocationSavesTheTmuxPaneAndTerminal(t *testing.T) {
	root := t.TempDir()
	if err := appstate.Save(root, appstate.State{Streak: 2}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX", "/private/tmp/tmux-501/default,123,0")
	t.Setenv("TMUX_PANE", "%4")
	t.Setenv("__CFBundleIdentifier", "com.mitchellh.ghostty")
	if err := recordLocation(root); err != nil {
		t.Fatal(err)
	}
	saved, _, _ := appstate.Load(root)
	want := appstate.TUILocation{PID: os.Getpid(), TmuxSocket: "/private/tmp/tmux-501/default", TmuxPane: "%4", Terminal: "com.mitchellh.ghostty"}
	if saved.TUILocation == nil || *saved.TUILocation != want || saved.Streak != 2 {
		t.Fatalf("saved = %+v", saved)
	}
	if err := clearLocation(root); err != nil {
		t.Fatal(err)
	}
	if saved, _, _ := appstate.Load(root); saved.TUILocation != nil || saved.Streak != 2 {
		t.Errorf("after clear = %+v", saved)
	}
}

func TestRecordLocationOutsideTmuxLeavesThePaneEmpty(t *testing.T) {
	root := t.TempDir()
	if err := appstate.Save(root, appstate.State{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	t.Setenv("__CFBundleIdentifier", "com.apple.Terminal")
	if err := recordLocation(root); err != nil {
		t.Fatal(err)
	}
	saved, _, _ := appstate.Load(root)
	if saved.TUILocation == nil || saved.TUILocation.TmuxPane != "" || saved.TUILocation.TmuxSocket != "" || saved.TUILocation.Terminal != "com.apple.Terminal" {
		t.Fatalf("saved = %+v", saved.TUILocation)
	}
}

func TestClearLocationKeepsAnotherInstancesLocation(t *testing.T) {
	root := t.TempDir()
	other := appstate.TUILocation{PID: os.Getpid() + 1}
	if err := appstate.Save(root, appstate.State{TUILocation: &other}); err != nil {
		t.Fatal(err)
	}
	if err := clearLocation(root); err != nil {
		t.Fatal(err)
	}
	if saved, _, _ := appstate.Load(root); saved.TUILocation == nil || *saved.TUILocation != other {
		t.Errorf("another instance's location was cleared: %+v", saved.TUILocation)
	}
}

func TestRecordLocationNeverCreatesTheAppState(t *testing.T) {
	root := t.TempDir()
	if err := recordLocation(root); err != nil {
		t.Fatal(err)
	}
	if err := clearLocation(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(appstate.Path(root)); !os.IsNotExist(err) {
		t.Errorf("app state should stay missing for requireAppState: %v", err)
	}
}
