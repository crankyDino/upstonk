// service/search/interface.go
package search

import (
	"context"
	"strings"
	"upstonk/internal/domain"
)

// Provider defines the contract for ETF search implementations
type Provider interface {
	Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error)
}

// CacheReporter is implemented by providers that cache results. SearchCached
// behaves like Search and also reports whether the result came from cache.
type CacheReporter interface {
	SearchCached(ctx context.Context, criteria Criteria) (etfs []domain.ETF, cacheHit bool, err error)
}

type Criteria struct {
	Markets      []string
	Sectors      []string
	AssetClasses []string
	Companies    []string
	Country      string
	Vehicles     []string
}

// SourcesFailedError means every data source failed, so the search has no
// basis for an answer (as opposed to sources answering with no matches).
type SourcesFailedError struct {
	Errs []error
}

func (e *SourcesFailedError) Error() string {
	msgs := make([]string, len(e.Errs))
	for i, err := range e.Errs {
		msgs[i] = err.Error()
	}
	return "all data sources failed: " + strings.Join(msgs, "; ")
}
