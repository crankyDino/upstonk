// service/search/interface.go
package search

import (
	"context"
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
