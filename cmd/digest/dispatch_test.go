package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/notify"
)

func parseForTest(t *testing.T, args ...string) invocation {
	t.Helper()
	flagSet := flag.NewFlagSet("digest", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	parsed, err := parseInvocation(flagSet, args)
	if err != nil {
		t.Fatalf("parseInvocation(%v): %v", args, err)
	}
	return parsed
}

func TestGlobalDryRunBehavesLikeTheCommandFlag(t *testing.T) {
	cases := map[string]struct {
		global []string
		local  []string
	}{
		"janitor":       {[]string{"--dry-run", "janitor", "--root", "/tmp/a"}, []string{"janitor", "--dry-run", "--root", "/tmp/a"}},
		"repo-sync":     {[]string{"--dry-run", "repo-sync"}, []string{"repo-sync", "--dry-run"}},
		"branch-reaper": {[]string{"-dry-run", "branch-reaper", "--root", "/tmp/b"}, []string{"branch-reaper", "--dry-run", "--root", "/tmp/b"}},
		"with config":   {[]string{"--config", "/tmp/c.yaml", "--dry-run", "janitor"}, []string{"--config", "/tmp/c.yaml", "janitor", "--dry-run"}},
	}
	for name, c := range cases {
		global := parseForTest(t, c.global...)
		local := parseForTest(t, c.local...)
		if !reflect.DeepEqual(global, local) {
			t.Errorf("%s: global %+v, want %+v", name, global, local)
		}
		if !global.dryRun {
			t.Errorf("%s: dryRun = false, want true", name)
		}
	}
}

func TestTUIAndSetupIgnoreDryRun(t *testing.T) {
	plainTUI := parseForTest(t)
	for _, args := range [][]string{{"--dry-run"}, {"--dry-run", "tui"}, {"tui", "--dry-run"}, {"tui", "-dry-run=true"}} {
		if parsed := parseForTest(t, args...); !reflect.DeepEqual(parsed, plainTUI) {
			t.Errorf("%v: got %+v, want a plain tui %+v", args, parsed, plainTUI)
		}
	}
	plainSetup := parseForTest(t, "setup")
	for _, args := range [][]string{{"--dry-run", "setup"}, {"setup", "--dry-run"}} {
		if parsed := parseForTest(t, args...); !reflect.DeepEqual(parsed, plainSetup) {
			t.Errorf("%v: got %+v, want a plain setup %+v", args, parsed, plainSetup)
		}
	}
}

func TestHelpCommandIsTheHelpFlag(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		if parsed := parseForTest(t, args...); !parsed.showHelp {
			t.Errorf("%v: showHelp = false, want true", args)
		}
	}
}

func TestParseInvocationDefaultsToTheTUI(t *testing.T) {
	parsed := parseForTest(t)
	if parsed.command != "tui" || parsed.dryRun || parsed.showHelp || len(parsed.commandArgs) != 0 {
		t.Errorf("got %+v, want a plain tui", parsed)
	}
}

func TestParseInvocationKeepsTheConfigPathAndCommandArgs(t *testing.T) {
	parsed := parseForTest(t, "--config", "/tmp/custom.yaml", "uninstall", "notifications")
	if parsed.configPath != "/tmp/custom.yaml" {
		t.Errorf("configPath = %q, want /tmp/custom.yaml", parsed.configPath)
	}
	if parsed.command != "uninstall" || !reflect.DeepEqual(parsed.commandArgs, []string{"notifications"}) {
		t.Errorf("command = %q %v, want uninstall [notifications]", parsed.command, parsed.commandArgs)
	}
}

func TestLoadStartupConfigReadsTheGivenConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom.yaml")
	digestRoot := filepath.Join(t.TempDir(), "digest-root")
	if err := os.WriteFile(configPath, []byte("digest_root: "+digestRoot+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, firstRun, err := loadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("loadStartupConfig: %v", err)
	}
	if firstRun {
		t.Error("firstRun = true for an existing config, want false")
	}
	if cfg.Root() != digestRoot {
		t.Errorf("Root() = %q, want %q from the given config", cfg.Root(), digestRoot)
	}
}

func TestLoadStartupConfigWithoutAConfigIsAFirstRunThatWritesNothing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")
	cfg, firstRun, err := loadStartupConfig(configPath)
	if err != nil {
		t.Fatalf("loadStartupConfig: %v", err)
	}
	if !firstRun {
		t.Error("firstRun = false without a config, want true")
	}
	if config.Exists(configPath) {
		t.Error("startup wrote the default config; only the setup wizard should")
	}
	if cfg.Root() != config.DefaultConfig().Root() {
		t.Errorf("Root() = %q, want the defaults in memory", cfg.Root())
	}
}

func TestOnlyNonTUICommandsWarnAboutAConfigError(t *testing.T) {
	configErr := errors.New("yaml parse error")
	var tuiOutput, jobOutput strings.Builder
	reportConfigError(&tuiOutput, "tui", configErr)
	reportConfigError(&jobOutput, "janitor", configErr)
	if tuiOutput.Len() != 0 {
		t.Errorf("tui output = %q, want nothing; the header shows the error", tuiOutput.String())
	}
	if jobOutput.String() != "Warning: yaml parse error\n" {
		t.Errorf("janitor output = %q, want the warning", jobOutput.String())
	}
	reportConfigError(&jobOutput, "janitor", nil)
	if jobOutput.String() != "Warning: yaml parse error\n" {
		t.Errorf("no error should print nothing more, got %q", jobOutput.String())
	}
}

