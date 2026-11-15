package ingest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/ingest"
	"upstonk/internal/store/storedb"
	"upstonk/internal/testutil/dbtest"
)

// fakeFetcher serves canned bodies by URL; anything else is an error.
type fakeFetcher map[string]string

func (f fakeFetcher) Get(_ context.Context, url string) ([]byte, error) {
	if body, ok := f[url]; ok {
		return []byte(body), nil
	}
	return nil, errors.New("HTTP 404 for " + url)
}

func listETF(t *testing.T, q *storedb.Queries, code string) {
	t.Helper()
	if err := q.UpsertETF(context.Background(), storedb.UpsertETFParams{
		Code: code, Name: code + " ETF", Exchange: "JSE", ListingSource: "JSE",
		ListedAsOf: pgtype.Date{Time: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
}

func chartURL(code string) string {
	return "https://query1.finance.yahoo.com/v8/finance/chart/" + code + ".JO?range=3mo&interval=1d"
}

// Synthetic Yahoo chart response: JSE prices are quoted in cents ("ZAc").
// 2026-11-11, 12, 13 at 00:00 UTC; the last day has no volume.
const stx40Chart = `{"chart":{"result":[{"meta":{"currency":"ZAc","symbol":"STX40.JO","exchangeName":"JNB"},
 "timestamp":[1794355200,1794441600,1794528000],
 "indicators":{"quote":[{"close":[10500,10600.5,10700],"volume":[100000,200000,null]}]}}],"error":null}}`

func TestPricesStoresEachTradingDayInRand(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listETF(t, q, "STX40")

	report, err := ingest.Prices(context.Background(), q, fakeFetcher{chartURL("STX40"): stx40Chart})
	if err != nil {
		t.Fatal(err)
	}
	if report.Stored != 3 || len(report.Failed) != 0 {
		t.Fatalf("report = %+v, want 3 days stored and no failures", report)
	}

	rows, err := q.ListDailyPrices(context.Background(), "STX40")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		day    string
		close  float64
		volume *int64
	}{
		{"2026-11-11", 105.00, ptr(int64(100000))},
		{"2026-11-12", 106.005, ptr(int64(200000))},
		{"2026-11-13", 107.00, nil}, // no volume reported: stays NULL
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, w := range want {
		r := rows[i]
		if got := r.TradeDate.Time.Format("2006-01-02"); got != w.day {
			t.Errorf("row %d: day %s, want %s", i, got, w.day)
		}
		if r.Close == nil || *r.Close != w.close || r.Currency == nil || *r.Currency != "ZAR" {
			t.Errorf("row %d: close %v %v, want %v ZAR", i, r.Close, r.Currency, w.close)
		}
		if (r.Volume == nil) != (w.volume == nil) || (r.Volume != nil && *r.Volume != *w.volume) {
			t.Errorf("row %d: volume %v, want %v", i, r.Volume, w.volume)
		}
		if r.SourceName != "Yahoo Finance" || r.SourceUrl != "https://finance.yahoo.com/quote/STX40.JO" {
			t.Errorf("row %d: source %q %q", i, r.SourceName, r.SourceUrl)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestPricesRerunDoesNotDuplicateDays(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listETF(t, q, "STX40")
	src := fakeFetcher{chartURL("STX40"): stx40Chart}

	for i := 0; i < 2; i++ {
		if _, err := ingest.Prices(context.Background(), q, src); err != nil {
			t.Fatal(err)
		}
	}

	rows, _ := q.ListDailyPrices(context.Background(), "STX40")
	if len(rows) != 3 {
		t.Fatalf("got %d rows after two runs, want 3", len(rows))
	}
}

func TestPricesFailureForOneETFStoresNothingForItAndKeepsTheRest(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listETF(t, q, "STX40")
	listETF(t, q, "GONE") // Yahoo has no data for it

	report, err := ingest.Prices(context.Background(), q, fakeFetcher{chartURL("STX40"): stx40Chart})
	if err != nil {
		t.Fatal(err)
	}

	if _, failed := report.Failed["GONE"]; !failed || len(report.Failed) != 1 {
		t.Fatalf("failed = %v, want only GONE", report.Failed)
	}
	gone, _ := q.ListDailyPrices(context.Background(), "GONE")
	kept, _ := q.ListDailyPrices(context.Background(), "STX40")
	if len(gone) != 0 || len(kept) != 3 {
		t.Fatalf("GONE rows %d (want 0), STX40 rows %d (want 3)", len(gone), len(kept))
	}
}
