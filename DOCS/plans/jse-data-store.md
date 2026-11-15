# Plan: stored, sourced data for every JSE ETF

**Status:** in progress (started 2026-11-15) on `dev`. Done: 8a (store, stored provider, ZA routing), 8b (daily prices), 8d for Satrix. Waiting on the user: the JSE ETF List file (8c), and the Supabase password in `server/.env`.
**Goal:** accurate data for every ETF listed on the JSE, with the source and "as of" date of every value. Free sources only: the JSE list, issuer factsheets and Yahoo end-of-day prices. Use an API where one exists; otherwise scrape within robots.txt.

## Decisions (2026-11-15)

| Topic | Decision |
|---|---|
| Scope | All JSE-listed ETFs/ETNs |
| Budget | Free sources only |
| Freshness | Prices daily; fund facts from the latest monthly factsheet, dated |
| Collection | **Personal use only** (user decision, 2026-11-15, after the research found that no free source permits automated collection). Collect as the user would by hand: sequential requests at human pace, a browser User-Agent, every file cached, factsheets monthly and prices daily. **Bot challenges and logins** (JSE, Sygnia, 1nvest) may be passed, but **only after saying so explicitly first**: which site, which protection, and how (user decision, 2026-11-15). A manual import of files the user saves stays available as the fallback. If the app is ever shared or hosted, this decision has to be revisited. |
| Store | Supabase Postgres via the session pooler (`:5432`, `sslmode=require`), configured by `DATABASE_URL` |
| DB code | All in the Go backend: pgx v5.13.0, sqlc v1.31.1 (typed queries), goose v3.30.1 (migrations embedded in the binary) |
| Schema | Tables in their own `upstonk` schema, not `public` (so Supabase's REST API doesn't expose them) |
| Test seams | `discovery.Service` with the DB-backed provider; ingest job entry points (fake HTTP sources); the existing HTTP router |
| Unstated facts | Only apply what a source states; no assumptions (user, 2026-11-15). E.g. Satrix factsheets don't state leverage or inverse exposure, so those stay unknown, even though "index proportions" wording suggests neither. |
| Currency | JSE-listed ETFs are recorded in **ZAR** (user, 2026-11-15) |
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

### 8f: Single-instance deployment on Render (requested 2026-11-15)
One Render **web service**, not separate GUI and API deployments:
- A multi-stage Dockerfile builds the GUI (`npm run build`) and the Go server. The server serves the built GUI at `/` (SPA fallback to `index.html`) and the API at `/api/v1`. Same origin, so the GUI calls `/api/v1` relatively.
- `DATABASE_URL` (Supabase session pooler) is a Render secret. Migrations run at start-up.
- **Ingest stays on the user's machine**, against Supabase (the JSE list is downloaded by hand anyway). The Render instance only reads. Running it in-process on a timer is unreliable on the free tier, which sleeps idle services.
- **Must come first:** authentication (known gap 22). A Render URL is public, and the personal-use basis for collecting the data requires the app to stay private. Simplest option: HTTP basic auth or a single shared password in front of everything, set as a Render secret.

### 8g: Supabase Auth, invitations and admin dashboard (requested 2026-11-15)
**Sharing scope (user decision):** invite-only, for family/friends and ideally a financial adviser who checks whether the insights are accurate. The user judges that circle as still personal use. Anything wider means revisiting the data terms first.

- **Sign-in:** Supabase Auth, email + password, via supabase-js in the GUI (project URL + publishable key).
- **API protection:** the Go server verifies `Authorization: Bearer <JWT>` against the project's JWKS (ES256, `https://<ref>.supabase.co/auth/v1/.well-known/jwks.json`), plus issuer, audience `authenticated` and expiry. No Supabase secret on the server. `/api/v1/health` stays public.
- **Membership (ours, `upstonk` schema):**
  - `app_users`: Supabase user id, email, role admin/member, status active/disabled, invited_by;
  - `invitations`: email, one-time token, invited_by, expiry, used_at/by, revoked;
  - `app_settings`: `invite_only` (default **on**), a notice text.
- **Admission on first call:**
  - an email in `ADMIN_EMAILS` → admin;
  - `invite_only` off → member;
  - otherwise an open invitation whose token (from the invite link) and email both match → member, and the invitation is marked used;
  - anything else → 403. Disabled users → 403.
- **Invitations:** any active member can create an invite **link** for an email and share it however they like. Admins can list and revoke them.
- **Admin dashboard `/admin`** (admins only):
  - users and roles (promote/demote, disable/enable);
  - invitations;
  - app settings (`invite_only`, notice);
  - data status (per source: last import/run, counts, failures, stalest data).
- **Without `SUPABASE_URL`** (local development) the API stays open and logs a warning. `SUPABASE_URL` without `DATABASE_URL` is a start-up error.
- **Tests at the router seam:** tests mint ES256 tokens against a fake JWKS server; nothing touches Supabase.

## Verification
- `go test ./...` runs the DB tests against Docker Postgres. They skip with a message if neither Docker nor `TEST_DATABASE_URL` is available.
- `go run ./cmd/migrate` against Supabase, then `go run ./cmd/ingest prices`, then a ZA discovery returning stored values with their sources.
