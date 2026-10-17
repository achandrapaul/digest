package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/store"
	"github.com/achandrapaul/digest/pkg/system"
)

const mainArgsSeparator = "\x1f"

func TestRunDigestMain(t *testing.T) {
	raw, found := os.LookupEnv("DIGEST_MAIN_ARGS")
	if !found {
		return
	}
	os.Args = []string{"digest"}
	if raw != "" {
		os.Args = append(os.Args, strings.Split(raw, mainArgsSeparator)...)
	}
	main()
	os.Exit(0)
}

func runDigest(t *testing.T, env []string, args ...string) (string, int) {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunDigestMain$")
	cmd.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "DIGEST_MAIN_ARGS="+strings.Join(args, mainArgsSeparator))
	cmd.Env = append(cmd.Env, env...)
	output, _ := cmd.CombinedOutput()
	return string(output), cmd.ProcessState.ExitCode()
}

func writeTestConfig(t *testing.T, content string) (string, string) {
	t.Helper()
	digestRoot := filepath.Join(t.TempDir(), "digest-root")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("digest_root: "+digestRoot+"\n"+content), 0600); err != nil {
		t.Fatal(err)
	}
	return configPath, digestRoot
}

func TestHelpPrintsDigestUsage(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"-help"}, {"janitor", "-h"}, {"migrate", "--help"}, {"doctor", "-h"}} {
		output, code := runDigest(t, nil, args...)
		if code != 0 || !strings.Contains(output, "Usage: digest [flags] [command]") || strings.Contains(output, "Usage of") || strings.Contains(output, "Unknown command") {
			t.Errorf("%v: exit %d, want digest's usage:\n%s", args, code, output)
		}
	}
}

func TestUnknownFlagsPrintDigestUsage(t *testing.T) {
	for _, args := range [][]string{{"--bogus"}, {"-test.v"}, {"janitor", "--bogus"}, {"repo-sync", "-x"}, {"doctor", "--bogus"}, {"tui", "--bogus"}, {"pr-review", "--bogus"}, {"migrate", "--bogus"}} {
		output, code := runDigest(t, nil, args...)
		if code != 1 || !strings.Contains(output, "Usage: digest [flags] [command]") || strings.Contains(output, "Usage of") {
			t.Errorf("%v: exit %d, want digest's usage and exit 1:\n%s", args, code, output)
		}
		if !strings.Contains(output, "flag provided but not defined") {
			t.Errorf("%v: output should name the bad flag:\n%s", args, output)
		}
	}
}

func TestCommandsWithoutDryRunSayItIsNotSupported(t *testing.T) {
	cases := map[string][]string{
		"pr-review":             {"pr-review", "--url", "https://github.com/o/r/pull/1"},
		"brag":                  {"brag", "--week", "2026-W40"},
		"automation":            {"automation", "--note", "x", "--name", "y"},
		"init":                  {"init"},
		"summary":               {"summary"},
		"install notifications": {"install", "notifications"},
	}
	for name, args := range cases {
		configPath, _ := writeTestConfig(t, "")
		for _, form := range [][]string{append([]string{"--config", configPath, "--dry-run"}, args...), append(append([]string{"--config", configPath}, args...), "--dry-run")} {
			output, code := runDigest(t, nil, form...)
			if code == 0 || !strings.Contains(output, name+" doesn't support dry-run") || strings.Contains(output, "flag provided but not defined") {
				t.Errorf("%v: exit %d, want %q:\n%s", form, code, name+" doesn't support dry-run", output)
			}
		}
		if content, _ := os.ReadFile(configPath); strings.Contains(string(content), "jobs:") {
			t.Errorf("%s: a refused dry run touched the config", name)
		}
	}
}

func TestInitWritesTheConfigAndNeedsForceToRewrite(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "digest", "config.yaml")
	if output, code := runDigest(t, nil, "--config", configPath, "init"); code != 0 || !strings.Contains(output, "Wrote "+configPath) {
		t.Fatalf("init: exit %d\n%s", code, output)
	}
	if err := os.WriteFile(configPath, []byte("digest_root: /mine\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if output, code := runDigest(t, nil, "--config", configPath, "init"); code == 0 || !strings.Contains(output, "pass --force") {
		t.Errorf("init over an existing config: exit %d\n%s", code, output)
	}
	output, code := runDigest(t, nil, "--config", configPath, "init", "--force")
	if code != 0 || !strings.Contains(output, "Backed up "+configPath+".bak") || !strings.Contains(output, "Wrote "+configPath) {
		t.Errorf("init --force: exit %d\n%s", code, output)
	}
	if backup, _ := os.ReadFile(configPath + ".bak"); string(backup) != "digest_root: /mine\n" {
		t.Errorf("backup = %q", backup)
	}
	if output, code := runDigest(t, nil, "--config", configPath, "export", "config"); code == 0 {
		t.Errorf("export config should be gone, exit %d\n%s", code, output)
	}
}

