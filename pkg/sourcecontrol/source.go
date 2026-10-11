package sourcecontrol

import (
	"context"
	"io"

	"github.com/achandrapaul/digest/pkg/review"

	"github.com/charmbracelet/log"
)

type PRSource interface {
	Hosts(ctx context.Context) ([]string, error)
	Search(ctx context.Context, host string, searches []review.PRSearch) ([]review.SearchResult, error)
	States(ctx context.Context, refs []review.PRRef) (map[string]string, error)
	Prepare(ctx context.Context, ref review.PRRef, root string, logger *log.Logger, output io.Writer) (string, error)
}

type githubSource struct{}

func (githubSource) Hosts(ctx context.Context) ([]string, error) {
	return review.GHHosts(ctx)
}

func (githubSource) Search(ctx context.Context, host string, searches []review.PRSearch) ([]review.SearchResult, error) {
	return review.RunPRSearches(ctx, host, searches)
}

func (githubSource) States(ctx context.Context, refs []review.PRRef) (map[string]string, error) {
	return review.FetchPRStates(ctx, refs)
}

func (githubSource) Prepare(ctx context.Context, ref review.PRRef, root string, logger *log.Logger, output io.Writer) (string, error) {
	return review.PrepareTo(ctx, ref, root, logger, output)
}

var prSource PRSource = githubSource{}
