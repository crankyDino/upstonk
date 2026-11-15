package discovery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/api/dto"
	"upstonk/internal/service/discovery"
	"upstonk/internal/service/search"
	"upstonk/internal/store/storedb"
	"upstonk/internal/testutil/dbtest"
)

// Tests in this file run the real pipeline over the Postgres-backed provider,
// each against its own fresh database (see internal/testutil/dbtest).

func TestStoredProviderWithEmptyStoreIsNoResults(t *testing.T) {
	svc := newService(search.NewStoredProvider(dbtest.New(t)))

	_, err := svc.DiscoverETFs(context.Background(), tfsaRequest())

	var noResults *discovery.NoResultsError
	if !errors.As(err, &noResults) {
		t.Fatalf("err = %v, want *NoResultsError", err)
	}
}

func date(t *testing.T, s string) pgtype.Date {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return pgtype.Date{Time: d, Valid: true}
}

func ptr[T any](v T) *T { return &v }

// storeSatrix40 stores STX40's JSE listing and one Satrix factsheet.
func storeSatrix40(t *testing.T, q *storedb.Queries, asOf string, ter float64) {
	t.Helper()
	ctx := context.Background()
	if err := q.UpsertETF(ctx, storedb.UpsertETFParams{
		Code: "STX40", Name: "Satrix 40 ETF", Isin: ptr("ZAE000027108"), Issuer: ptr("Satrix"),
		Exchange: "JSE", Currency: ptr("ZAR"),
		ListingSource: "JSE", ListingUrl: ptr("https://www.jse.co.za/etf-list"), ListedAsOf: date(t, "2026-11-10"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.InsertFactsheet(ctx, storedb.InsertFactsheetParams{
		EtfCode: "STX40", AsOf: date(t, asOf),
		SourceName: "Satrix factsheet", SourceUrl: "https://satrix.co.za/factsheets/stx40-" + asOf + ".pdf",
		TerPercent: ptr(ter), Aum: ptr(15e9), AumCurrency: ptr("ZAR"),
		Replication: ptr("physical"), IsLeveraged: ptr(false), IsInverse: ptr(false),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStoredFactsheetValuesReachTheResultWithTheirSource(t *testing.T) {
	pool := dbtest.New(t)
	storeSatrix40(t, storedb.New(pool), "2026-10-31", 0.10)
	req := standardRequest(dto.ExposureRequest{})
	req.OutputOptions.IncludeSourceLinks = true

	res, err := newService(search.NewStoredProvider(pool)).DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	r := onlyResult(t, res)
	assertScore(t, "ter", r.TER, 0.10)
	assertScore(t, "aum", r.AUM, 15e9)
	if r.Name != "Satrix 40 ETF" || r.Provider != "Satrix" || r.ISIN != "ZAE000027108" {
		t.Fatalf("identity = %q/%q/%q", r.Name, r.Provider, r.ISIN)
	}
	want := map[string]string{ // provider -> date
		"Satrix factsheet": "2026-10-31",
		"JSE":              "2026-11-10",
	}
	for _, ds := range r.DataSources {
		if d, ok := want[ds.Provider]; ok && d == ds.Date && ds.URL != "" {
			delete(want, ds.Provider)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing provenance %v in %+v", want, r.DataSources)
	}
}

func storedResult(t *testing.T, pool *pgxpool.Pool, req dto.DiscoveryRequest) dto.ETFResult {
	t.Helper()
	res, err := newService(search.NewStoredProvider(pool)).DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return onlyFund(t, res)
}

func TestTheLatestFactsheetWins(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	storeSatrix40(t, q, "2026-09-30", 0.20)
	storeSatrix40(t, q, "2026-10-31", 0.10)

	assertScore(t, "ter", storedResult(t, pool, standardRequest(dto.ExposureRequest{})).TER, 0.10)
}

func TestAverageDailyVolumeComesFromStoredPrices(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	storeSatrix40(t, q, "2026-10-31", 0.10)
	for d, vol := range map[string]int64{"2026-11-11": 100000, "2026-11-12": 200000, "2026-11-13": 300000} {
		if err := q.UpsertDailyPrice(context.Background(), storedb.UpsertDailyPriceParams{
			EtfCode: "STX40", TradeDate: date(t, d), Close: ptr(105.22), Currency: ptr("ZAR"), Volume: ptr(vol),
			SourceName: "Yahoo Finance", SourceUrl: "https://finance.yahoo.com/quote/STX40.JO",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// (100k + 200k + 300k) / 3 trading days
	assertScore(t, "averageDailyVolume", storedResult(t, pool, standardRequest(dto.ExposureRequest{})).AverageDailyVolume, 200000)
}

func TestTFSAWithAFullyStoredSatrixFactsheetIsEligible(t *testing.T) {
	pool := dbtest.New(t)
	storeSatrix40(t, storedb.New(pool), "2026-10-31", 0.10)

	if got := storedResult(t, pool, tfsaRequest()).Eligibility.Status; got != "eligible" {
		t.Fatalf("status = %q, want eligible (JSE, ZAR, Satrix, physical, not leveraged/inverse)", got)
	}
}
