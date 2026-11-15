// Command ingest fills the UPSTONK store. Run from server/ (reads .env):
//
//	go run ./cmd/ingest jse-list    # after downloading a new JSE ETF/ETN List into data/import/
//	go run ./cmd/ingest prices      # daily: Yahoo closes and volumes for every active ETF
//	go run ./cmd/ingest factsheets  # monthly: issuer factsheets (Satrix so far)
//
// Personal use only: requests are sequential and spaced out (see
// DOCS/plans/jse-data-store.md).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"upstonk/internal/config"
	"upstonk/internal/ingest"
	"upstonk/internal/store"
	"upstonk/internal/store/storedb"
)

const usage = `usage: ingest <command>

commands:
  jse-list [file...]  import the JSE ETF/ETN List (default: newest of each in data/import/)
  prices      fetch ~3 months of daily prices and volumes for every active ETF
  factsheets  fetch and read the latest issuer factsheet for every active ETF`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set (see server/.env.example)")
	}

	ctx := context.Background()
	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	q := storedb.New(pool)
	fetcher := ingest.NewPoliteFetcher(3 * time.Second)

	// Every run is recorded for the admin data-status page.
	runID, err := q.StartIngestRun(ctx, os.Args[1])
	if err != nil {
		log.Fatalf("record run: %v", err)
	}
	stored, failed := 0, map[string]string{}
	defer func() {
		f, _ := json.Marshal(failed)
		if err := q.FinishIngestRun(context.Background(), storedb.FinishIngestRunParams{
			ID: runID, Stored: int32(stored), Failed: f,
		}); err != nil {
			log.Printf("record run: %v", err)
		}
	}()

	switch os.Args[1] {
	case "jse-list":
		files := os.Args[2:]
		if len(files) == 0 {
			files, err = newestLists("data/import")
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, path := range files {
			report, err := ingest.ImportJSEList(ctx, q, path, time.Now())
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%s: %d imported, %d no longer listed (marked inactive) from %s",
				report.Source, report.Imported, report.Deactivated, filepath.Base(path))
			stored += report.Imported
		}
	case "prices":
		report, err := ingest.Prices(ctx, q, fetcher)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("prices: %d trading days stored, %d ETFs failed", report.Stored, len(report.Failed))
		printSorted(report.Failed)
		stored, failed = report.Stored, report.Failed
	case "factsheets":
		report, err := ingest.Factsheets(ctx, q, fetcher, ingest.PdfToText{})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("factsheets: %d stored, %d failed, %d skipped (no reader for issuer)",
			report.Stored, len(report.Failed), len(report.Skipped))
		printSorted(report.Failed)
		stored, failed = report.Stored, report.Failed
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func printSorted(failed map[string]string) {
	codes := make([]string, 0, len(failed))
	for c := range failed {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		log.Printf("  %s: %s", c, failed[c])
	}
}

// newestLists picks the highest-version "ETF List" and "ETN List" workbook in
// dir (the JSE names downloads like "ETF List v.99.xlsx"). AMC lists are out
// of scope.
func newestLists(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	if err != nil {
		return nil, err
	}
	version := regexp.MustCompile(`(?i)v\.?\s*(\d+)`)
	best := map[string]struct {
		path string
		v    int
	}{}
	for _, p := range paths {
		base := filepath.Base(p)
		var kind string
		switch {
		case regexp.MustCompile(`(?i)^ETF List`).MatchString(base):
			kind = "ETF"
		case regexp.MustCompile(`(?i)^ETN List`).MatchString(base):
			kind = "ETN"
		default:
			continue
		}
		v := 0
		if m := version.FindStringSubmatch(base); m != nil {
			v, _ = strconv.Atoi(m[1])
		}
		if cur, ok := best[kind]; !ok || v > cur.v {
			best[kind] = struct {
				path string
				v    int
			}{p, v}
		}
	}
	if len(best) == 0 {
		return nil, fmt.Errorf("no \"ETF List\" or \"ETN List\" .xlsx in %s", dir)
	}
	out := []string{}
	for _, k := range []string{"ETF", "ETN"} {
		if b, ok := best[k]; ok {
			out = append(out, b.path)
		}
	}
	return out, nil
}
