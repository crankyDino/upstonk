package ingest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// parseSatrixMDD reads a Satrix Minimum Disclosure Document as produced by
// `pdftotext -layout` (two columns of label/value pairs). Only facts the
// document states are filled; everything else stays nil.
func parseSatrixMDD(text string) (Factsheet, error) {
	fs := Factsheet{SourceName: "Satrix MDD"}
	lines := strings.Split(text, "\n")

	// Header: fund name, "Minimum Disclosure Document", as-of date.
	var header []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			header = append(header, t)
			if len(header) == 3 {
				break
			}
		}
	}
	if len(header) < 3 || header[1] != "Minimum Disclosure Document" {
		return fs, fmt.Errorf("not a Satrix MDD (unexpected header)")
	}
	fs.AsOf = parseLongDate(header[2])

	fs.Code = labelValue(text, "JSE Code")
	if fs.Code == "" {
		return fs, fmt.Errorf("no JSE code in document")
	}
	fs.TrackedIndex = nonEmpty(labelValue(text, "Benchmark"))
	fs.InceptionDate = parseLongDate(labelValue(text, "Inception Date"))
	if c := labelValue(text, "ASISA Classification"); c != "" {
		fs.AssetClass = assetClassFromASISA(c)
	}
	if d := labelValue(text, "Distribution Frequency"); d != "" {
		fs.DistributionPolicy = nonEmpty("Distributing (" + strings.ToLower(d) + ")")
	}
	if m := regexp.MustCompile(`Portfolio Value\s{2,}R\s?([\d.,]+)\s*(billion|million)`).FindStringSubmatch(text); m != nil {
		if v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			scale := 1e6
			if m[2] == "billion" {
				scale = 1e9
			}
			aum, cur := v*scale, "ZAR"
			fs.AUM, fs.AUMCurrency = &aum, &cur
		}
	}
	if m := regexp.MustCompile(`Total Expense Ratio \(TER\)\s+(\d+(?:\.\d+)?)`).FindStringSubmatch(text); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			fs.TERPercent = &v // the 1-year figure (first column)
		}
	}
	// Phrases can be split by numbers from the other column's chart.
	flat := strings.Join(strings.Fields(text), " ")
	if phrase(`direct`, `replication`).MatchString(flat) {
		fs.Replication = nonEmpty("physical")
	}
	if phrase(`registered as a`, `Collective Investment Scheme`).MatchString(flat) {
		fs.LegalStructure = nonEmpty("Collective Investment Scheme")
	}

	fs.Holdings, fs.Exposures = parseSatrixAllocations(lines)
	return fs, nil
}

// labelValue finds "Label<2+ spaces>Value" and returns Value (up to the next
// run of 2+ spaces, which starts the other column).
func labelValue(text, label string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(label) + `\s{2,}(\S(?:.*?\S)?)(?:\s{2,}|\s*$)`)
	if m := re.FindStringSubmatch(text); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

var (
	cellSplit  = regexp.MustCompile(`\s{2,}`)
	percentVal = regexp.MustCompile(`^(\d+(?:\.\d+)?)%$`)
	plainVal   = regexp.MustCompile(`^\d+(?:\.\d+)?$`)
)

// parseSatrixAllocations reads the page-2 block where sector weights (left,
// "33.10%") and top holdings (right, "7.68") share lines.
func parseSatrixAllocations(lines []string) ([]Holding, []Exposure) {
	var holdings []Holding
	var exposures []Exposure
	in := false
	for _, l := range lines {
		switch {
		case strings.Contains(l, "PORTFOLIO ALLOCATION"):
			in = true
			continue
		case strings.Contains(l, "CONTACT DETAILS"):
			in = false
		}
		if !in {
			continue
		}
		cells := cellSplit.Split(strings.TrimSpace(l), -1)
		for i := 1; i < len(cells); i++ {
			label := cells[i-1]
			if m := percentVal.FindStringSubmatch(cells[i]); m != nil && isLabel(label) {
				v, _ := strconv.ParseFloat(m[1], 64)
				exposures = append(exposures, Exposure{Kind: "sector", Name: label, WeightPercent: v})
			} else if plainVal.MatchString(cells[i]) && isLabel(label) {
				v, _ := strconv.ParseFloat(cells[i], 64)
				holdings = append(holdings, Holding{Name: label, WeightPercent: &v})
			}
		}
	}
	return holdings, exposures
}

// phrase matches the given words in order, allowing stray numbers (from the
// other column of a two-column layout) between them.
func phrase(parts ...string) *regexp.Regexp {
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile(strings.Join(quoted, `(?:\s+\d+(?:\.\d+)?)*\s+`))
}

func isLabel(s string) bool {
	return s != "" && !percentVal.MatchString(s) && !plainVal.MatchString(s) && s != "HOLDING" && !strings.HasPrefix(s, "WEIGHT")
}

func assetClassFromASISA(c string) *string {
	lc := strings.ToLower(c)
	switch {
	case strings.Contains(lc, "equity"):
		return nonEmpty("Equity")
	case strings.Contains(lc, "interest bearing"), strings.Contains(lc, "bond"):
		return nonEmpty("Bond")
	case strings.Contains(lc, "real estate"), strings.Contains(lc, "property"):
		return nonEmpty("Property")
	case strings.Contains(lc, "multi-asset"), strings.Contains(lc, "multi asset"):
		return nonEmpty("Multi Asset")
	case strings.Contains(lc, "commodit"):
		return nonEmpty("Commodity")
	}
	return nonEmpty(c) // unrecognised: keep the issuer's wording rather than guess
}

func parseLongDate(s string) pgtype.Date {
	for _, layout := range []string{"2 January 2006", "02 January 2006", "2 Jan 2006"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return pgtype.Date{Time: t, Valid: true}
		}
	}
	return pgtype.Date{}
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
