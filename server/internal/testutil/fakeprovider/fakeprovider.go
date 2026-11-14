// Package fakeprovider is a test double for search.Provider. It stands in for
// the external data sources (Yahoo, etf.com, Alpha Vantage) so tests control
// exactly which ETFs the pipeline sees.
package fakeprovider

import (
	"context"
	"sync"

	"upstonk/internal/domain"
	"upstonk/internal/service/search"
)

// Provider returns a fixed set of ETFs (or a fixed error) for every search.
type Provider struct {
	ETFs []domain.ETF
	Err  error

	mu    sync.Mutex
	calls int
}

// New returns a Provider that always returns etfs.
func New(etfs ...domain.ETF) *Provider {
	return &Provider{ETFs: etfs}
}

// Search implements search.Provider. It returns a copy so callers can't
// mutate the canned data between calls.
func (p *Provider) Search(_ context.Context, _ search.Criteria) ([]domain.ETF, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.Err != nil {
		return nil, p.Err
	}
	out := make([]domain.ETF, len(p.ETFs))
	copy(out, p.ETFs)
	return out, nil
}

// Calls reports how many times Search was invoked.
func (p *Provider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}
