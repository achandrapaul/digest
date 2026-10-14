package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeRaycastApp(t *testing.T, installed bool) string {
	t.Helper()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	previous := raycastInstalled
	raycastInstalled = func() bool { return installed }
	t.Cleanup(func() { raycastInstalled = previous })
	return filepath.Join(configHome, "digest", "raycast")
}

func TestInstallRaycastFailsWhenRaycastIsMissing(t *testing.T) {
	dir := fakeRaycastApp(t, false)
	var out strings.Builder
	if err := runRaycastCommand("install", &out); !errors.Is(err, errRaycastMissing) || err.Error() != "the Raycast app is not installed" {
		t.Fatalf("err = %v, want the Raycast app is not installed", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("script folder written without Raycast: %v", err)
	}
}

func TestInstallRaycastWritesTheScriptAndSaysHowToAddIt(t *testing.T) {
	dir := fakeRaycastApp(t, true)
	var out strings.Builder
	if err := runRaycastCommand("install", &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "digest-note-add.sh")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), dir) || !strings.Contains(out.String(), "Add Directories") {
		t.Errorf("output:\n%s", out.String())
	}
}

func TestUninstallRaycastRemovesTheScriptEvenWithoutRaycast(t *testing.T) {
	dir := fakeRaycastApp(t, true)
	if err := runRaycastCommand("install", &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	raycastInstalled = func() bool { return false }
	if err := runRaycastCommand("uninstall", &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("script folder still there: %v", err)
	}
}
