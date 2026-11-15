# Plan: stored, sourced data for every JSE ETF

**Status:** in progress (started 2026-11-15) on `dev`. Done: 8a (store, stored provider, ZA routing), 8b (daily prices), 8d for Satrix. Waiting on the user: the JSE ETF List file (8c), and the Supabase password in `server/.env`.
**Goal:** accurate data for every ETF listed on the JSE, with the source and "as of" date of every value. Free sources only: the JSE list, issuer factsheets and Yahoo end-of-day prices. Use an API where one exists; otherwise scrape within robots.txt.

## Decisions (2026-11-15)

| Topic | Decision |
|---|---|
| Scope | All JSE-listed ETFs/ETNs |
| Budget | Free sources only |
| Freshness | Prices daily; fund facts from the latest monthly factsheet, dated |
| Collection | **Personal use only** (user decision, 2026-11-15, after the research found that no free source permits automated collection). Collect as the user would by hand: sequential requests at human pace, a browser User-Agent, every file cached, factsheets monthly and prices daily. **Never defeat bot challenges or logins**; those issuers (Sygnia, 1nvest) get a manual import of files the user saves. If the app is ever shared or hosted, this decision has to be revisited. |
| Store | Supabase Postgres via the session pooler (`:5432`, `sslmode=require`), configured by `DATABASE_URL` |
| DB code | All in the Go backend: pgx v5.13.0, sqlc v1.31.1 (typed queries), goose v3.30.1 (migrations embedded in the binary) |
| Schema | Tables in their own `upstonk` schema, not `public` (so Supabase's REST API doesn't expose them) |
| Test seams | `discovery.Service` with the DB-backed provider; ingest job entry points (fake HTTP sources); the existing HTTP router |
| Test DB | Throwaway local Postgres (Docker `postgres:16-alpine`, or `TEST_DATABASE_URL`), one fresh database per test. Never Supabase. |

## Data model (`upstonk` schema)

Each table that holds collected data records where the data came from.

- **`etfs`**: one row per listed instrument. JSE code (key), name, ISIN, issuer, exchange, currency, active flag. Listing provenance: `listing_source`, `listing_url`, `listed_as_of`.
- **`factsheets`**: one row per issuer factsheet read (a snapshot), keeping history. `as_of`, `source_name`, `source_url`, `fetched_at`, and the facts: TER, AUM and its currency, tracked index, asset class, domicile, legal structure, replication, leveraged/inverse (nullable = unknown), distribution policy, inception date, TFSA eligibility (nullable).
- **`factsheet_holdings`**, **`factsheet_exposures`** (kind = sector / region / country): rows belonging to a factsheet.
- **`daily_prices`**: one row per ETF per trading day. Close, currency, volume, `source_name`, `source_url`, `fetched_at`.

Everything is nullable unless a source always provides it. A missing value stays NULL; it never becomes a default.

## Phases

### 8a: Store and DB-backed provider (now)
1. Migrations, the test-DB helper, and a tracer test: empty store → `NoResultsError`.
2. ETF + latest factsheet → a discovery result carrying TER/AUM/flags and provenance (factsheet URL, as-of date).
3. The latest factsheet wins over older ones.
4. Average daily volume computed from stored daily prices (last 3 months).
5. TFSA end to end: a stored Satrix factsheet with all facts → `eligible`. Missing facts → skipped criteria.
6. Wiring: with `DATABASE_URL` set, ZA searches read the store; other countries keep the live provider.
7. Router end to end with the DB provider.

### 8b: Daily price ingest (now)
`cmd/ingest prices`: Yahoo v8 chart for every active ETF, stored in `daily_prices`. Re-running is idempotent. Failures are logged per ticker and never written as data. Tested with a fake Yahoo server.

### 8c: JSE listing ingest (after research)
The authoritative ETF list from the JSE → `etfs`. The source is decided by `DOCS/research/jse-etf-data-sources.md`.

### 8d: Factsheet ingest (after research)
A reader per issuer for factsheets/MDDs → `factsheets` (+ holdings, exposures). Tested with saved sample documents.

### 8e: Retire the hardcoded JSE ticker list
Once the JSE list and factsheet ingest are running.

## Verification
- `go test ./...` runs the DB tests against Docker Postgres. They skip with a message if neither Docker nor `TEST_DATABASE_URL` is available.
- `go run ./cmd/migrate` against Supabase, then `go run ./cmd/ingest prices`, then a ZA discovery returning stored values with their sources.
