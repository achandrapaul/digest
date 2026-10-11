package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system"
)

func TestRelockRootCmdOnlyRunsForAProtectedRoot(t *testing.T) {
	root := t.TempDir()
	if relockRootCmd(root) != nil {
		t.Fatalf("an unprotected root must not be relocked")
	}
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	system.Protect(root)
	t.Cleanup(func() {
		system.Protect("")
		_ = system.UnlockTree(root)
	})
	cmd := relockRootCmd(root)
	if cmd == nil {
		t.Fatalf("a protected root should be relocked in the background")
	}
	if msg, isRelock := cmd().(rootRelockedMsg); !isRelock || msg.err != nil {
		t.Fatalf("relock msg = %#v", msg)
	}
}

func TestInitRelocksAProtectedRoot(t *testing.T) {
	m := syncTestModel(t)
	system.Protect(m.cfg.Root())
	t.Cleanup(func() {
		system.Protect("")
		_ = system.UnlockTree(m.cfg.Root())
	})
	found := false
	for _, msg := range collectMsgsWithoutTicks(m.Init()) {
		if _, isRelock := msg.(rootRelockedMsg); isRelock {
			found = true
		}
	}
	if !found {
		t.Fatalf("Init should relock the digest root")
	}
}

func TestRelockFailureShowsAsAMessage(t *testing.T) {
	m := syncTestModel(t)
	next, _ := m.Update(rootRelockedMsg{err: errors.New("2 locked, 1 failed")})
	if got := latestMessageText(next.(Model)); got != "2 locked, 1 failed" {
		t.Fatalf("relock failure message = %q", got)
	}
}

func TestUnknownConfigKeysShowANoticeOnStartup(t *testing.T) {
	cfg := &config.Config{DigestRoot: t.TempDir(), GreenOnly: true, UnknownKeys: []string{"green_onyl", "show_tag"}}
	m := NewModel(cfg, nil)
	m.width, m.height = 120, 50
	if m.mode != ViewDashboard {
		t.Fatalf("the notice must not block the dashboard, mode %v", m.mode)
	}
	if got := headerTopRow(m); !strings.Contains(got, "config.yaml: unknown keys green_onyl, show_tag — run digest doctor") {
		t.Fatalf("header should show the unknown keys notice: %q", got)
	}
	for _, message := range syncTestModel(t).messages {
		if strings.Contains(message.text, "unknown keys") {
			t.Fatalf("no notice without unknown keys: %+v", message)
		}
	}
}
