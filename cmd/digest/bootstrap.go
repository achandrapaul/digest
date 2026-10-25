package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/achandrapaul/digest/pkg/appstate"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/doctor"
	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/tui"
)

type bootstrapStage struct {
	name string
	run  func() error
	// once stages fail deterministically, so a retry would only repeat the same error
	once bool
}

func runStages(stages []bootstrapStage, out io.Writer) error {
	for index, stage := range stages {
		fmt.Fprintf(out, "\n[%d/%d] %s\n", index+1, len(stages), stage.name)
		err := stage.run()
		if err != nil && !stage.once {
			fmt.Fprintf(out, "✗ %s failed: %v\nRetrying %s…\n", stage.name, err, stage.name)
			err = stage.run()
		}
		if err != nil {
			again := " again"
			if stage.once {
				again = ""
			}
			fmt.Fprintf(out, "✗ %s failed%s: %v\n", stage.name, again, err)
			return fmt.Errorf("bootstrap stopped at %s: %w", stage.name, err)
		}
		fmt.Fprintf(out, "✓ %s\n", stage.name)
	}
	return nil
}

func initConfigStage(configPath string, out io.Writer) error {
	if config.Exists(configPath) {
		fmt.Fprintln(out, "Keeping "+config.Path(configPath))
		return nil
	}
	written, _, err := config.Init(configPath, false)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Wrote "+written)
	return nil
}

func initAppStateStage(root string, out io.Writer) error {
	statePath := appstate.Path(root)
	if system.Exists(statePath) {
		fmt.Fprintln(out, "Keeping "+statePath)
		return nil
	}
	if err := appstate.Save(root, appstate.State{}); err != nil {
		return err
	}
	fmt.Fprintln(out, "Wrote "+statePath)
	return nil
}

func doctorStage(cfg *config.Config, out io.Writer) error {
	results := doctor.Check(cfg, exec.LookPath)
	fmt.Fprintln(out, doctor.Format(results))
	if !doctor.Failed(results) {
		return nil
	}
	var missing []string
	for _, result := range results {
		if result.Required && !result.Found {
			missing = append(missing, result.Name)
		}
	}
	return errors.New("required tools missing: " + strings.Join(missing, ", "))
}

func configStage(configPath string, configErr error) bootstrapStage {
	return bootstrapStage{name: "config", once: true, run: func() error {
		if configErr != nil {
			return fmt.Errorf("fix %s and run digest bootstrap again: %w", config.Path(configPath), configErr)
		}
		return nil
	}}
}

func bootstrapStages(configPath string, cfg *config.Config, configErr error) []bootstrapStage {
	return []bootstrapStage{
		configStage(configPath, configErr),
		{name: "install", run: func() error { return runInstall(cfg) }},
		{name: "doctor", once: true, run: func() error { return doctorStage(cfg, os.Stdout) }},
		{name: "init", run: func() error {
			if err := initConfigStage(configPath, os.Stdout); err != nil {
				return err
			}
			cfg, configErr = config.LoadOrDefault(configPath)
			system.Protect(cfg.Root(), cfg.ReviewRootDir(), cfg.LogsDir(), config.Path(configPath))
			return initAppStateStage(cfg.Root(), os.Stdout)
		}},
		{name: "setup", run: func() error { return tui.Run(cfg, configErr, config.Path(configPath), true) }},
	}
}
