package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/sourcecontrol"
)

type previewHeading struct {
	title  string
	status string
	tag    string
}

type itemKindHandler struct {
	key        func(item NavItem) string
	preview    func(m *Model, item NavItem, width, height int)
	stamp      func(m Model, item NavItem) (previewStamp, string)
	header     func(m Model, item NavItem, header *previewHeading)
	followsLog bool
}

var kindHandlers map[NavItemKind]itemKindHandler

var fallbackKind itemKindHandler

func init() {
	fallbackKind = noteKind()
	kindHandlers = map[NavItemKind]itemKindHandler{
		KindGitRepo:       gitRepoKind(),
		KindPendingGit:    pendingPRKind(),
		KindMyPR:          myPRKind(),
		KindJobDraft:      jobDraftKind(),
		KindReviewRun:     reviewRunKind(),
		KindBragRun:       bragRunKind(),
		KindAutomationRun: automationRunKind(),
		KindYesterdayDone: noteKind(),
		KindTodayNote:     noteKind(),
		KindTodayDone:     noteKind(),
		KindCarriedNote:   noteKind(),
	}
}

func kindHandlerFor(kind NavItemKind) itemKindHandler {
	if handler, known := kindHandlers[kind]; known {
		return handler
	}
	return fallbackKind
}

func markdownPreview(render func(m *Model, item NavItem, width int) string) func(m *Model, item NavItem, width, height int) {
	return func(m *Model, item NavItem, width, height int) {
		m.setPreviewContent(item, render(m, item, width), width, height)
	}
}

func backgroundRunPreview(m *Model, item NavItem, width, height int) {
	content := ""
	if run := backgroundRunFor(item); run != nil {
		content = run.Preview(*m, width)
	}
	m.setPreviewContent(item, content, width, height)
}

func gitRepoKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.GitRepo == nil {
				return ""
			}
			return "repo:" + item.GitRepo.Name
		},
		preview: markdownPreview(func(m *Model, item NavItem, width int) string {
			commitsLine := ""
			if m.cfg.DailyCommitsEnabled() {
				commitsLine = fmt.Sprintf("- Commits: %d\n", item.GitRepo.Commits)
			}
			return renderMarkdown(fmt.Sprintf("# Repository Activity: %s\n%s- PRs Reviewed: %d\n- PRs Assigned: %d\n\nPress **[Tab]** on this repository item to view and open PRs directly in your browser.", item.GitRepo.Name, commitsLine, item.GitRepo.Reviewed, item.GitRepo.Assigned), width)
		}),
		header: func(m Model, item NavItem, header *previewHeading) {
			header.title = fmt.Sprintf(" GIT REPO: %s ", item.GitRepo.Name)
			header.status = badgeActive.Render("SYNCED")
			header.tag = tagStyle.Render("#git")
		},
	}
}

func pendingPRKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.PendingGitPR == nil {
				return ""
			}
			return "pr:" + item.PendingGitPR.URL
		},
		preview: func(m *Model, item NavItem, width, height int) {
			if item.PendingGitPR == nil {
				m.setPreviewContent(item, "", width, height)
				return
			}
			m.setPRPreviewContent(item.PendingGitPR, width, height)
		},
		stamp: func(m Model, item NavItem) (previewStamp, string) {
			if item.PendingGitPR == nil {
				return previewStamp{}, ""
			}
			var stamp previewStamp
			var logPath string
			stamp.review, stamp.reviewFound = m.previewedLocalReview()
			if queued, err := queuedFor(item.PendingGitPR); err == nil {
				logPath = filepath.Join(review.StateDir(m.reviewRoot(), queued.Ref), review.LogFile)
			}
			return stamp, logPath
		},
		header: func(m Model, item NavItem, header *previewHeading) {
			if item.PendingGitPR == nil {
				return
			}
			header.title = fmt.Sprintf(" PENDING PR REVIEW: %s ", item.PendingGitPR.Repository)
			if item.PendingGitPR.Kind == sourcecontrol.ReReviewKind {
				header.title = fmt.Sprintf(" RE-REVIEW: %s ", item.PendingGitPR.Repository)
			}
			header.status = m.prStateBadge(item.PendingGitPR)
			if pid, running := m.reviewPIDFor(item.PendingGitPR); running {
				header.status = badgeActive.Render(fmt.Sprintf("RUNNING · PID %d", pid))
			}
			header.tag = tagStyle.Render("#github")
		},
	}
}

func myPRKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.MyPR == nil {
				return ""
			}
			return "mine:" + item.MyPR.Ref.URL
		},
		preview: markdownPreview(func(m *Model, item NavItem, width int) string {
			if item.MyPR == nil {
				return ""
			}
			return renderMarkdown(myPRDetailsMarkdown(*item.MyPR), width)
		}),
		header: func(m Model, item NavItem, header *previewHeading) {
			if item.MyPR == nil {
				return
			}
			header.title = fmt.Sprintf(" MY PR: %s #%d ", item.MyPR.Ref.Repo, item.MyPR.Ref.Number)
			header.status = badgeActive.Render(strings.ToUpper(myPRCIText(item.MyPR.CIState)))
			header.tag = tagStyle.Render("#my-pr")
		},
	}
}

func jobDraftKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.Draft == nil {
				return ""
			}
			return "job:" + item.Draft.Name
		},
		preview: markdownPreview(func(m *Model, item NavItem, width int) string {
			if item.Draft == nil {
				return ""
			}
			m.refreshJobStates()
			return m.jobPreviewContent(item.Draft, loadJobOutput(item.Draft.Name, m.jobDryRunOutputFor(item.Draft.Name)), width)
		}),
		header: func(m Model, item NavItem, header *previewHeading) {
			header.title = fmt.Sprintf(" JOB: %s ", item.Draft.Name)
			switch {
			case m.jobs.runningJobPIDs[item.Draft.Name] > 0:
				header.status = badgeActive.Render(fmt.Sprintf("RUNNING · PID %d", m.jobs.runningJobPIDs[item.Draft.Name]))
			case item.Draft.DryRunInFlight:
				header.status = badgeActive.Render("DRY RUNNING")
			case m.jobs.previewJobLogFinished:
				header.status = badgeDone.Render("FINISHED")
			case item.Draft.HasRunDryRun && item.Draft.ExitCode == 0:
				header.status = badgeDone.Render("SUCCESS")
			case item.Draft.HasRunDryRun:
				header.status = staleStyle.Render("NEED ACT")
			default:
				header.status = badgeActive.Render("IDLE")
			}
			header.tag = tagStyle.Render("#job")
		},
		followsLog: true,
	}
}

func reviewRunKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.ReviewRun == nil {
				return ""
			}
			return "review:" + item.ReviewRun.Meta.Ref.URL
		},
		preview: backgroundRunPreview,
		stamp: func(m Model, item NavItem) (previewStamp, string) {
			if item.ReviewRun == nil {
				return previewStamp{}, ""
			}
			return previewStamp{runStatus: int(item.ReviewRun.Status)}, filepath.Join(review.StateDir(m.reviewRoot(), item.ReviewRun.Meta.Ref), review.LogFile)
		},
		header: func(m Model, item NavItem, header *previewHeading) {
			header.title = fmt.Sprintf(" REVIEW JOB: %s ", reviewRunLabel(*item.ReviewRun))
			header.status = stateStyle(review.StateFailed).Render("FAILED")
			if state := m.review.localReviews[review.StateDir(m.reviewRoot(), item.ReviewRun.Meta.Ref)]; state.pid > 0 {
				header.status = badgeActive.Render(fmt.Sprintf("RUNNING · PID %d", state.pid))
			}
			header.tag = tagStyle.Render("#github")
		},
		followsLog: true,
	}
}

func bragRunKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.BragRun == nil {
				return ""
			}
			return "brag:" + item.BragRun.Meta.ID
		},
		preview: backgroundRunPreview,
		stamp: func(m Model, item NavItem) (previewStamp, string) {
			if item.BragRun == nil {
				return previewStamp{}, ""
			}
			return previewStamp{runStatus: int(item.BragRun.Status)}, filepath.Join(brag.StateDir(m.bragRoot(), item.BragRun.Meta.ID), brag.RunLogFile)
		},
		header: func(m Model, item NavItem, header *previewHeading) {
			header.title = fmt.Sprintf(" BRAG JOB: %s ", item.BragRun.Meta.ID)
			header.status = stateStyle(review.StateFailed).Render("FAILED")
			if item.BragRun.Status == brag.RunRunning {
				header.status = badgeActive.Render("RUNNING")
			}
			header.tag = tagStyle.Render("#brag")
		},
		followsLog: true,
	}
}

func automationRunKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.AutomationRun == nil {
				return ""
			}
			return automationNavKeyPrefix + item.AutomationRun.Meta.NoteID
		},
		preview: backgroundRunPreview,
		stamp: func(m Model, item NavItem) (previewStamp, string) {
			if item.AutomationRun == nil {
				return previewStamp{}, ""
			}
			return previewStamp{runStatus: int(item.AutomationRun.Status)}, automation.LogPath(m.automationRoot(), item.AutomationRun.Meta.NoteID)
		},
		header: func(m Model, item NavItem, header *previewHeading) {
			header.title = fmt.Sprintf(" AUTOMATION JOB: %s ", item.AutomationRun.Meta.Automation)
			header.status = stateStyle(review.StateFailed).Render("FAILED")
			if m.automationJobRunning(item.AutomationRun) {
				header.status = badgeActive.Render("RUNNING")
			}
			header.tag = tagStyle.Render("#" + string(item.AutomationRun.Meta.Phase))
		},
		followsLog: true,
	}
}

func noteKind() itemKindHandler {
	return itemKindHandler{
		key: func(item NavItem) string {
			if item.Note == nil {
				return ""
			}
			return fmt.Sprintf("note:%d:%s", item.Kind, item.Note.ID)
		},
		preview: markdownPreview(func(m *Model, item NavItem, width int) string {
			switch {
			case item.Note == nil:
				return ""
			case m.onDraftTab():
				return renderMarkdown(m.draftTabMarkdown(item.Note), width)
			}
			fullText := fmt.Sprintf("# %s", item.Note.Summary)
			if strings.TrimSpace(item.Note.Body) != "" {
				fullText += fmt.Sprintf("\n\n%s", item.Note.Body)
			}
			return renderMarkdown(fullText, width)
		}),
		header: func(m Model, item NavItem, header *previewHeading) {
			if item.Note == nil {
				return
			}
			header.title = " PREVIEW NOTE "
			if item.Note.Status == model.StatusDone {
				header.status = badgeDone.Render("DONE")
			}
			if item.Note.Subject != "" {
				header.tag = tagStyle.Render("#" + item.Note.Subject)
			}
		},
	}
}
