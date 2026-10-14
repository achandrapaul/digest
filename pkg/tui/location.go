package tui

import (
	"os"
	"strings"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/system"
)

func recordLocation(root string) error {
	if !system.Exists(appstate.Path(root)) {
		return nil
	}
	tmuxSocket, _, _ := strings.Cut(os.Getenv("TMUX"), ",")
	location := appstate.TUILocation{
		PID:        os.Getpid(),
		TmuxSocket: tmuxSocket,
		TmuxPane:   os.Getenv("TMUX_PANE"),
		Terminal:   os.Getenv("__CFBundleIdentifier"),
	}
	if location.TmuxSocket == "" {
		location.TmuxPane = ""
	}
	return appstate.Update(root, func(state *appstate.State) { state.TUILocation = &location })
}

func clearLocation(root string) error {
	if !system.Exists(appstate.Path(root)) {
		return nil
	}
	return appstate.Update(root, func(state *appstate.State) {
		if state.TUILocation != nil && state.TUILocation.PID == os.Getpid() {
			state.TUILocation = nil
		}
	})
}
