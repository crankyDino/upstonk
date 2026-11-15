package ingest

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/store/storedb"
)

// TextExtractor turns a PDF into text that keeps the page layout.
type TextExtractor interface {
	Text(ctx context.Context, pdf []byte) (string, error)
}

// PdfToText uses poppler's pdftotext -layout (must be installed).
type PdfToText struct{}

func (PdfToText) Text(ctx context.Context, pdf []byte) (string, error) {
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}

// Factsheet is what an issuer reader extracts from one document. Nil
// fields weren't stated in the document.
type Factsheet struct {
	Code               string // fund code the document is for
	AsOf               pgtype.Date
	SourceName         string
	TERPercent         *float64
	AUM                *float64
	AUMCurrency        *string
	TrackedIndex       *string
	AssetClass         *string
	Domicile           *string
	LegalStructure     *string
	Replication        *string // "physical" | "synthetic"
	IsLeveraged        *bool
	IsInverse          *bool
	DistributionPolicy *string
	InceptionDate      pgtype.Date
	TFSAEligible       *bool
	Holdings           []Holding
	Exposures          []Exposure
}

type Holding struct {
	Name          string
	WeightPercent *float64
}

type Exposure struct {
	Kind          string // sector | region | country
	Name          string
	WeightPercent float64
}

// issuerReader knows where an issuer publishes factsheets and how to read them.
type issuerReader struct {
	url   func(code string) string
	parse func(text string) (Factsheet, error)
}

var readers = map[string]issuerReader{
	"satrix": {url: func(code string) string { return "https://satrix.co.za/fund/mdd/" + code }, parse: parseSatrixMDD},
}

// FactsheetsReport says what a factsheet run stored, skipped and failed.
type FactsheetsReport struct {
	Stored  int
	Skipped map[string]string // ETF code -> why no reader applies
	Failed  map[string]string // ETF code -> reason; nothing stored for these
}

// Factsheets fetches and reads the latest factsheet for every active ETF
// whose issuer has a reader. A document is fully parsed and checked against
// the ETF's code before anything is written.
func Factsheets(ctx context.Context, q *storedb.Queries, f Fetcher, pdf TextExtractor) (FactsheetsReport, error) {
	report := FactsheetsReport{Skipped: map[string]string{}, Failed: map[string]string{}}
	etfs, err := q.ListActiveETFs(ctx)
	if err != nil {
		return report, fmt.Errorf("list ETFs: %w", err)
	}
	for _, e := range etfs {
		issuer := ""
		if e.Issuer != nil {
			issuer = strings.ToLower(*e.Issuer)
		}
		r, ok := readers[issuer]
		if !ok {
			report.Skipped[e.Code] = fmt.Sprintf("no factsheet reader for issuer %q", issuer)
			continue
		}
		if err := factsheetFor(ctx, q, f, pdf, r, e.Code); err != nil {
			report.Failed[e.Code] = err.Error()
			log.Printf("factsheet %s: %v", e.Code, err)
			continue
		}
		report.Stored++
	}
	return report, nil
}

func factsheetFor(ctx context.Context, q *storedb.Queries, f Fetcher, pdf TextExtractor, r issuerReader, code string) error {
	url := r.url(code)
	body, err := f.Get(ctx, url)
	if err != nil {
		return err
	}
	text, err := pdf.Text(ctx, body)
	if err != nil {
		return err
	}
	fs, err := r.parse(text)
	if err != nil {
		return fmt.Errorf("read %s: %w", url, err)
	}
	if !strings.EqualFold(fs.Code, code) {
		return fmt.Errorf("document at %s is for %q, not %q", url, fs.Code, code)
	}
	if !fs.AsOf.Valid {
		return fmt.Errorf("document at %s has no as-of date", url)
	}

	id, err := q.InsertFactsheet(ctx, storedb.InsertFactsheetParams{
		EtfCode: code, AsOf: fs.AsOf, SourceName: fs.SourceName, SourceUrl: url,
		TerPercent: fs.TERPercent, Aum: fs.AUM, AumCurrency: fs.AUMCurrency,
		TrackedIndex: fs.TrackedIndex, AssetClass: fs.AssetClass, Domicile: fs.Domicile,
		LegalStructure: fs.LegalStructure, Replication: fs.Replication,
		IsLeveraged: fs.IsLeveraged, IsInverse: fs.IsInverse,
		DistributionPolicy: fs.DistributionPolicy, InceptionDate: fs.InceptionDate,
		TfsaEligible: fs.TFSAEligible,
	})
	if err != nil {
		return fmt.Errorf("store factsheet: %w", err)
	}
	if err := q.DeleteFactsheetDetails(ctx, id); err != nil {
		return err
	}
	if err := q.DeleteFactsheetExposures(ctx, id); err != nil {
		return err
	}
	for i, h := range fs.Holdings {
		if err := q.InsertHolding(ctx, storedb.InsertHoldingParams{
			FactsheetID: id, Position: int32(i + 1), Name: h.Name, WeightPercent: h.WeightPercent,
		}); err != nil {
			return fmt.Errorf("store holding: %w", err)
		}
	}
	for _, x := range fs.Exposures {
		if err := q.InsertExposure(ctx, storedb.InsertExposureParams{
			FactsheetID: id, Kind: x.Kind, Name: x.Name, WeightPercent: x.WeightPercent,
		}); err != nil {
			return fmt.Errorf("store exposure: %w", err)
		}
	}
	return nil
}
