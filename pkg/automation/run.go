package automation

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/system/process"
)

const (
	runPIDFile  = "automation.pid"
	runExitFile = "automation.exit"
	runLogFile  = "automation.log"
	runMetaFile = "meta.json"
	draftFile   = "draft.yaml"
)

const ExitNeedsReauth = 3

type RunStatus int

const (
	RunIdle RunStatus = iota
	RunRunning
	RunFailed
	RunNeedsReauth
	RunDraftReady
	RunCreated
)

type RunMeta struct {
	NoteID     string `json:"note_id"`
	Automation string `json:"automation"`
	Phase      Phase  `json:"phase"`
}

type Run struct {
	Meta     RunMeta
	Status   RunStatus
	HasDraft bool
}

var ErrRunning = errors.New("this note's automation is already running")

var executablePath = os.Executable

func StateDir(root, noteID string) string {
	return filepath.Join(root, noteID)
}

func LogPath(root, noteID string) string {
	return filepath.Join(StateDir(root, noteID), runLogFile)
}

func running(root, noteID string) bool {
	pid, found := process.ReadInt(filepath.Join(StateDir(root, noteID), runPIDFile))
	return found && process.PIDAlive(pid)
}

func writeMeta(dir string, meta RunMeta) error {
	return system.WriteJSON(filepath.Join(dir, runMetaFile), meta)
}

func exitCodeFor(runErr error) int {
	switch {
	case runErr == nil:
		return 0
	case errors.Is(runErr, ErrNeedsReauth):
		return ExitNeedsReauth
	default:
		return 1
	}
}

func claimRun(root, noteID string) (func(runErr error), error) {
	if !validNoteID(root, noteID) {
		return nil, ErrInvalidNoteID
	}
	dir := StateDir(root, noteID)
	if err := system.MkdirAll(dir); err != nil {
		return nil, err
	}
	pidPath := filepath.Join(dir, runPIDFile)
	if pid, found := process.ReadInt(pidPath); found && process.PIDAlive(pid) && pid != os.Getpid() {
		return nil, ErrRunning
	}
	if err := system.Remove(filepath.Join(dir, runExitFile)); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := system.Write(pidPath, []byte(strconv.Itoa(os.Getpid()))); err != nil {
		return nil, err
	}
	return func(runErr error) {
		_ = system.Write(filepath.Join(dir, runExitFile), []byte(strconv.Itoa(exitCodeFor(runErr))))
		_ = system.Remove(pidPath)
	}, nil
}

func recordUnreportedExit(dir string, child *os.Process, state *os.ProcessState) {
	exitPath := filepath.Join(dir, runExitFile)
	if state == nil || !state.Exited() || system.Exists(exitPath) {
		return
	}
	_ = system.Write(exitPath, []byte(strconv.Itoa(state.ExitCode())))
	pidPath := filepath.Join(dir, runPIDFile)
	if pid, found := process.ReadInt(pidPath); found && pid == child.Pid {
		_ = system.Remove(pidPath)
	}
}

func Status(root, noteID string) Run {
	dir := StateDir(root, noteID)
	run := Run{Meta: RunMeta{NoteID: noteID}}
	_, _ = system.ReadJSON(filepath.Join(dir, runMetaFile), &run.Meta)
	run.HasDraft = system.Exists(filepath.Join(dir, draftFile))
	if running(root, noteID) {
		run.Status = RunRunning
		return run
	}
	exitCode, found := process.ReadInt(filepath.Join(dir, runExitFile))
	switch {
	case !found:
		run.Status = RunIdle
		if run.HasDraft {
			run.Status = RunDraftReady
		}
		if system.Exists(filepath.Join(dir, runPIDFile)) {
			run.Status = RunFailed
		}
	case exitCode == ExitNeedsReauth:
		run.Status = RunNeedsReauth
	case exitCode != 0:
		run.Status = RunFailed
	case run.Meta.Phase == PhaseCreate:
		run.Status = RunCreated
	default:
		run.Status = RunDraftReady
	}
	return run
}

func ListRuns(root string) map[string]Run {
	entries, err := system.List(root)
	if err != nil {
		return nil
	}
	runs := map[string]Run{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if run := Status(root, entry.Name()); run.Status != RunIdle {
			runs[entry.Name()] = run
		}
	}
	return runs
}

func StartBackground(root, noteID, automationName string, phase Phase) error {
	if running(root, noteID) {
		return ErrRunning
	}
	dir := StateDir(root, noteID)
	if err := system.MkdirAll(dir); err != nil {
		return err
	}
	stale := []string{runExitFile, runPIDFile}
	if phase == PhaseDraft {
		stale = append(stale, draftFile)
	}
	for _, staleFile := range stale {
		if err := system.Remove(filepath.Join(dir, staleFile)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := writeMeta(dir, RunMeta{NoteID: noteID, Automation: automationName, Phase: phase}); err != nil {
		return err
	}
	executable, err := executablePath()
	if err != nil {
		return fmt.Errorf("locate digest binary: %w", err)
	}
	return process.SpawnDetached(process.DetachSpec{
		Label:   "automation",
		Args:    []string{executable, "automation", "--note", noteID, "--name", automationName, "--phase", string(phase)},
		Dir:     dir,
		LogPath: LogPath(root, noteID),
		PIDPath: filepath.Join(dir, runPIDFile),
		OnExit:  func(cmd *exec.Cmd) { recordUnreportedExit(dir, cmd.Process, cmd.ProcessState) },
	})
}

func Stop(root, noteID string) error {
	dir := StateDir(root, noteID)
	pidPath := filepath.Join(dir, runPIDFile)
	pid, found := process.ReadInt(pidPath)
	if !found || !process.PIDAlive(pid) {
		return nil
	}
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("stop automation for %s: %w", noteID, err)
	}
	if err := system.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return system.Write(filepath.Join(dir, runExitFile), []byte("143"))
}

var ErrInvalidNoteID = errors.New("note id does not name a folder inside the automations root")

func validNoteID(root, noteID string) bool {
	return noteID != "" && filepath.Dir(StateDir(root, noteID)) == filepath.Clean(root) && filepath.Base(StateDir(root, noteID)) == noteID
}

func Dismiss(root, noteID string) error {
	if !validNoteID(root, noteID) {
		return ErrInvalidNoteID
	}
	if running(root, noteID) {
		return ErrRunning
	}
	return system.RemoveAll(StateDir(root, noteID))
}

func AnyRunning(root string) bool {
	entries, err := system.List(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() && running(root, entry.Name()) {
			return true
		}
	}
	return false
}
