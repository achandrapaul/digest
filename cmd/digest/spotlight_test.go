package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeShortcuts(t *testing.T, available bool) (dir string, installs *int) {
	t.Helper()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	installCount := 0
	previousAvailable, previousInstall := spotlightAvailable, spotlightInstall
	spotlightAvailable = func() bool { return available }
	spotlightInstall = func(executable, dir string) (string, error) {
		installCount++
		return filepath.Join(dir, "Add Digest Note.shortcut"), os.MkdirAll(dir, 0700)
	}
	t.Cleanup(func() { spotlightAvailable, spotlightInstall = previousAvailable, previousInstall })
	return filepath.Join(configHome, "digest", "spotlight"), &installCount
}

func TestInstallSpotlightFailsWithoutTheShortcutsCommand(t *testing.T) {
	_, installs := fakeShortcuts(t, false)
	err := runSpotlightCommand("install", &strings.Builder{})
	if !errors.Is(err, errShortcutsMissing) || err.Error() != "the shortcuts command is not available (needs macOS 12 or later)" {
		t.Fatalf("err = %v", err)
	}
	if *installs != 0 {
		t.Error("installed without the shortcuts command")
	}
}

func TestInstallSpotlightSaysHowToFinishAdding(t *testing.T) {
	_, installs := fakeShortcuts(t, true)
	var out strings.Builder
	if err := runSpotlightCommand("install", &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Add Shortcut", "Settings", "Advanced", "Allow Running Scripts", "menu bar"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
	if *installs != 1 {
		t.Errorf("installs = %d", *installs)
	}
}

func TestUninstallSpotlightRemovesTheFolderAndSaysToDeleteTheShortcut(t *testing.T) {
	dir, _ := fakeShortcuts(t, false)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := runSpotlightCommand("uninstall", &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("folder still there: %v", err)
	}
	if !strings.Contains(out.String(), `delete "Add Digest Note"`) {
		t.Errorf("output:\n%s", out.String())
	}
}
