package ingest

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"

	"upstonk/internal/store/storedb"
)

// The page the user downloads the lists from (it shows a bot challenge, so
// the files are imported rather than fetched).
const jseListsPage = "https://www.jse.co.za/trade/equity-market/exchange-traded-products"

// issuerByDomain names the issuer behind each issuer site the JSE list links
// to. A fund whose link isn't here (or has no link) keeps an unknown issuer.
var issuerByDomain = map[string]string{
	"satrix.co.za":            "Satrix",
	"1nvest.co.za":            "1nvest",
	"prescient.co.za":         "Prescient",
	"etfs.easyequities.co.za": "EasyETFs",
	"coreshares.co.za":        "CoreShares",
	"10x.co.za":               "10X",
	"fnb.co.za":               "FNB",
	"firstrand.co.za":         "FirstRand",
	"etfcib.absa.co.za":       "Absa",
	"sygnia.co.za":            "Sygnia",
	"coronation.com":          "Coronation",
	"27four.com":              "27four",
	"reitwayglobal.com":       "Reitway",
	"etfsa.co.za":             "etfSA",
	"allangray.co.za":         "Allan Gray",
	"ninetyone.com":           "Ninety One",
	"rmb.co.za":               "RMB",
	"numoro.co.za":            "Numoro",
	"portfoliometrix.com":     "PortfolioMetrix",
	"realfin.co.za":           "Realfin",
	"arysteq.com":             "Arysteq",
	"lauriumcapital.com":      "Laurium",
}

// ListReport says what a JSE list import did.
type ListReport struct {
	InstrumentType string // ETF or ETN
	Source         string // e.g. "JSE ETF List v.99"
	Imported       int
	Deactivated    int64 // previously listed instruments missing from this list
}

var (
	jseCode     = regexp.MustCompile(`^[A-Z0-9]{2,12}$`)
	listVersion = regexp.MustCompile(`(?i)v\.?\s*(\d+)`)
)

// ImportJSEList imports a JSE "ETF List" or "ETN List" workbook saved by the
// user. The file has no date of its own, so its facts are recorded as of
// importDay. Instruments of the same type missing from the file are marked
// inactive. JSE-listed instruments are recorded in ZAR (user decision).
func ImportJSEList(ctx context.Context, q *storedb.Queries, path string, importDay time.Time) (ListReport, error) {
	var report ListReport
	f, err := excelize.OpenFile(path)
	if err != nil {
		return report, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return report, fmt.Errorf("read %s: %w", path, err)
	}

	headerRow, cols := -1, map[string]int{}
	for i, r := range rows {
		if len(r) > 0 && strings.EqualFold(strings.TrimSpace(r[0]), "Alpha") {
			headerRow = i
			for j, h := range r {
				cols[strings.ToLower(strings.TrimSpace(h))] = j
			}
			break
		}
	}
	if headerRow < 0 {
		return report, fmt.Errorf("%s: no header row starting with \"Alpha\"", filepath.Base(path))
	}
	_, isETF := colPrefix(cols, "tfsa friendly")
	report.InstrumentType = "ETN"
	if isETF {
		report.InstrumentType = "ETF"
	}
	report.Source = "JSE " + report.InstrumentType + " List"
	if m := listVersion.FindStringSubmatch(filepath.Base(path)); m != nil {
		report.Source += " v." + m[1]
	}

	nameCol, _ := colPrefix(cols, "long name")
	if _, ok := colPrefix(cols, "long name"); !ok {
		nameCol, _ = colPrefix(cols, "instrument name")
	}
	linkCol, hasLink := colPrefix(cols, "link")
	asOf := pgtype.Date{Time: time.Date(importDay.Year(), importDay.Month(), importDay.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
	zar := "ZAR"

	var codes []string
	category := ""
	for i := headerRow + 1; i < len(rows); i++ {
		r := rows[i]
		code := cell(r, 0)
		name := cell(r, nameCol)
		if code != "" && name == "" && !strings.HasPrefix(code, "*") {
			category = code // a section heading such as "Top 40 Equity"
			continue
		}
		if !jseCode.MatchString(code) || name == "" {
			continue // totals, notes, disclaimer
		}

		var issuer, issuerURL *string
		if hasLink {
			link := ""
			axis, _ := excelize.CoordinatesToCellName(linkCol+1, i+1)
			if ok, target, _ := f.GetCellHyperLink(sheet, axis); ok {
				link = target
			} else if t := cell(r, linkCol); strings.HasPrefix(t, "http") {
				link = t
			}
			if link != "" {
				issuerURL = &link
				issuer = issuerFor(link)
			}
		}

		p := storedb.UpsertListingParams{
			Code: code, Name: name, Issuer: issuer, IssuerUrl: issuerURL, Currency: &zar,
			InstrumentType: report.InstrumentType, Category: nonEmpty(category),
			ListingSource: report.Source, ListingUrl: nonEmpty(jseListsPage), ListedAsOf: asOf,
		}
		if c, ok := colPrefix(cols, "underlying"); ok {
			p.UnderlyingIndex = nonEmpty(cell(r, c))
		}
		if c, ok := colPrefix(cols, "tfsa friendly"); ok {
			p.TfsaFriendly = yesNo(cell(r, c))
		}
		if c, ok := colPrefix(cols, "foreign status"); ok {
			p.ForeignStatus = yesNo(cell(r, c))
		}
		if c, ok := colPrefix(cols, "treated as domestic"); ok {
			p.SarbClassification = nonEmpty(cell(r, c))
		}
		if c, ok := colPrefix(cols, "disclosure"); ok {
			p.DisclosureFrequency = nonEmpty(cell(r, c))
		}
		if c, ok := colPrefix(cols, "description"); ok && p.UnderlyingIndex == nil {
			p.UnderlyingIndex = nonEmpty(cell(r, c))
		}
		if err := q.UpsertListing(ctx, p); err != nil {
			return report, fmt.Errorf("store %s: %w", code, err)
		}
		codes = append(codes, code)
		report.Imported++
	}
	if report.Imported == 0 {
		return report, fmt.Errorf("%s: no instruments found", filepath.Base(path))
	}
	n, err := q.DeactivateMissing(ctx, storedb.DeactivateMissingParams{InstrumentType: report.InstrumentType, Codes: codes})
	if err != nil {
		return report, fmt.Errorf("deactivate delisted: %w", err)
	}
	report.Deactivated = n
	return report, nil
}

func colPrefix(cols map[string]int, prefix string) (int, bool) {
	for name, i := range cols {
		if strings.HasPrefix(name, prefix) {
			return i, true
		}
	}
	return 0, false
}

// cell returns a trimmed cell, also trimming non-breaking spaces.
func cell(r []string, i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(r[i], " ", " "))
}

func yesNo(s string) *bool {
	switch strings.ToLower(s) {
	case "yes", "y":
		v := true
		return &v
	case "no", "n":
		v := false
		return &v
	}
	return nil
}

func issuerFor(link string) *string {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	for domain, name := range issuerByDomain {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			n := name
			return &n
		}
	}
	return nil
}
