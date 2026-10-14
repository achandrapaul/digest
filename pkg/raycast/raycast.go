package raycast

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/achandrapaul/digest/pkg/paths"
	"github.com/achandrapaul/digest/pkg/system"
)

const scriptName = "digest-note-add.sh"

var appLocations = []string{"/Applications/Raycast.app", "~/Applications/Raycast.app"}

var spotlightLookup = func() bool {
	output, err := exec.Command("mdfind", "kMDItemCFBundleIdentifier == 'com.raycast.macos'").Output()
	return err == nil && spotlightFound(string(output))
}

func spotlightFound(output string) bool {
	for _, appPath := range strings.Split(output, "\n") {
		if strings.HasSuffix(appPath, "Raycast.app") && !strings.Contains(appPath, "/Application Support/") {
			return true
		}
	}
	return false
}

func Installed() bool {
	for _, location := range appLocations {
		if expanded, err := paths.Expand(location); err == nil && system.Exists(expanded) {
			return true
		}
	}
	return spotlightLookup()
}

func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" || !filepath.IsAbs(base) {
		expanded, err := paths.Expand("~/.config")
		if err != nil {
			return "", err
		}
		base = expanded
	}
	return filepath.Join(base, "digest", "raycast"), nil
}

func ScriptPath(dir string) string {
	return filepath.Join(dir, scriptName)
}

func Script(executable string) string {
	quotedExecutable := "'" + strings.ReplaceAll(executable, "'", `'\''`) + "'"
	return strings.Join([]string{
		"#!/bin/bash",
		"",
		"# @raycast.schemaVersion 1",
		"# @raycast.title Add Digest Note",
		"# @raycast.mode silent",
		"# @raycast.packageName Digest",
		"# @raycast.icon 📝",
		`# @raycast.argument1 { "type": "text", "placeholder": "Note" }`,
		"",
		quotedExecutable + ` note add "$1"`,
		"",
	}, "\n")
}

func Install(executable, dir string) (string, error) {
	if err := system.MkdirAll(dir); err != nil {
		return "", err
	}
	scriptPath := ScriptPath(dir)
	return scriptPath, system.WriteWithMode(scriptPath, []byte(Script(executable)), 0700)
}

func Uninstall(dir string) error {
	return system.RemoveAll(dir)
}
