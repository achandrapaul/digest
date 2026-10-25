package model

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var (
	linkPattern        = regexp.MustCompile(`https?://[^\s<>()"']+`)
	pullRequestPattern = regexp.MustCompile(`^https?://[^/]+/[^/]+/[^/]+/pull/\d+`)
)

func LinksIn(text string) []string {
	var links []string
	for _, link := range linkPattern.FindAllString(text, -1) {
		link = strings.TrimRight(link, ".,;:")
		if !slices.Contains(links, link) {
			links = append(links, link)
		}
	}
	return links
}

func PullRequestURLsIn(text string) []string {
	var pullRequests []string
	for _, link := range LinksIn(text) {
		pullRequest := pullRequestPattern.FindString(link)
		if pullRequest != "" && !slices.Contains(pullRequests, pullRequest) {
			pullRequests = append(pullRequests, pullRequest)
		}
	}
	return pullRequests
}

func PullRequestURLsOnHosts(text string, hosts []string) []string {
	var onHosts []string
	for _, pullRequestURL := range PullRequestURLsIn(text) {
		parsed, err := url.Parse(pullRequestURL)
		if err == nil && slices.ContainsFunc(hosts, func(host string) bool { return strings.EqualFold(host, parsed.Host) }) {
			onHosts = append(onHosts, pullRequestURL)
		}
	}
	return onHosts
}
