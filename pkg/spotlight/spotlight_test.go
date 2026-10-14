package spotlight

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
}

func fakeCommands(t *testing.T, listOutput string, failing string) *[]recordedCommand {
	t.Helper()
	var recorded []recordedCommand
	previous := runCommand
	runCommand = func(name string, args ...string) ([]byte, error) {
		recorded = append(recorded, recordedCommand{name, args})
		if len(args) > 0 && args[0] == failing {
			return []byte("not signed in"), errors.New("exit status 1")
		}
		if len(args) > 0 && args[0] == "list" {
			return []byte(listOutput), nil
		}
		return nil, nil
	}
	t.Cleanup(func() { runCommand = previous })
	return &recorded
}

func TestAvailableNeedsTheShortcutsCommand(t *testing.T) {
	previous := shortcutsPath
	t.Cleanup(func() { shortcutsPath = previous })
	shortcutsPath = filepath.Join(t.TempDir(), "shortcuts")
	if Available() {
		t.Error("Available() without the shortcuts command")
	}
	if err := os.WriteFile(shortcutsPath, nil, 0700); err != nil {
		t.Fatal(err)
	}
	if !Available() {
		t.Error("Available() missed the shortcuts command")
	}
}

func TestInstalledLooksForTheShortcutByName(t *testing.T) {
	fakeCommands(t, "Morning\nAdd Digest Note\n", "")
	if !Installed() {
		t.Error("the added shortcut was missed")
	}
	fakeCommands(t, "Morning\nAdd Digest Note copy\n", "")
	if Installed() {
		t.Error("a differently named shortcut counted")
	}
}

func TestWorkflowAsksForTheNoteRunsNoteAddAndPinsToTheMenuBar(t *testing.T) {
	workflow := Workflow("/Users/me/my <bin>/digest")
	for _, want := range []string{
		"<string>is.workflow.actions.ask</string>",
		"<string>Note</string>",
		"<string>is.workflow.actions.runshellscript</string>",
		"<string>to stdin</string>",
		`<string>&#39;/Users/me/my &lt;bin&gt;/digest&#39; note add &#34;$(cat)&#34;</string>`,
		"<key>WFWorkflowTypes</key>\n\t<array>\n\t\t<string>MenuBar</string>",
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("workflow is missing %q:\n%s", want, workflow)
		}
	}
	if strings.Count(workflow, askActionUUID) != 2 {
		t.Error("the shell script should read the ask action's output")
	}
}

func TestScriptQuotesTheDigestPath(t *testing.T) {
	if got := Script("/Users/o'neil/digest"); got != `'/Users/o'\''neil/digest' note add "$(cat)"` {
		t.Errorf("Script() = %s", got)
	}
}

func TestInstallSignsAndOpensTheShortcut(t *testing.T) {
	recorded := fakeCommands(t, "", "")
	dir := filepath.Join(t.TempDir(), "spotlight")
	signed, err := Install("/bin/digest", dir)
	if err != nil {
		t.Fatal(err)
	}
	if signed != ShortcutPath(dir) {
		t.Errorf("signed = %q, want %q", signed, ShortcutPath(dir))
	}
	if content, err := os.ReadFile(unsignedPath(dir)); err != nil || !strings.Contains(string(content), "&#39;/bin/digest&#39; note add") {
		t.Fatalf("unsigned shortcut = %q, %v", content, err)
	}
	want := []recordedCommand{
		{shortcutsPath, []string{"sign", "--mode", "anyone", "--input", unsignedPath(dir), "--output", signed}},
		{"open", []string{signed}},
	}
	if !slices.EqualFunc(*recorded, want, func(a, b recordedCommand) bool { return a.name == b.name && slices.Equal(a.args, b.args) }) {
		t.Errorf("commands = %v, want %v", *recorded, want)
	}
}

func TestInstallStopsWhenSigningFails(t *testing.T) {
	recorded := fakeCommands(t, "", "sign")
	_, err := Install("/bin/digest", filepath.Join(t.TempDir(), "spotlight"))
	if err == nil || !strings.Contains(err.Error(), "shortcuts sign") || !strings.Contains(err.Error(), "not signed in") {
		t.Fatalf("err = %v", err)
	}
	if len(*recorded) != 1 {
		t.Errorf("nothing should be opened after a failed sign: %v", *recorded)
	}
}

func TestUninstallRemovesTheFolder(t *testing.T) {
	fakeCommands(t, "", "")
	dir := filepath.Join(t.TempDir(), "spotlight")
	if _, err := Install("/bin/digest", dir); err != nil {
		t.Fatal(err)
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
	if dir, err := Dir(); err != nil || dir != "/tmp/xdg/digest/spotlight" {
		t.Errorf("Dir() = %q, %v", dir, err)
	}
}
