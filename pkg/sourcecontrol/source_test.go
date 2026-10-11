package sourcecontrol

import (
	"context"
	"io"
	"testing"

	"github.com/achandrapaul/digest/pkg/review"

	"github.com/charmbracelet/log"
)

type fakePRSource struct {
	githubSource
	hosts  func(ctx context.Context) ([]string, error)
	search func(ctx context.Context, host string, searches []review.PRSearch) ([]review.SearchResult, error)
	states func(ctx context.Context, refs []review.PRRef) (map[string]string, error)
}

func (fake *fakePRSource) Hosts(ctx context.Context) ([]string, error) {
	if fake.hosts != nil {
		return fake.hosts(ctx)
	}
	return fake.githubSource.Hosts(ctx)
}

func (fake *fakePRSource) Search(ctx context.Context, host string, searches []review.PRSearch) ([]review.SearchResult, error) {
	if fake.search != nil {
		return fake.search(ctx, host, searches)
	}
	return fake.githubSource.Search(ctx, host, searches)
}

func (fake *fakePRSource) States(ctx context.Context, refs []review.PRRef) (map[string]string, error) {
	if fake.states != nil {
		return fake.states(ctx, refs)
	}
	return fake.githubSource.States(ctx, refs)
}

func (fake *fakePRSource) Prepare(ctx context.Context, ref review.PRRef, root string, logger *log.Logger, output io.Writer) (string, error) {
	return fake.githubSource.Prepare(ctx, ref, root, logger, output)
}

func useFakePRSource(t *testing.T) *fakePRSource {
	t.Helper()
	if fake, ok := prSource.(*fakePRSource); ok {
		return fake
	}
	fake := &fakePRSource{}
	original := prSource
	prSource = fake
	t.Cleanup(func() { prSource = original })
	return fake
}
