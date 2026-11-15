// Command ingest fills the UPSTONK store. Run from server/ (reads .env):
//
//	go run ./cmd/ingest prices      # daily: Yahoo closes and volumes for every active ETF
//
// Personal use only: requests are sequential and spaced out (see
// DOCS/plans/jse-data-store.md).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"upstonk/internal/config"
	"upstonk/internal/ingest"
	"upstonk/internal/store"
	"upstonk/internal/store/storedb"
)

const usage = `usage: ingest <command>

commands:
  prices   fetch ~3 months of daily prices and volumes for every active ETF`

func main() {
	if len(os.Args) != 2 {
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

	switch os.Args[1] {
	case "prices":
		report, err := ingest.Prices(ctx, q, fetcher)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("prices: %d trading days stored, %d ETFs failed", report.Stored, len(report.Failed))
		codes := make([]string, 0, len(report.Failed))
		for c := range report.Failed {
			codes = append(codes, c)
		}
		sort.Strings(codes)
		for _, c := range codes {
			log.Printf("  %s: %s", c, report.Failed[c])
		}
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}
