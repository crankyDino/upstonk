package ingest_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"upstonk/internal/ingest"
	"upstonk/internal/store/storedb"
	"upstonk/internal/testutil/dbtest"
)

// writeETFList builds a workbook in the layout of the JSE "ETF List"
// download (synthetic rows): header, category rows, funds with a "Link"
// hyperlink, a total row and a disclaimer.
func writeETFList(t *testing.T, name string, funds [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	sh := "ETF Web data"
	f.SetSheetName("Sheet1", sh)
	_ = f.SetSheetRow(sh, "A1", &[]string{"Alpha", "Long Name", "TFSA Friendly (CIS)", "Disclosure", "Underlying",
		"Foreign Status (Y/N)", "Treated as Domestic per SARB Classification", "Link", "Number"})
	row := 2
	_ = f.SetSheetRow(sh, "A2", &[]string{"Top 40 Equity"})
	row++
	for _, fund := range funds { // code, name, tfsa, disclosure, index, foreign, sarb, url
		cell, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(sh, cell, &[]string{fund[0], fund[1], fund[2], fund[3], fund[4], fund[5], fund[6], "Link"})
		if fund[7] != "" {
			link, _ := excelize.CoordinatesToCellName(8, row)
			_ = f.SetCellHyperLink(sh, link, fund[7], "External")
		}
		row++
	}
	total, _ := excelize.CoordinatesToCellName(8, row)
	_ = f.SetSheetRow(sh, total, &[]string{"Total", "2"})
	disc, _ := excelize.CoordinatesToCellName(1, row+2)
	_ = f.SetCellValue(sh, disc, "*Disclaimer: This list reflects the TFSA compliance ...")
	path := filepath.Join(t.TempDir(), name)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

var importDay = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

func TestJSEETFListImportStoresEachFundWithItsJSEFacts(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	path := writeETFList(t, "ETF List v.99.xlsx", [][]string{
		{"STX40", "Satrix 40 ETF", "Yes", "Daily", "FTSE/JSE Top 40 Index ", "No", "Domestic", "https://satrix.co.za/products/satrix-40"},
		{"GLDX", "Example Gold ETF", "No", "Quarterly", "Gold Spot", "Yes", "Foreign", ""},
	})

	report, err := ingest.ImportJSEList(context.Background(), q, path, importDay)
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 2 || report.InstrumentType != "ETF" {
		t.Fatalf("report = %+v, want 2 ETFs", report)
	}

	rows, err := q.ListListings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d listings, want 2 (category, total and disclaimer rows skipped)", len(rows))
	}
	stx, gld := rows[1], rows[0] // ordered by code: GLDX, STX40
	checks := []struct {
		name      string
		got, want any
	}{
		{"code", stx.Code, "STX40"},
		{"name", stx.Name, "Satrix 40 ETF"},
		{"type", stx.InstrumentType, "ETF"},
		{"category", deref(stx.Category), "Top 40 Equity"},
		{"index (nbsp trimmed)", deref(stx.UnderlyingIndex), "FTSE/JSE Top 40 Index"},
		{"tfsa", deref(stx.TfsaFriendly), true},
		{"foreign", deref(stx.ForeignStatus), false},
		{"sarb", deref(stx.SarbClassification), "Domestic"},
		{"disclosure", deref(stx.DisclosureFrequency), "Daily"},
		{"issuer from JSE link", deref(stx.Issuer), "Satrix"},
		{"issuer url", deref(stx.IssuerUrl), "https://satrix.co.za/products/satrix-40"},
		{"currency", deref(stx.Currency), "ZAR"},
		{"source", stx.ListingSource, "JSE ETF List v.99"},
		{"as of", stx.ListedAsOf.Time.Format("2006-01-02"), "2026-11-15"},
		{"gold tfsa", deref(gld.TfsaFriendly), false},
		{"gold foreign", deref(gld.ForeignStatus), true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if gld.Issuer != nil || gld.IssuerUrl != nil {
		t.Errorf("no link: issuer must stay unknown, got %v %v", gld.Issuer, gld.IssuerUrl)
	}
}

func TestFundMissingFromANewerListBecomesInactive(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	ctx := context.Background()
	fund := func(code string) []string {
		return []string{code, code + " ETF", "Yes", "Daily", "Index", "No", "Domestic", ""}
	}
	if _, err := ingest.ImportJSEList(ctx, q, writeETFList(t, "ETF List v.98.xlsx", [][]string{fund("KEEP"), fund("GONE")}), importDay); err != nil {
		t.Fatal(err)
	}

	report, err := ingest.ImportJSEList(ctx, q, writeETFList(t, "ETF List v.99.xlsx", [][]string{fund("KEEP")}), importDay)
	if err != nil {
		t.Fatal(err)
	}

	if report.Deactivated != 1 {
		t.Fatalf("deactivated = %d, want 1", report.Deactivated)
	}
	active := map[string]bool{}
	rows, _ := q.ListListings(ctx)
	for _, r := range rows {
		active[r.Code] = r.Active
	}
	if !active["KEEP"] || active["GONE"] {
		t.Fatalf("active = %v, want KEEP active and GONE inactive", active)
	}
}

func TestETNListImportsAsETNs(t *testing.T) {
	pool := dbtest.New(t)
	q := storedb.New(pool)
	f := excelize.NewFile()
	_ = f.SetSheetRow("Sheet1", "A1", &[]string{"Alpha", "Long Name", "Description", "Link", "Number"})
	_ = f.SetSheetRow("Sheet1", "A2", &[]string{"Equity "})
	_ = f.SetSheetRow("Sheet1", "A3", &[]string{"EXETNC", "Example ETN", "Linked to the performance of Example Inc", "https://www.firstrand.co.za/DI/EXETNC", "1"})
	path := filepath.Join(t.TempDir(), "ETN List v.22.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	report, err := ingest.ImportJSEList(context.Background(), q, path, importDay)
	if err != nil {
		t.Fatal(err)
	}

	rows, _ := q.ListListings(context.Background())
	if report.InstrumentType != "ETN" || len(rows) != 1 || rows[0].InstrumentType != "ETN" ||
		deref(rows[0].Issuer) != "FirstRand" || rows[0].ListingSource != "JSE ETN List v.22" || rows[0].TfsaFriendly != nil {
		t.Fatalf("report %+v, row %+v", report, rows)
	}
}
