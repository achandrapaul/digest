package focus

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/config"
)

type focusCalls struct {
	commands []string
	signaled []int
}

func stubFocus(t *testing.T, alive bool) *focusCalls {
	t.Helper()
	calls := &focusCalls{}
	originalAlive, originalSignal, originalRun := pidAlive, signalTUI, runCommand
	t.Cleanup(func() { pidAlive, signalTUI, runCommand = originalAlive, originalSignal, originalRun })
	pidAlive = func(int) bool { return alive }
	signalTUI = func(pid int) error {
		calls.signaled = append(calls.signaled, pid)
		return nil
	}
	runCommand = func(name string, args ...string) error {
		calls.commands = append(calls.commands, name+" "+strings.Join(args, " "))
		return nil
	}
	return calls
}

func focusConfig(t *testing.T, location *appstate.TUILocation) *config.Config {
	t.Helper()
	cfg := &config.Config{DigestRoot: t.TempDir(), TerminalApp: "com.apple.Terminal"}
	if err := appstate.Save(cfg.Root(), appstate.State{TUILocation: location}); err != nil {
		t.Fatal(err)
	}
	return cfg
}

var reviewedPR = Selector{Org: "acme", Repo: "control", Title: "fix: header"}

func TestFocusInTmuxSwitchesToThePaneAndRaisesTheTerminal(t *testing.T) {
	calls := stubFocus(t, true)
	cfg := focusConfig(t, &appstate.TUILocation{PID: 55, TmuxSocket: "/tmp/tmux-501/default", TmuxPane: "%7", Terminal: "com.mitchellh.ghostty"})
	if err := Run(cfg, reviewedPR); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"tmux -S /tmp/tmux-501/default switch-client -t %7",
		"tmux -S /tmp/tmux-501/default select-window -t %7",
		"tmux -S /tmp/tmux-501/default select-pane -t %7",
		"open -b com.mitchellh.ghostty",
	}
	if !reflect.DeepEqual(calls.commands, want) {
		t.Errorf("commands = %q", calls.commands)
	}
	if !reflect.DeepEqual(calls.signaled, []int{55}) {
		t.Errorf("signaled = %v", calls.signaled)
	}
	if taken, found, _ := Take(cfg.Root()); !found || taken != reviewedPR {
		t.Errorf("focus file = %+v, %v", taken, found)
	}
}

func TestFocusOnABareTerminalOnlyRaisesTheApp(t *testing.T) {
	calls := stubFocus(t, true)
	cfg := focusConfig(t, &appstate.TUILocation{PID: 55, Terminal: "com.mitchellh.ghostty"})
	if err := Run(cfg, reviewedPR); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls.commands, []string{"open -b com.mitchellh.ghostty"}) || len(calls.signaled) != 1 {
		t.Errorf("commands = %q, signaled = %v", calls.commands, calls.signaled)
	}
}

func TestFocusFallsBackToTheConfiguredTerminal(t *testing.T) {
	calls := stubFocus(t, true)
	cfg := focusConfig(t, &appstate.TUILocation{PID: 55})
	if err := Run(cfg, reviewedPR); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls.commands, []string{"open -b com.apple.Terminal"}) {
		t.Errorf("commands = %q", calls.commands)
	}
}

func TestFocusWithoutARunningDigestOnlyRaisesTheTerminal(t *testing.T) {
	for name, location := range map[string]*appstate.TUILocation{
		"no location": nil,
		"stale pid":   {PID: 55, TmuxSocket: "/tmp/s", TmuxPane: "%1", Terminal: "com.mitchellh.ghostty"},
	} {
		t.Run(name, func(t *testing.T) {
			calls := stubFocus(t, false)
			cfg := focusConfig(t, location)
			if err := Run(cfg, reviewedPR); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls.commands, []string{"open -b com.apple.Terminal"}) || len(calls.signaled) != 0 {
				t.Errorf("commands = %q, signaled = %v", calls.commands, calls.signaled)
			}
			if _, found, _ := Take(cfg.Root()); found {
				t.Error("no focus file should be written when digest isn't running")
			}
		})
	}
}

func TestFocusKeepsRaisingWhenTmuxFails(t *testing.T) {
	calls := stubFocus(t, true)
	runCommand = func(name string, args ...string) error {
		calls.commands = append(calls.commands, name)
		if name == "tmux" {
			return errors.New("no current client")
		}
		return nil
	}
	cfg := focusConfig(t, &appstate.TUILocation{PID: 55, TmuxSocket: "/tmp/s", TmuxPane: "%1", Terminal: "com.mitchellh.ghostty"})
	if err := Run(cfg, reviewedPR); err == nil || !strings.Contains(err.Error(), "no current client") {
		t.Errorf("err = %v", err)
	}
	if calls.commands[len(calls.commands)-1] != "open" {
		t.Errorf("terminal not raised: %q", calls.commands)
	}
}
