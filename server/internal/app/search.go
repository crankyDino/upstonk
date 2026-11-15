package app

import (
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/service/search"
)

// NewSearchProvider returns the production search provider: live public
// sources, plus Alpha Vantage when a real key (not "demo") is supplied, all
// behind the aggregating cache. With a store (pool != nil), South African
// searches are answered from the stored JSE data instead.
func NewSearchProvider(alphaVantageKey string, pool *pgxpool.Pool) search.Provider {
	live := newLiveProvider(alphaVantageKey)
	if pool == nil {
		return live
	}
	log.Printf("Store enabled: ZA searches use stored JSE data")
	return search.ByCountry(map[string]search.Provider{"ZA": search.NewStoredProvider(pool)}, live)
}

func newLiveProvider(alphaVantageKey string) search.Provider {
	providers := []search.Provider{
		search.NewLiveProvider(), // Yahoo Finance, ETF.com, JSE
	}

	if alphaVantageKey != "" && alphaVantageKey != "demo" {
		providers = append(providers, search.NewAlphaVantageProvider(alphaVantageKey))
		log.Printf("Alpha Vantage provider enabled")
	} else {
		log.Printf("Alpha Vantage provider disabled (no API key). Get free key at https://www.alphavantage.co/support/#api-key")
	}

	return search.NewAggregatedProvider(providers...)
}
