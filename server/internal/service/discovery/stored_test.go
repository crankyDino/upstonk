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
	q := storedb.New(pool)
	storeSatrix40(t, q, "2026-10-31", 0.10)
	listing(t, q, "STX40", "ETF", ptr(true))

	if got := storedResult(t, pool, tfsaRequest()).Eligibility.Status; got != "eligible" {
		t.Fatalf("status = %q, want eligible (JSE, ZAR, Satrix, physical, not leveraged/inverse)", got)
	}
}

func TestStoredHoldingsAndSectorsDriveMatchingAndAppearInResults(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	ctx := context.Background()
	if err := q.UpsertETF(ctx, storedb.UpsertETFParams{
		Code: "STXNDQ", Name: "Satrix Nasdaq 100 ETF", Issuer: ptr("Satrix"), Exchange: "JSE", Currency: ptr("ZAR"),
		ListingSource: "JSE", ListedAsOf: date(t, "2026-11-10"),
	}); err != nil {
		t.Fatal(err)
	}
	id, err := q.InsertFactsheet(ctx, storedb.InsertFactsheetParams{
		EtfCode: "STXNDQ", AsOf: date(t, "2026-10-31"), SourceName: "Satrix MDD", SourceUrl: "https://satrix.co.za/fund/mdd/STXNDQ",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []storedb.InsertExposureParams{
		{FactsheetID: id, Kind: "sector", Name: "Information Technology", WeightPercent: 60},
		{FactsheetID: id, Kind: "sector", Name: "Communication", WeightPercent: 15},
	} {
		if err := q.InsertExposure(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.InsertHolding(ctx, storedb.InsertHoldingParams{FactsheetID: id, Position: 1, Name: "NVIDIA", WeightPercent: ptr(8.35)}); err != nil {
		t.Fatal(err)
	}

	req := standardRequest(dto.ExposureRequest{Assets: dto.AssetExposureRequest{Sectors: []string{"technology"}}})
	r := storedResult(t, pool, req)

	// 60% of the fund is in the requested sector.
	assertScore(t, "matchScore", r.MatchScore, 60)
	if len(r.TopHoldings) != 1 || r.TopHoldings[0].Name != "NVIDIA" || r.TopHoldings[0].Weight != 8.35 {
		t.Fatalf("topHoldings = %+v", r.TopHoldings)
	}
}

func listing(t *testing.T, q *storedb.Queries, code, instrument string, tfsa *bool) {
	t.Helper()
	if err := q.UpsertListing(context.Background(), storedb.UpsertListingParams{
		Code: code, Name: code + " fund", Issuer: ptr("Satrix"), Currency: ptr("ZAR"), InstrumentType: instrument,
		TfsaFriendly: tfsa, ListingSource: "JSE ETF List v.99", ListedAsOf: date(t, "2026-11-15"),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStoreServesETFsOnlyNotETNs(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listing(t, q, "STX40", "ETF", ptr(true))
	listing(t, q, "EXETNC", "ETN", nil)

	res, err := newService(search.NewStoredProvider(pool)).DiscoverETFs(context.Background(), standardRequest(dto.ExposureRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := tickers(res.Results); len(got) != 1 || got[0] != "STX40" {
		t.Fatalf("results = %v, want [STX40]", got)
	}
}

func TestJSETFSAFlagDecidesTheTFSACriterion(t *testing.T) {
	cases := []struct {
		name      string
		flag      *bool
		ineligble bool
	}{
		{"listed TFSA-friendly", ptr(true), false},
		{"listed not TFSA-friendly", ptr(false), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := dbtest.New(t)
			q := storedb.New(pool)
			storeSatrix40(t, q, "2026-10-31", 0.10)
			listing(t, q, "STX40", "ETF", tc.flag) // JSE list facts on top of the factsheet

			res, err := newService(search.NewStoredProvider(pool)).DiscoverETFs(context.Background(), tfsaRequest())
			if err != nil {
				t.Fatal(err)
			}
			if tc.ineligble {
				if res.Summary.TotalIneligible != 1 {
					t.Fatalf("summary = %+v, want STX40 ineligible", res.Summary)
				}
				return
			}
			passed := onlyFund(t, res).Eligibility.RulesPassed
			found := false
			for _, p := range passed {
				found = found || p == "jse_tfsa_list"
			}
			if !found {
				t.Fatalf("rulesPassed = %v, want jse_tfsa_list", passed)
			}
		})
	}
}
