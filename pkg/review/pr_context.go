package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/achandrapaul/digest/pkg/system"
)

var runGHPR = func(ctx context.Context, args ...string) ([]byte, error) {
	output, err := system.GHCommand(ctx, append([]string{"pr"}, args...)...).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return output, err
}

const prContextFields = "title,body,author,state,baseRefName,headRefName,additions,deletions,files"

type prContextDetails struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State       string `json:"state"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Files       []struct {
		Path      string `json:"path"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
	} `json:"files"`
}

func FetchPRContext(ctx context.Context, ref PRRef, maxDiffBytes int) (string, error) {
	viewOutput, err := runGHPR(ctx, "view", ref.URL, "--json", prContextFields)
	if err != nil {
		return "", fmt.Errorf("gh pr view %s: %w", ref.URL, err)
	}
	var details prContextDetails
	if err := json.Unmarshal(viewOutput, &details); err != nil {
		return "", fmt.Errorf("gh pr view %s: %w", ref.URL, err)
	}
	diff, diffErr := runGHPR(ctx, "diff", ref.URL)
	var text strings.Builder
	fmt.Fprintf(&text, "### %s#%d %s\n\n%s\n", ref.NameWithOwner(), ref.Number, details.Title, ref.URL)
	fmt.Fprintf(&text, "Author %s · %s · %s ← %s · +%d −%d\n", details.Author.Login, details.State, details.BaseRefName, details.HeadRefName, details.Additions, details.Deletions)
	if body := strings.TrimSpace(details.Body); body != "" {
		fmt.Fprintf(&text, "\n#### Description\n\n%s\n", body)
	}
	if len(details.Files) > 0 {
		text.WriteString("\n#### Files\n\n")
		for _, file := range details.Files {
			fmt.Fprintf(&text, "- %s (+%d −%d)\n", file.Path, file.Additions, file.Deletions)
		}
	}
	if diffErr != nil {
		fmt.Fprintf(&text, "\n#### Diff\n\nUnavailable: gh pr diff %s: %v. Summarise from the description and the file list.\n", ref.URL, diffErr)
	} else {
		fmt.Fprintf(&text, "\n#### Diff\n\n```diff\n%s```\n", cutDiff(string(diff), maxDiffBytes))
	}
	return text.String(), nil
}

func cutDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		if diff != "" && !strings.HasSuffix(diff, "\n") {
			diff += "\n"
		}
		return diff
	}
	kept := diff[:maxBytes]
	if lineEnd := strings.LastIndex(kept, "\n"); lineEnd >= 0 {
		kept = kept[:lineEnd+1]
	}
	return kept + fmt.Sprintf("… diff cut at %d bytes\n", maxBytes)
}
