package raycast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeRaycast(t *testing.T, appPresent bool, spotlightFinds bool) {
	t.Helper()
	appDir := filepath.Join(t.TempDir(), "Raycast.app")
	if appPresent {
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	previousLocations, previousLookup := appLocations, spotlightLookup
	appLocations = []string{appDir}
	spotlightLookup = func() bool { return spotlightFinds }
	t.Cleanup(func() { appLocations, spotlightLookup = previousLocations, previousLookup })
}

func TestInstalledFindsTheAppOrFallsBackToSpotlight(t *testing.T) {
	cases := []struct {
		appPresent, spotlightFinds, want bool
	}{{true, false, true}, {false, true, true}, {false, false, false}}
	for _, testCase := range cases {
		fakeRaycast(t, testCase.appPresent, testCase.spotlightFinds)
		if got := Installed(); got != testCase.want {
			t.Errorf("app %v, spotlight %v: Installed() = %v, want %v", testCase.appPresent, testCase.spotlightFinds, got, testCase.want)
		}
	}
}

func TestSpotlightIgnoresRaycastsOwnUpdateCopies(t *testing.T) {
	if spotlightFound("/Users/me/Library/Application Support/com.raycast.macos/Updates/2.7/Raycast.app\n") {
		t.Error("an update copy counted as an installed Raycast")
	}
	if !spotlightFound("/Users/me/Library/Application Support/com.raycast.macos/Updates/2.7/Raycast.app\n/Applications/Setapp/Raycast.app\n") {
		t.Error("a real Raycast.app was missed")
	}
}

func TestScriptRunsNoteAddWithTheFullDigestPath(t *testing.T) {
	script := Script("/Users/me/my bin/digest")
	for _, want := range []string{"#!/bin/bash", "# @raycast.schemaVersion 1", "# @raycast.title Add Digest Note", "# @raycast.mode silent", `# @raycast.argument1 { "type": "text", "placeholder": "Note" }`, `'/Users/me/my bin/digest' note add "$1"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script is missing %q:\n%s", want, script)
		}
	}
}

func TestInstallWritesAnExecutableScriptAndUninstallRemovesTheFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "raycast")
	scriptPath, err := Install("/bin/digest", dir)
	if err != nil {
		t.Fatal(err)
	}
	if scriptPath != ScriptPath(dir) {
		t.Errorf("scriptPath = %q, want %q", scriptPath, ScriptPath(dir))
	}
	info, err := os.Stat(scriptPath)
	if err != nil || info.Mode().Perm()&0100 == 0 {
		t.Fatalf("script not executable: %v %v", info, err)
	}
	if err := Uninstall(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("folder still there: %v", err)
	}
}

func TestDirFollowsXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if dir, err := Dir(); err != nil || dir != "/tmp/xdg/digest/raycast" {
		t.Errorf("Dir() = %q, %v", dir, err)
	}
}
