package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/achandrapaul/digest/pkg/system"
)

type DetachSpec struct {
	Label   string
	Args    []string
	Dir     string
	LogPath string
	PIDPath string
	OnExit  func(cmd *exec.Cmd)
}

func SpawnDetached(spec DetachSpec) error {
	logFile, err := system.OpenLog(spec.LogPath)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.Command(spec.Args[0], spec.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Dir = spec.Dir
	if err := cmd.Start(); err != nil {
		return err
	}
	pidErr := system.Write(spec.PIDPath, []byte(strconv.Itoa(cmd.Process.Pid)))
	go func() {
		_ = cmd.Wait()
		if spec.OnExit != nil {
			spec.OnExit(cmd)
		}
	}()
	if pidErr != nil {
		return fmt.Errorf("%s started but its pid file could not be written: %w", spec.Label, pidErr)
	}
	return nil
}
