package app

import (
	"log"

	"upstonk/internal/service/search"
)

// NewSearchProvider returns the production search provider: live public
// sources, plus Alpha Vantage when a real key (not "demo") is supplied, all
// behind the aggregating cache.
func NewSearchProvider(alphaVantageKey string) search.Provider {
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