func TestInitAutomationsRewritesOnlyTheAutomations(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "digest", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}
	original := "digest_root: /mine\nautomations:\n  - name: jira\n    draft_prompt: \"\"\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	output, code := runDigest(t, nil, "--config", configPath, "init", "--automations")
	if code != 0 || !strings.Contains(output, "Backed up "+configPath+".bak") || !strings.Contains(output, "Wrote the default automations to "+configPath) {
		t.Fatalf("init --automations: exit %d\n%s", code, output)
	}
	content, _ := os.ReadFile(configPath)
	if !strings.HasPrefix(string(content), "digest_root: /mine\n") || !strings.Contains(string(content), "name: google calendar") {
		t.Errorf("config = %s", content)
	}
}

func TestBuiltInJobsRunAgainstTheirRootsOption(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "gone")
	configPath, _ := writeTestConfig(t, "jobs:\n  - name: janitor\n    options:\n      roots: ["+missingRoot+"]\n")
	output, code := runDigest(t, nil, "--config", configPath, "janitor", "--dry-run")
	if code == 0 || !strings.Contains(output, "Running janitor job against "+missingRoot) {
		t.Errorf("janitor should use its roots option: exit %d\n%s", code, output)
	}
	otherRoot := t.TempDir()
	output, _ = runDigest(t, nil, "--config", configPath, "janitor", "--dry-run", "--root", otherRoot)
	if !strings.Contains(output, "Running janitor job against "+otherRoot+" ") {
		t.Errorf("--root should override the roots option:\n%s", output)
	}
	output, _ = runDigest(t, []string{"DIGEST_OPTION_ROOTS=" + otherRoot}, "--config", configPath, "janitor", "--dry-run")
	if !strings.Contains(output, "Running janitor job against "+otherRoot+" ") {
		t.Errorf("a dashboard run's roots env should win over config.yaml:\n%s", output)
	}
}

func TestStartupWritesNoConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing", "config.yaml")
	runDigest(t, nil, "--config", configPath, "doctor")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("a command wrote the default config, stat err = %v", err)
	}
}

const oldJanitorKeysConfig = "retention_days: 3\njanitor_patterns: [a.log]\njobs:\n  - name: janitor\n    command: digest janitor\n"

func TestMigrateDryRunChangesNothing(t *testing.T) {
	for _, form := range [][]string{{"migrate", "--dry-run"}, {"--dry-run", "migrate"}} {
		configPath, _ := writeTestConfig(t, oldJanitorKeysConfig)
		before, _ := os.ReadFile(configPath)
		output, code := runDigest(t, nil, append([]string{"--config", configPath}, form...)...)
		after, _ := os.ReadFile(configPath)
		if code != 0 || string(before) != string(after) {
			t.Errorf("%v: exit %d, config changed %v:\n%s", form, code, string(before) != string(after), output)
		}
		if !strings.Contains(output, "would move retention_days, janitor_patterns into the janitor job's options") || !strings.Contains(output, "notes would be migrated") {
			t.Errorf("%v: output should list what a real run does:\n%s", form, output)
		}
	}
}

func TestJanitorNoQuarantineDeletesForGood(t *testing.T) {
	for name, setting := range map[string]struct {
		flag []string
		env  []string
	}{
		"flag":   {[]string{"--no-quarantine"}, nil},
		"option": {nil, []string{"DIGEST_OPTION_NO_QUARANTINE=true"}},
	} {
		configPath, digestRoot := writeTestConfig(t, "")
		scanned := t.TempDir()
		heapDump := filepath.Join(scanned, "java.hprof")
		if err := os.WriteFile(heapDump, []byte("heap"), 0600); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"--config", configPath, "janitor", "--root", scanned}, setting.flag...)
		output, _ := runDigest(t, append([]string{"DIGEST_OPTION_PATTERNS=*.hprof"}, setting.env...), args...)
		if _, err := os.Stat(heapDump); !os.IsNotExist(err) {
			t.Errorf("%s: the match should be deleted:\n%s", name, output)
		}
		if _, err := os.Stat(filepath.Join(digestRoot, ".quarantine")); !os.IsNotExist(err) {
			t.Errorf("%s: nothing should be quarantined:\n%s", name, output)
		}
	}
}

