package review

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const prViewJSON = `{"title":"Add retries","body":"Retries failed uploads.","author":{"login":"alice"},"state":"OPEN","baseRefName":"main","headRefName":"retries","additions":12,"deletions":3,"files":[{"path":"upload.go","additions":10,"deletions":3},{"path":"upload_test.go","additions":2,"deletions":0}]}`

func fakePRCommand(t *testing.T, diff string, err error) *[][]string {
	t.Helper()
	var calls [][]string
	original := runGHPR
	runGHPR = func(ctx context.Context, args ...string) ([]byte, error) {
		calls = append(calls, args)
		if err != nil {
			return nil, err
		}
		if args[0] == "diff" {
			return []byte(diff), nil
		}
		return []byte(prViewJSON), nil
	}
	t.Cleanup(func() { runGHPR = original })
	return &calls
}

func TestFetchPRContextDescribesThePullRequestAndItsDiff(t *testing.T) {
	calls := fakePRCommand(t, "diff --git a/upload.go b/upload.go\n+retry()\n", nil)
	ref := PRRef{Host: "github.com", Owner: "acme", Repo: "api", Number: 12, URL: "https://github.com/acme/api/pull/12"}
	text, err := FetchPRContext(context.Background(), ref, 1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"### acme/api#12 Add retries",
		"https://github.com/acme/api/pull/12",
		"alice", "OPEN", "main ← retries", "+12 −3",
		"Retries failed uploads.",
		"- upload.go (+10 −3)",
		"```diff\ndiff --git a/upload.go b/upload.go\n+retry()\n```",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context is missing %q:\n%s", want, text)
		}
	}
	if len(*calls) != 2 || (*calls)[0][0] != "view" || (*calls)[0][1] != ref.URL || (*calls)[1][0] != "diff" || (*calls)[1][1] != ref.URL {
		t.Errorf("gh calls = %v", *calls)
	}
}

func TestFetchPRContextCutsALongDiffOnALineBreak(t *testing.T) {
	fakePRCommand(t, "line one\nline two\nline three\n", nil)
	text, err := FetchPRContext(context.Background(), PRRef{Owner: "acme", Repo: "api", Number: 1, URL: "u"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "line one\nline two\n… diff cut at 20 bytes\n```") || strings.Contains(text, "line three") {
		t.Errorf("context:\n%s", text)
	}
}

func TestFetchPRContextReportsTheGHError(t *testing.T) {
	fakePRCommand(t, "", errors.New("HTTP 404"))
	if _, err := FetchPRContext(context.Background(), PRRef{URL: "https://github.com/acme/api/pull/9"}, 10); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("err = %v", err)
	}
}

func TestFetchPRContextKeepsThePullRequestWhenOnlyTheDiffFails(t *testing.T) {
	original := runGHPR
	runGHPR = func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "diff" {
			return nil, errors.New("HTTP 406: the diff exceeded the maximum number of files (300)")
		}
		return []byte(prViewJSON), nil
	}
	t.Cleanup(func() { runGHPR = original })
	text, err := FetchPRContext(context.Background(), PRRef{Owner: "acme", Repo: "api", Number: 12, URL: "https://github.com/acme/api/pull/12"}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"### acme/api#12 Add retries",
		"Retries failed uploads.",
		"- upload.go (+10 −3)",
		"#### Diff\n\nUnavailable: gh pr diff https://github.com/acme/api/pull/12: HTTP 406: the diff exceeded the maximum number of files (300). Summarise from the description and the file list.\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context is missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "```diff") {
		t.Errorf("an unavailable diff should have no diff block:\n%s", text)
	}
}
