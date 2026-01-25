package search

import (
	"context"
	"log"
	"sync"
	"time"
	"upstonk/internal/domain"
)

// AggregatedProvider combines multiple data sources for comprehensive ETF discovery
type AggregatedProvider struct {
	providers []Provider
	cache     *ETFCache
}

func NewAggregatedProvider(providers ...Provider) Provider {
	return &AggregatedProvider{
		providers: providers,
		cache:     NewETFCache(),
	}
}

func (a *AggregatedProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	// Check cache first
	cacheKey := a.cache.GenerateKey(criteria)
	if cached, found := a.cache.Get(cacheKey); found {
		log.Printf("Cache hit for criteria: %+v", criteria)
		return cached, nil
	}

	// Search all providers in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan []domain.ETF, len(a.providers))
	errorsChan := make(chan error, len(a.providers))

	for _, provider := range a.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			results, err := p.Search(ctx, criteria)
			if err != nil {
				log.Printf("Provider error: %v", err)
				errorsChan <- err
				return
			}

			resultsChan <- results
		}(provider)
	}

	// Wait for all providers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Aggregate results
	allETFs := make([]domain.ETF, 0)
	for results := range resultsChan {
		allETFs = append(allETFs, results...)
	}

	// Deduplicate and merge data from multiple sources
	merged := a.deduplicateAndMerge(allETFs)

	// Cache results
	a.cache.Set(cacheKey, merged)

	return merged, nil
}

// deduplicateAndMerge combines ETF data from multiple sources
func (a *AggregatedProvider) deduplicateAndMerge(etfs []domain.ETF) []domain.ETF {
	// Group by ticker/ISIN
	etfMap := make(map[string][]domain.ETF)

	for _, etf := range etfs {
		key := etf.Ticker
		if key == "" {
			key = etf.ISIN
		}

		etfMap[key] = append(etfMap[key], etf)
	}

	// Merge data for each ETF
	merged := make([]domain.ETF, 0, len(etfMap))

	for _, etfGroup := range etfMap {
		if len(etfGroup) == 1 {
			merged = append(merged, etfGroup[0])
			continue
		}

		// Merge multiple sources into single ETF with best data
		mergedETF := a.mergeETFData(etfGroup)
		merged = append(merged, mergedETF)
	}

	return merged
}

// mergeETFData combines data from multiple sources, preferring primary sources
func (a *AggregatedProvider) mergeETFData(etfs []domain.ETF) domain.ETF {
	// Start with first ETF as base
	merged := etfs[0]

	// Combine data sources
	allSources := make([]domain.DataSource, 0)
	for _, etf := range etfs {
		allSources = append(allSources, etf.DataSources...)
	}
	merged.DataSources = allSources

	// Take most complete data for each field
	for _, etf := range etfs[1:] {
		if merged.Name == "" && etf.Name != "" {
			merged.Name = etf.Name
		}
		if merged.ISIN == "" && etf.ISIN != "" {
			merged.ISIN = etf.ISIN
		}
		if merged.Provider == "" && etf.Provider != "" {
			merged.Provider = etf.Provider
		}
		if merged.TER == 0 && etf.TER != 0 {
			merged.TER = etf.TER
		}
		if merged.AUM == 0 && etf.AUM != 0 {
			merged.AUM = etf.AUM
		}
		if merged.TrackingIndex == "" && etf.TrackingIndex != "" {
			merged.TrackingIndex = etf.TrackingIndex
		}

		// Merge holdings (prefer longer list)
		if len(etf.TopHoldings) > len(merged.TopHoldings) {
			merged.TopHoldings = etf.TopHoldings
		}

		// Merge sector exposure (prefer more detailed)
		if len(etf.SectorExposure) > len(merged.SectorExposure) {
			merged.SectorExposure = etf.SectorExposure
		}

		// Merge geographic exposure
		for region, weight := range etf.GeographicExposure.Regions {
			if _, exists := merged.GeographicExposure.Regions[region]; !exists {
				if merged.GeographicExposure.Regions == nil {
					merged.GeographicExposure.Regions = make(map[string]float64)
				}
				merged.GeographicExposure.Regions[region] = weight
			}
		}
	}

	return merged
}

// ETFCache provides simple in-memory caching
type ETFCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
	ttl  int64 // seconds
}

type cacheEntry struct {
	etfs      []domain.ETF
	timestamp int64
}

func NewETFCache() *ETFCache {
	return &ETFCache{
		data: make(map[string]cacheEntry),
		ttl:  3600, // 1 hour
	}
}

func (c *ETFCache) GenerateKey(criteria Criteria) string {
	// Simple key generation - in production use proper hashing
	key := criteria.Country
	for _, sector := range criteria.Sectors {
		key += "_" + sector
	}
	for _, market := range criteria.Markets {
		key += "_" + market
	}
	return key
}

func (c *ETFCache) Get(key string) ([]domain.ETF, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if entry.timestamp+c.ttl < nowUnix() {
		return nil, false
	}

	return entry.etfs, true
}

func (c *ETFCache) Set(key string, etfs []domain.ETF) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		etfs:      etfs,
		timestamp: nowUnix(),
	}
}

func nowUnix() int64 {
	return time.Now().Unix()
}
