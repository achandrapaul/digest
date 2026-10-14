package focus

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system/process"
)

var pidAlive = process.PIDAlive

var signalTUI = func(pid int) error {
	return syscall.Kill(pid, syscall.SIGUSR1)
}

var runCommand = func(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, output)
	}
	return nil
}

func Run(cfg *config.Config, selector Selector) error {
	state, _, loadErr := appstate.Load(cfg.Root())
	location := state.TUILocation
	if location == nil || !pidAlive(location.PID) {
		return errors.Join(loadErr, raiseTerminal(cfg.TerminalApp))
	}
	var errs []error
	if err := Write(cfg.Root(), selector); err != nil {
		errs = append(errs, err)
	} else if err := signalTUI(location.PID); err != nil {
		errs = append(errs, err)
	}
	if location.TmuxSocket != "" && location.TmuxPane != "" {
		for _, command := range []string{"switch-client", "select-window", "select-pane"} {
			if err := runCommand("tmux", "-S", location.TmuxSocket, command, "-t", location.TmuxPane); err != nil {
				errs = append(errs, err)
			}
		}
	}
	terminal := location.Terminal
	if terminal == "" {
		terminal = cfg.TerminalApp
	}
	return errors.Join(append(errs, raiseTerminal(terminal))...)
}

func raiseTerminal(bundleID string) error {
	if bundleID == "" {
		return nil
	}
	return runCommand("open", "-b", bundleID)
}