func TestNoteAddSavesAManualNote(t *testing.T) {
	configPath, digestRoot := writeTestConfig(t, "")
	t.Cleanup(func() { _ = system.UnlockTree(digestRoot) })
	output, code := runDigest(t, nil, "--config", configPath, "note", "add", "Call", "the bank")
	if code != 0 || !strings.Contains(output, "Added note ") {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	notes, err := store.New(filepath.Join(digestRoot, "notes")).List()
	if err != nil || len(notes) != 1 || notes[0].Summary != "Call the bank" || notes[0].Source != model.SourceManual || notes[0].Status != model.StatusActive {
		t.Fatalf("notes = %+v, err = %v", notes, err)
	}
}

func TestNoteAddRejectsEmptyTextAndUnknownSubcommands(t *testing.T) {
	configPath, _ := writeTestConfig(t, "")
	for _, args := range [][]string{{"note", "add"}, {"note", "add", "  "}, {"note"}, {"note", "remove", "x"}} {
		output, code := runDigest(t, nil, append([]string{"--config", configPath}, args...)...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1:\n%s", args, code, output)
		}
	}
}

func TestFocusRejectsABadSelector(t *testing.T) {
	configPath, _ := writeTestConfig(t, "")
	for _, args := range [][]string{{"focus"}, {"focus", "org:repo"}} {
		output, code := runDigest(t, nil, append([]string{"--config", configPath}, args...)...)
		if code != 1 || !strings.Contains(output, "Usage: digest focus <org>:<repo>:<pr title>") {
			t.Errorf("%v: exit %d:\n%s", args, code, output)
		}
	}
}

func TestFocusWithoutARunningDigestWritesNoFocusFile(t *testing.T) {
	configPath, digestRoot := writeTestConfig(t, "")
	t.Cleanup(func() { _ = system.UnlockTree(digestRoot) })
	if output, code := runDigest(t, nil, "--config", configPath, "focus", "o:r:fix: a title"); code != 0 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if system.Exists(filepath.Join(digestRoot, ".state", "focus.json")) {
		t.Error("focus file written without a running digest")
	}
}

func TestSummaryPrintsTheDashboardGroups(t *testing.T) {
	configPath, digestRoot := writeTestConfig(t, "")
	t.Cleanup(func() { _ = system.UnlockTree(digestRoot) })
	noteStore := store.New(filepath.Join(digestRoot, "notes"))
	now := time.Now()
	for _, note := range []*model.Note{
		{Summary: "pending item", Status: model.StatusActive, Source: model.SourceManual, Created: now.AddDate(0, 0, -10), Updated: now.AddDate(0, 0, -10)},
		{Summary: "today item", Status: model.StatusActive, Source: model.SourceManual, Created: now, Updated: now},
		{Summary: "Own:OPEN:o:r:my pr", Status: model.StatusActive, Source: model.SourceMyPR, Created: now, Updated: now},
		{Summary: "Approved:r:1 review", Status: model.StatusActive, Source: model.SourcePRReview, Created: now.AddDate(0, 0, -10), Updated: now.AddDate(0, 0, -10)},
	} {
		if err := noteStore.Save(note); err != nil {
			t.Fatal(err)
		}
	}

	output, code := runDigest(t, nil, "--config", configPath, "summary")
	if code != 0 || !strings.Contains(output, "Pending (2)") || !strings.Contains(output, "Today (2)") || !strings.Contains(output, "#my-pr") || !strings.Contains(output, "Closed today (0)\n  none") {
		t.Fatalf("exit %d:\n%s", code, output)
	}

	output, code = runDigest(t, nil, "--config", configPath, "summary", "--exclude=pr")
	if code != 0 || !strings.Contains(output, "Pending (1)\n  - pending item #manual") || !strings.Contains(output, "Today (1)\n  - today item #manual") || strings.Contains(output, "#pr-review") || strings.Contains(output, "#my-pr") {
		t.Fatalf("--exclude=pr: exit %d:\n%s", code, output)
	}

	if output, code = runDigest(t, nil, "--config", configPath, "summary", "--exclude=bogus"); code != 1 || !strings.Contains(output, `unknown --exclude value "bogus"`) {
		t.Fatalf("--exclude=bogus: exit %d:\n%s", code, output)
	}

	output, code = runDigest(t, nil, "--config", configPath, "summary", "--json", "--exclude=pr")
	var report summaryReport
	if code != 0 || json.Unmarshal([]byte(output), &report) != nil || len(report.Sections) != 4 {
		t.Fatalf("exit %d:\n%s", code, output)
	}
	if report.Sections[1].Key != "pending" || len(report.Sections[1].Notes) != 1 || report.Sections[1].Notes[0].Summary != "pending item" {
		t.Fatalf("pending section = %+v", report.Sections[1])
	}
}
