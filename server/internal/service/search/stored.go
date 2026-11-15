package search

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/domain"
	"upstonk/internal/store/storedb"
)

// StoredProvider serves ETFs from the Postgres store that the ingest jobs
// fill (JSE listing, issuer factsheets, daily prices). Every value it returns
// carries the source and as-of date it was stored with; a value no source
// reported stays empty.
type StoredProvider struct {
	q *storedb.Queries
}

func NewStoredProvider(pool *pgxpool.Pool) *StoredProvider {
	return &StoredProvider{q: storedb.New(pool)}
}

func (p *StoredProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	listed, err := p.q.ListActiveETFs(ctx)
	if err != nil {
		return nil, fmt.Errorf("stored ETFs: %w", err)
	}
	sheets, err := p.q.ListLatestFactsheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("stored factsheets: %w", err)
	}
	latest := make(map[string]storedb.ListLatestFactsheetsRow, len(sheets))
	ids := make([]int64, 0, len(sheets))
	for _, f := range sheets {
		latest[f.EtfCode] = f
		ids = append(ids, f.ID)
	}
	holdings, err := p.q.ListHoldingsFor(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("stored holdings: %w", err)
	}
	holdingsBy := map[int64][]domain.Holding{}
	for _, h := range holdings {
		holdingsBy[h.FactsheetID] = append(holdingsBy[h.FactsheetID], domain.Holding{
			Name: h.Name, Ticker: deref(h.Ticker), Weight: deref(h.WeightPercent),
		})
	}
	exposures, err := p.q.ListExposuresFor(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("stored exposures: %w", err)
	}
	sectorsBy := map[int64][]domain.SectorAllocation{}
	regionsBy := map[int64]map[string]float64{}
	countriesBy := map[int64]map[string]float64{}
	for _, x := range exposures {
		switch x.Kind {
		case "sector":
			sectorsBy[x.FactsheetID] = append(sectorsBy[x.FactsheetID], domain.SectorAllocation{Sector: x.Name, Percentage: x.WeightPercent})
		case "region":
			if regionsBy[x.FactsheetID] == nil {
				regionsBy[x.FactsheetID] = map[string]float64{}
			}
			regionsBy[x.FactsheetID][x.Name] = x.WeightPercent
		case "country":
			if countriesBy[x.FactsheetID] == nil {
				countriesBy[x.FactsheetID] = map[string]float64{}
			}
			countriesBy[x.FactsheetID][x.Name] = x.WeightPercent
		}
	}
	volStats, err := p.q.ListVolumeStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("stored prices: %w", err)
	}
	volumes := make(map[string]storedb.ListVolumeStatsRow, len(volStats))
	for _, v := range volStats {
		volumes[v.EtfCode] = v
	}

	etfs := make([]domain.ETF, 0, len(listed))
	for _, e := range listed {
		etf := domain.ETF{
			Ticker:   e.Code,
			Name:     e.Name,
			ISIN:     deref(e.Isin),
			Provider: deref(e.Issuer),
			Exchange: e.Exchange,
			Currency: deref(e.Currency),
			DataSources: []domain.DataSource{{
				Type: "ExchangeListing", Provider: e.ListingSource, URL: deref(e.ListingUrl),
				AccessDate: dateOf(e.ListedAsOf), Reliability: "Primary",
			}},
		}
		if e.Exchange == "JSE" {
			etf.ExchangeCountry = "ZA"
		}
		if f, ok := latest[e.Code]; ok {
			applyFactsheet(&etf, f)
			etf.TopHoldings = holdingsBy[f.ID]
			etf.SectorExposure = sectorsBy[f.ID]
			etf.GeographicExposure = domain.GeographicExposure{Regions: regionsBy[f.ID], Countries: countriesBy[f.ID]}
		}
		if v, ok := volumes[e.Code]; ok {
			etf.AverageDailyVolume = v.AvgVolume
			etf.DataSources = append(etf.DataSources, domain.DataSource{
				Type: "API", Provider: v.SourceName, URL: v.SourceUrl,
				AccessDate: dateOf(v.ToDate), Reliability: "Secondary",
			})
		}
		etfs = append(etfs, etf)
	}
	return etfs, nil
}

// applyFactsheet copies a factsheet's facts onto the ETF. NULL stays unknown:
// numbers stay 0 and structure flags stay nil.
func applyFactsheet(etf *domain.ETF, f storedb.ListLatestFactsheetsRow) {
	etf.TER = deref(f.TerPercent)
	etf.AUM = deref(f.Aum)
	etf.TrackingIndex = deref(f.TrackedIndex)
	etf.AssetClass = deref(f.AssetClass)
	etf.Domicile = deref(f.Domicile)
	etf.LegalStructure = deref(f.LegalStructure)
	etf.IsLeveraged = f.IsLeveraged
	etf.IsInverse = f.IsInverse
	if f.Replication != nil {
		etf.IsPhysical = domain.Flag(*f.Replication == "physical")
		etf.IsSynthetic = domain.Flag(*f.Replication == "synthetic")
	}
	etf.DataSources = append(etf.DataSources, domain.DataSource{
		Type: "FactSheet", Provider: f.SourceName, URL: f.SourceUrl,
		AccessDate: dateOf(f.AsOf), Reliability: "Primary",
	})
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func dateOf(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
