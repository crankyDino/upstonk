package ingest_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/ingest"
	"upstonk/internal/store/storedb"
	"upstonk/internal/testutil/dbtest"
)

// fakeText stands in for pdftotext: it returns fixed text for any PDF.
type fakeText string

func (f fakeText) Text(_ context.Context, _ []byte) (string, error) { return string(f), nil }

func satrixText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/satrix_mdd_synthetic.txt") // invented values, Satrix layout
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func listSatrixETF(t *testing.T, q *storedb.Queries, code string) {
	t.Helper()
	issuer := "Satrix"
	if err := q.UpsertETF(context.Background(), storedb.UpsertETFParams{
		Code: code, Name: code, Issuer: &issuer, Exchange: "JSE", ListingSource: "JSE",
		ListedAsOf: pgtype.Date{Time: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
}

const satrixURL = "https://satrix.co.za/fund/mdd/STXEX"

func TestSatrixFactsheetIsStoredWithItsFactsAndSource(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listSatrixETF(t, q, "STXEX")

	report, err := ingest.Factsheets(context.Background(), q, fakeFetcher{satrixURL: "%PDF-1.7"}, fakeText(satrixText(t)))
	if err != nil {
		t.Fatal(err)
	}
	if report.Stored != 1 || len(report.Failed) != 0 {
		t.Fatalf("report = %+v", report)
	}

	sheets, _ := q.ListLatestFactsheets(context.Background())
	if len(sheets) != 1 {
		t.Fatalf("got %d factsheets, want 1", len(sheets))
	}
	f := sheets[0]
	checks := []struct {
		name      string
		got, want any
	}{
		{"as_of", f.AsOf.Time.Format("2006-01-02"), "2026-10-31"},
		{"source", f.SourceName, "Satrix MDD"},
		{"url", f.SourceUrl, satrixURL},
		{"ter", deref(f.TerPercent), 0.11}, // 1-year TER
		{"aum", deref(f.Aum), 12.5e9},      // R12.5 billion
		{"aum currency", deref(f.AumCurrency), "ZAR"},
		{"index", deref(f.TrackedIndex), "Example Top 40 Index (X40)"},
		{"asset class", deref(f.AssetClass), "Equity"},
		{"legal structure", deref(f.LegalStructure), "Collective Investment Scheme"},
		{"replication", deref(f.Replication), "physical"}, // "direct replication"
		{"distribution", deref(f.DistributionPolicy), "Distributing (quarterly)"},
		{"inception", f.InceptionDate.Time.Format("2006-01-02"), "2010-03-01"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	// Not stated in a Satrix MDD: must stay unknown, never assumed.
	if f.IsLeveraged != nil || f.IsInverse != nil || f.Domicile != nil || f.TfsaEligible != nil {
		t.Errorf("unstated facts were filled: leveraged %v inverse %v domicile %v tfsa %v",
			f.IsLeveraged, f.IsInverse, f.Domicile, f.TfsaEligible)
	}

	holdings, _ := q.ListHoldings(context.Background(), f.ID)
	if len(holdings) != 3 || holdings[0].Name != "Alpha Holdings Ltd" || deref(holdings[0].WeightPercent) != 9.10 ||
		holdings[2].Name != "Gamma Bank Ltd" || deref(holdings[2].WeightPercent) != 6.75 {
		t.Errorf("holdings = %+v", holdings)
	}
	sectors, _ := q.ListExposures(context.Background(), f.ID)
	want := map[string]float64{"Financials": 40.00, "Basic Materials": 25.50, "Technology": 10.25}
	if len(sectors) != len(want) {
		t.Fatalf("exposures = %+v", sectors)
	}
	for _, s := range sectors {
		if s.Kind != "sector" || want[s.Name] != s.WeightPercent {
			t.Errorf("exposure %+v not expected", s)
		}
	}
}

func TestSatrixFactsheetWithoutATERLineStoresTERAsUnknown(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listSatrixETF(t, q, "STXEX")
	var kept []string
	for _, line := range strings.Split(satrixText(t), "\n") {
		if !strings.Contains(line, "Total Expense Ratio") {
			kept = append(kept, line)
		}
	}

	if _, err := ingest.Factsheets(context.Background(), q, fakeFetcher{satrixURL: "%PDF"}, fakeText(strings.Join(kept, "\n"))); err != nil {
		t.Fatal(err)
	}

	sheets, _ := q.ListLatestFactsheets(context.Background())
	if len(sheets) != 1 || sheets[0].TerPercent != nil {
		t.Fatalf("TER = %v, want NULL", sheets[0].TerPercent)
	}
}

func TestFactsheetForADifferentFundIsRejected(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listSatrixETF(t, q, "STXEX")
	wrong := strings.Replace(satrixText(t), "STXEX", "STXOTHER", 1)

	report, _ := ingest.Factsheets(context.Background(), q, fakeFetcher{satrixURL: "%PDF"}, fakeText(wrong))

	if _, failed := report.Failed["STXEX"]; !failed {
		t.Fatalf("report = %+v, want STXEX failed (document is for STXOTHER)", report)
	}
	if sheets, _ := q.ListLatestFactsheets(context.Background()); len(sheets) != 0 {
		t.Fatalf("stored %d factsheets from the wrong document", len(sheets))
	}
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// In the two-column PDF layout, chart numbers from the right column can land
// inside a left-column sentence ("registered as a 30 Collective Investment
// Scheme"), as in the real STXNDQ document.
func TestSatrixStructureWordingSurvivesInterleavedChartNumbers(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	listSatrixETF(t, q, "STXEX")
	text := strings.NewReplacer(
		"registered as a Collective Investment Scheme", "registered as a                30\n  Collective Investment Scheme",
		"through direct replication", "through direct          21.84\n  replication",
	).Replace(satrixText(t))

	if _, err := ingest.Factsheets(context.Background(), q, fakeFetcher{satrixURL: "%PDF"}, fakeText(text)); err != nil {
		t.Fatal(err)
	}

	f, _ := q.ListLatestFactsheets(context.Background())
	if len(f) != 1 || deref(f[0].LegalStructure) != "Collective Investment Scheme" || deref(f[0].Replication) != "physical" {
		t.Fatalf("legal structure %v, replication %v", f[0].LegalStructure, f[0].Replication)
	}
}
