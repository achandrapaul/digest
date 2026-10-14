package spotlight

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/achandrapaul/digest/pkg/paths"
	"github.com/achandrapaul/digest/pkg/system"
)

const (
	ShortcutName     = "Add Digest Note"
	shortcutsCommand = "/usr/bin/shortcuts"
	askActionUUID    = "6F1C2D8A-3B4E-4C5F-9A7B-1D2E3F4A5B6C"
)

var (
	runCommand = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	}
	shortcutsPath = shortcutsCommand
)

func Available() bool {
	return system.Exists(shortcutsPath)
}

func Installed() bool {
	output, err := runCommand(shortcutsPath, "list")
	if err != nil {
		return false
	}
	for name := range strings.SplitSeq(string(output), "\n") {
		if strings.TrimSpace(name) == ShortcutName {
			return true
		}
	}
	return false
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
	return filepath.Join(base, "digest", "spotlight"), nil
}

func ShortcutPath(dir string) string {
	return filepath.Join(dir, ShortcutName+".shortcut")
}

func unsignedPath(dir string) string {
	return filepath.Join(dir, ShortcutName+".unsigned.shortcut")
}

func xmlText(value string) string {
	var escaped bytes.Buffer
	xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

func Script(executable string) string {
	quotedExecutable := "'" + strings.ReplaceAll(executable, "'", `'\''`) + "'"
	return quotedExecutable + ` note add "$(cat)"`
}

func Workflow(executable string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>WFWorkflowActions</key>
	<array>
		<dict>
			<key>WFWorkflowActionIdentifier</key>
			<string>is.workflow.actions.ask</string>
			<key>WFWorkflowActionParameters</key>
			<dict>
				<key>UUID</key>
				<string>` + askActionUUID + `</string>
				<key>WFAskActionPrompt</key>
				<string>Note</string>
				<key>WFInputType</key>
				<string>Text</string>
			</dict>
		</dict>
		<dict>
			<key>WFWorkflowActionIdentifier</key>
			<string>is.workflow.actions.runshellscript</string>
			<key>WFWorkflowActionParameters</key>
			<dict>
				<key>Input</key>
				<dict>
					<key>Value</key>
					<dict>
						<key>OutputName</key>
						<string>Provided Input</string>
						<key>OutputUUID</key>
						<string>` + askActionUUID + `</string>
						<key>Type</key>
						<string>ActionOutput</string>
					</dict>
					<key>WFSerializationType</key>
					<string>WFTextTokenAttachment</string>
				</dict>
				<key>InputMode</key>
				<string>to stdin</string>
				<key>Script</key>
				<string>` + xmlText(Script(executable)) + `</string>
				<key>Shell</key>
				<string>/bin/zsh</string>
			</dict>
		</dict>
	</array>
	<key>WFWorkflowClientVersion</key>
	<string>2302.0.4</string>
	<key>WFWorkflowMinimumClientVersion</key>
	<integer>900</integer>
	<key>WFWorkflowIcon</key>
	<dict>
		<key>WFWorkflowIconGlyphNumber</key>
		<integer>59511</integer>
		<key>WFWorkflowIconStartColor</key>
		<integer>4282601983</integer>
	</dict>
	<key>WFWorkflowImportQuestions</key>
	<array/>
	<key>WFWorkflowInputContentItemClasses</key>
	<array>
		<string>WFStringContentItem</string>
	</array>
	<key>WFWorkflowTypes</key>
	<array>
		<string>MenuBar</string>
	</array>
</dict>
</plist>
`
}

func Install(executable, dir string) (string, error) {
	if err := system.MkdirAll(dir); err != nil {
		return "", err
	}
	unsigned := unsignedPath(dir)
	if err := system.WriteWithMode(unsigned, []byte(Workflow(executable)), 0600); err != nil {
		return "", err
	}
	signed := ShortcutPath(dir)
	if output, err := runCommand(shortcutsPath, "sign", "--mode", "anyone", "--input", unsigned, "--output", signed); err != nil {
		return "", commandError("shortcuts sign", output, err)
	}
	if output, err := runCommand("open", signed); err != nil {
		return "", commandError("open", output, err)
	}
	return signed, nil
}

func Uninstall(dir string) error {
	return system.RemoveAll(dir)
}

func commandError(command string, output []byte, err error) error {
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		return fmt.Errorf("%s: %w: %s", command, err, trimmed)
	}
	return fmt.Errorf("%s: %w", command, err)
}
