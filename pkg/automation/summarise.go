package automation

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/store"
)

const (
	maxSummarisedPRs = 10
	maxPRDiffBytes   = 60 * 1024
	prSummaryHeading = "## PR summary"
)

var (
	fetchPRContext = review.FetchPRContext
	lookupGHHosts  = review.GHHosts
)

func isPRSummary(spec config.AutomationSpec) bool {
	return spec.Type == config.AutomationTypePRSummary
}

func pullRequestsInput(ctx context.Context, note *model.Note) (string, error) {
	ghHosts, err := lookupGHHosts(ctx)
	if err != nil {
		return "", err
	}
	pullRequestURLs := model.PullRequestURLsOnHosts(note.Summary+"\n"+note.Body, ghHosts)
	if len(pullRequestURLs) == 0 {
		return "", fmt.Errorf("no pull request link on a host gh is logged in to (%s)", strings.Join(ghHosts, ", "))
	}
	var text strings.Builder
	text.WriteString("\n## Pull requests\n\n")
	if len(pullRequestURLs) > maxSummarisedPRs {
		fmt.Fprintf(&text, "Only the first %d of %d pull requests are included.\n\n", maxSummarisedPRs, len(pullRequestURLs))
		pullRequestURLs = pullRequestURLs[:maxSummarisedPRs]
	}
	var failures []error
	for _, pullRequestURL := range pullRequestURLs {
		heading := pullRequestURL
		ref, err := review.ParsePRURL(pullRequestURL)
		if err == nil {
			heading = fmt.Sprintf("%s#%d", ref.NameWithOwner(), ref.Number)
			var block string
			if block, err = fetchPRContext(ctx, ref, maxPRDiffBytes); err == nil {
				text.WriteString(block + "\n")
				continue
			}
		}
		failures = append(failures, err)
		fmt.Fprintf(&text, "### %s\n\nUnavailable: %v\n\n", heading, err)
	}
	if len(failures) == len(pullRequestURLs) {
		return "", fmt.Errorf("could not fetch any pull request: %w", errors.Join(failures...))
	}
	return text.String(), nil
}

func replaceSection(body, heading, section string) string {
	lines := strings.Split(body, "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = index
			break
		}
	}
	if start < 0 {
		return joinSections(body, section)
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(lines[index], "## ") || strings.HasPrefix(lines[index], "# ") {
			end = index
			break
		}
	}
	return joinSections(strings.Join(lines[:start], "\n"), section, strings.Join(lines[end:], "\n"))
}

func joinSections(parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n\n")
}

func writePRSummary(noteStore *store.NoteStore, note *model.Note, root, draft string, stdout, stderr io.Writer) error {
	fields, err := ParseDraft(draft)
	if err != nil {
		return err
	}
	summary, _ := fields["summary"].(string)
	description, _ := fields["description"].(string)
	section := prSummaryHeading + "\n\n" + strings.TrimSpace(cmp.Or(strings.TrimSpace(description), summary))
	note.Body = replaceSection(note.Body, prSummaryHeading, section)
	note.Updated = time.Now()
	if err := saveNoteAndDropDraft(noteStore, note, root, "PR summary written", stderr); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Wrote the PR summary: %s\n", summary)
	return nil
}