func TestPrepareTUIStartsTheSetupWizardOnFirstRun(t *testing.T) {
	gitOn := true
	cfg := &config.Config{ShowGit: &gitOn, GitRepositoryRoots: []string{filepath.Join(t.TempDir(), "no-repos")}}
	showSetup, startupErr := prepareTUI(cfg, nil, true)
	if !showSetup {
		t.Error("showSetup = false on first run, want true")
	}
	if startupErr != nil {
		t.Errorf("startupErr = %v, want the git roots left to the wizard", startupErr)
	}
	if !cfg.GitEnabled() {
		t.Error("first run switched git off before the wizard")
	}
}

func TestPrepareTUISwitchesGitOffForInvalidRoots(t *testing.T) {
	gitOn := true
	emptyRoot := t.TempDir()
	cfg := &config.Config{ShowGit: &gitOn, GitRepositoryRoots: []string{emptyRoot}}
	showSetup, startupErr := prepareTUI(cfg, nil, false)
	if showSetup {
		t.Error("showSetup = true for an existing config, want false")
	}
	if cfg.GitEnabled() {
		t.Error("git still enabled with no repos under the roots")
	}
	if startupErr == nil || !strings.Contains(startupErr.Error(), "no git repos found under "+emptyRoot) {
		t.Errorf("startupErr = %v, want the git roots error for the header", startupErr)
	}
}

func TestPrepareTUIKeepsTheConfigErrorAlongsideTheGitRootsError(t *testing.T) {
	gitOn := true
	cfg := &config.Config{ShowGit: &gitOn}
	configErr := os.ErrPermission
	_, startupErr := prepareTUI(cfg, configErr, false)
	if startupErr == nil || !strings.Contains(startupErr.Error(), configErr.Error()) || !strings.Contains(startupErr.Error(), "git_repository_roots is empty") {
		t.Errorf("startupErr = %v, want both the config and git roots errors", startupErr)
	}
}

func TestPrepareTUILeavesValidGitRootsAlone(t *testing.T) {
	gitOff := false
	cfg := &config.Config{ShowGit: &gitOff}
	showSetup, startupErr := prepareTUI(cfg, nil, false)
	if startupErr != nil || showSetup {
		t.Errorf("got (%v, %v), want (nil, false) with git off", startupErr, showSetup)
	}
}

func fakeLaunchctl(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	callsPath := filepath.Join(binDir, "calls")
	script := "#!/bin/sh\necho \"$@\" >> " + callsPath + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "launchctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	return callsPath
}

func TestUninstallNotificationsRemovesTheLaunchAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	callsPath := fakeLaunchctl(t)
	plistPath := notify.LaunchAgentPath()
	if !strings.HasPrefix(plistPath, home) {
		t.Fatalf("LaunchAgentPath() = %q, want it under the test home %q", plistPath, home)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plistPath, []byte("<plist/>"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := runNotificationsCommand(&config.Config{}, "uninstall", []string{"notifications"}); err != nil {
		t.Fatalf("uninstall notifications: %v", err)
	}
	if notify.Installed() {
		t.Error("launch agent still installed after uninstall")
	}
	calls, err := os.ReadFile(callsPath)
	if err != nil || !strings.Contains(string(calls), "bootout") || !strings.Contains(string(calls), notify.LaunchAgentLabel) {
		t.Errorf("launchctl calls = %q (%v), want a bootout of %s", calls, err, notify.LaunchAgentLabel)
	}
}

func TestNotificationsCommandRejectsWrongUsage(t *testing.T) {
	cases := map[string][]string{
		"no target":      nil,
		"other target":   {"reminders"},
		"extra argument": {"notifications", "now"},
	}
	for name, args := range cases {
		err := runNotificationsCommand(&config.Config{}, "uninstall", args)
		if err == nil || err.Error() != "usage: digest uninstall notifications" {
			t.Errorf("%s: err = %v, want usage: digest uninstall notifications", name, err)
		}
	}
}

func TestExitForJobExitCodes(t *testing.T) {
	if outcome := os.Getenv("DIGEST_EXIT_FOR_JOB"); outcome != "" {
		var jobErr error
		if outcome == "error" {
			jobErr = os.ErrNotExist
		}
		exitForJob("janitor", outcome == "needs-action", jobErr)
		os.Exit(0)
	}
	cases := map[string]int{"clean": 0, "needs-action": 1, "error": 1}
	for outcome, wantCode := range cases {
		cmd := exec.Command(os.Args[0], "-test.run=^TestExitForJobExitCodes$")
		cmd.Env = append(os.Environ(), "DIGEST_EXIT_FOR_JOB="+outcome)
		output, _ := cmd.CombinedOutput()
		if gotCode := cmd.ProcessState.ExitCode(); gotCode != wantCode {
			t.Errorf("%s: exit code %d, want %d\n%s", outcome, gotCode, wantCode, output)
		}
		if outcome == "error" && !strings.Contains(string(output), "Error running janitor: "+os.ErrNotExist.Error()) {
			t.Errorf("error output = %q, want the job error", output)
		}
	}
}
