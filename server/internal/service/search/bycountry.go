package search

import (
	"context"
	"strings"

	"upstonk/internal/domain"
)

// countryRouter sends each search to the provider registered for the
// investor's country, or to the fallback.
type countryRouter struct {
	byCountry map[string]Provider
	fallback  Provider
}

// ByCountry routes searches by Criteria.Country (case-insensitive), e.g. ZA
// to the stored JSE data and everything else to the live sources.
func ByCountry(byCountry map[string]Provider, fallback Provider) Provider {
	m := make(map[string]Provider, len(byCountry))
	for c, p := range byCountry {
		m[strings.ToUpper(c)] = p
	}
	return &countryRouter{byCountry: m, fallback: fallback}
}

func (r *countryRouter) pick(country string) Provider {
	if p, ok := r.byCountry[strings.ToUpper(country)]; ok {
		return p
	}
	return r.fallback
}

func (r *countryRouter) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	return r.pick(criteria.Country).Search(ctx, criteria)
}

// SearchCached passes cache reporting through from the chosen provider.
func (r *countryRouter) SearchCached(ctx context.Context, criteria Criteria) ([]domain.ETF, bool, error) {
	p := r.pick(criteria.Country)
	if cr, ok := p.(CacheReporter); ok {
		return cr.SearchCached(ctx, criteria)
	}
	etfs, err := p.Search(ctx, criteria)
	return etfs, false, err
}
