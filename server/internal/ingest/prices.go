package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/store/storedb"
)

// PricesReport says what a price run stored and which ETFs failed.
type PricesReport struct {
	Stored int               // trading-day rows written
	Failed map[string]string // ETF code -> reason; nothing is stored for these
}

// Prices fetches about three months of daily closes and volumes from Yahoo's
// chart endpoint for every active ETF and upserts them (re-running is safe).
func Prices(ctx context.Context, q *storedb.Queries, f Fetcher) (PricesReport, error) {
	report := PricesReport{Failed: map[string]string{}}
	etfs, err := q.ListActiveETFs(ctx)
	if err != nil {
		return report, fmt.Errorf("list ETFs: %w", err)
	}
	for _, e := range etfs {
		n, err := pricesFor(ctx, q, f, e.Code)
		if err != nil {
			report.Failed[e.Code] = err.Error()
			log.Printf("prices %s: %v", e.Code, err)
			continue
		}
		report.Stored += n
	}
	return report, nil
}

func pricesFor(ctx context.Context, q *storedb.Queries, f Fetcher, code string) (int, error) {
	symbol := code + ".JO"
	body, err := f.Get(ctx, "https://query1.finance.yahoo.com/v8/finance/chart/"+symbol+"?range=3mo&interval=1d")
	if err != nil {
		return 0, err
	}
	var chart struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency  string `json:"currency"`
					GMTOffset int    `json:"gmtoffset"`
				} `json:"meta"`
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close  []*float64 `json:"close"`
						Volume []*int64   `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &chart); err != nil {
		return 0, fmt.Errorf("parse chart: %w", err)
	}
	if len(chart.Chart.Result) == 0 || len(chart.Chart.Result[0].Indicators.Quote) == 0 {
		return 0, fmt.Errorf("no chart data")
	}
	r := chart.Chart.Result[0]
	quote := r.Indicators.Quote[0]
	currency, divisor := majorUnits(r.Meta.Currency)
	zone := time.FixedZone("exchange", r.Meta.GMTOffset)

	stored := 0
	for i, ts := range r.Timestamp {
		day := time.Unix(ts, 0).In(zone)
		var close *float64
		if i < len(quote.Close) && quote.Close[i] != nil {
			v := *quote.Close[i] / divisor
			close = &v
		}
		var volume *int64
		if i < len(quote.Volume) {
			volume = quote.Volume[i]
		}
		if close == nil && volume == nil {
			continue // a day with no data at all isn't a data point
		}
		if err := q.UpsertDailyPrice(ctx, storedb.UpsertDailyPriceParams{
			EtfCode:    code,
			TradeDate:  pgtype.Date{Time: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC), Valid: true},
			Close:      close,
			Currency:   &currency,
			Volume:     volume,
			SourceName: "Yahoo Finance",
			SourceUrl:  "https://finance.yahoo.com/quote/" + symbol,
		}); err != nil {
			return stored, fmt.Errorf("store %s: %w", day.Format("2006-01-02"), err)
		}
		stored++
	}
	return stored, nil
}

// majorUnits maps Yahoo's minor-unit currencies to ISO codes and the divisor
// that converts prices into major units.
func majorUnits(code string) (string, float64) {
	switch code {
	case "ZAc", "ZAC":
		return "ZAR", 100
	case "GBp", "GBX":
		return "GBP", 100
	}
	return code, 1
}
