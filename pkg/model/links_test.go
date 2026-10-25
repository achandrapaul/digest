package model

import (
	"slices"
	"testing"
)

func TestLinksInKeepsOrderTrimsPunctuationAndDedups(t *testing.T) {
	text := "see https://example.com/a, then (https://example.com/b).\nagain https://example.com/a; done"
	want := []string{"https://example.com/a", "https://example.com/b"}
	if got := LinksIn(text); !slices.Equal(got, want) {
		t.Errorf("links = %v, want %v", got, want)
	}
	if got := LinksIn("no links here"); got != nil {
		t.Errorf("links = %v, want none", got)
	}
}

func TestPullRequestURLsInCutsEachLinkToItsPullRequest(t *testing.T) {
	text := "fetch pr https://github.com/acme/api/pull/12/files and https://github.com/acme/web/pull/7.\n" +
		"docs https://example.com/pulls/3 and again https://github.com/acme/api/pull/12#discussion_r1"
	want := []string{"https://github.com/acme/api/pull/12", "https://github.com/acme/web/pull/7"}
	if got := PullRequestURLsIn(text); !slices.Equal(got, want) {
		t.Errorf("pull requests = %v, want %v", got, want)
	}
}

func TestPullRequestURLsOnHostsKeepsOnlyKnownHosts(t *testing.T) {
	text := "https://github.com/acme/api/pull/1 https://code.example.com/acme/web/pull/2 https://gitlab.example.com/x/y/pull/3"
	want := []string{"https://github.com/acme/api/pull/1", "https://code.example.com/acme/web/pull/2"}
	if got := PullRequestURLsOnHosts(text, []string{"github.com", "CODE.example.com"}); !slices.Equal(got, want) {
		t.Errorf("pull requests = %v, want %v", got, want)
	}
	if got := PullRequestURLsOnHosts(text, nil); got != nil {
		t.Errorf("no hosts should keep nothing, got %v", got)
	}
}
