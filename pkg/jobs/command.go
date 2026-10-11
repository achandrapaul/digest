package jobs

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system"
)

var builtinJobNames = map[string]bool{
	"repo-sync":     true,
	"janitor":       true,
	"branch-reaper": true,
}

var ErrNoCommand = errors.New("no command configured")

var ErrDryRunFlagMissing = errors.New("dry-run command does not pass --dry-run")

func digestExecutable() string {
	if execPath, err := os.Executable(); err == nil {
		return system.ShellQuote(execPath)
	}
	return "digest"
}

func HasDryRun(dryRunCommand string) bool {
	return dryRunCommand != ""
}

func withRunningDigest(command string, dryRun bool) (string, error) {
	words := strings.Fields(command)
	if len(words) < 2 || words[0] != "digest" || !builtinJobNames[words[1]] {
		return command, nil
	}
	if dryRun && !slices.ContainsFunc(words[2:], func(word string) bool { return word == "--dry-run" || word == "-dry-run" }) {
		return "", ErrDryRunFlagMissing
	}
	return digestExecutable() + strings.TrimPrefix(strings.TrimSpace(command), "digest"), nil
}

func ResolveCommand(spec config.JobSpec, dryRun bool) (string, error) {
	configured := spec.Command
	if dryRun {
		configured = spec.DryRunCommand
	}
	if configured != "" {
		command, err := withRunningDigest(configured, dryRun)
		if err != nil {
			return "", fmt.Errorf("job %q: %w", spec.Name, err)
		}
		return command, nil
	}
	if !builtinJobNames[spec.Name] {
		return "", fmt.Errorf("job %q: %w", spec.Name, ErrNoCommand)
	}
	command := fmt.Sprintf("%s %s", digestExecutable(), spec.Name)
	if dryRun {
		command += " --dry-run"
	}
	return command, nil
}

func FindSpec(cfg *config.Config, name string) config.JobSpec {
	if cfg != nil {
		for _, spec := range cfg.JobList() {
			if spec.Name == name {
				return spec
			}
		}
	}
	return config.JobSpec{Name: name}
}
