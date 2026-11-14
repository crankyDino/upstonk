# UPSTONK

ETF discovery with explicit eligibility rules. You describe an investor (country,
account type) and the exposure you want (markets, sectors, asset classes, companies);
the service searches live public data sources, evaluates each candidate against codified
regulatory rules, filters on your constraints, and returns a ranked list where every
result carries a justification, a confidence level, and its data provenance.

The guiding rule of the eligibility engine: **never guess**. If the data needed to make a
call is missing, the result is marked `unknown` or `conditional` rather than `eligible`.

> Originally vibecoded. The documentation below is written from the code as it actually
> behaves today — see [Known gaps](#known-gaps--gotchas) for where the code and the
> older aspirational docs disagree.

---

## Repo layout

```
upstonk/
├── server/              Go REST API (module `upstonk`, Go 1.25.5)
│   ├── cmd/server/      main.go — config load, server start, graceful shutdown
│   ├── internal/
│   │   ├── api/
│   │   │   ├── dto/         request/response contracts + validation tags
│   │   │   ├── handlers/    DiscoveryHandler (discover, top performers, health)
│   │   │   ├── middleware/  logging, recovery, CORS, timeout
│   │   │   └── router/      route table + HTML docs page (test seam)
│   │   ├── app/         engine + search provider wiring shared by server and tests
│   │   ├── testutil/    fakeprovider — test double for search.Provider
│   │   ├── config/      env loading (.env via godotenv)
│   │   ├── domain/      ETF, EligibilityResult, RankingScore, DataSource
│   │   └── service/
│   │       ├── discovery/    7-step orchestration pipeline
│   │       ├── search/       Provider interface + Live/AlphaVantage/Aggregated
│   │       ├── eligibility/  rule engine + rules/tfsa_za.go
│   │       └── ranking/       WeightedScorer
│   ├── examples/test_payloads.sh    curl examples against a running server
│   └── scripts/test_live_data.sh    smoke test for live data fetching
└── gui/                 Vite + React 18 + TypeScript SPA
    ├── src/components/  feature components + shadcn/ui primitives in ui/
    ├── src/stores/      discoveryStore.ts (zustand — form state + API call)
    ├── src/lib/api/     upstonk.ts (fetch client + formatters)
    ├── src/types/api.ts TS mirror of the server DTOs
    └── src/pages/       Index.tsx (the whole app), NotFound.tsx
```

---

## Architecture

```
                    ┌──────────────────────────────────────────┐
  Browser  ───────► │  gui/  Vite + React + zustand            │
                    │  discoveryStore → lib/api/upstonk.ts     │
                    └────────────────────┬─────────────────────┘
                                         │ POST /api/v1/discover
                    ┌────────────────────▼─────────────────────┐
                    │  gorilla/mux router                      │
                    │  middleware: Logger → Recovery → CORS →  │
                    │              Timeout(30s)                │
                    └────────────────────┬─────────────────────┘
                    ┌────────────────────▼─────────────────────┐
                    │  handlers.DiscoveryHandler               │
                    │  struct validation + business rules      │
                    └────────────────────┬─────────────────────┘
                    ┌────────────────────▼─────────────────────┐
                    │  discovery.Service  (orchestrator)       │
                    └──┬──────────────┬──────────────┬─────────┘
                       │              │              │
         ┌─────────────▼───┐ ┌────────▼───────┐ ┌───▼────────────┐
         │ search.Provider │ │ eligibility.   │ │ ranking.Engine │
         │                 │ │ Engine         │ │                │
         │ Aggregated      │ │ rule registry  │ │ WeightedScorer │
         │  ├─ Live        │ │  └─ TFSA_ZA    │ │                │
         │  └─ AlphaVantage│ │                │ │                │
         │  + 1h TTL cache │ │                │ │                │
         └────────┬────────┘ └────────────────┘ └────────────────┘
                  │
         Yahoo Finance · ETF.com · JSE tickers · Alpha Vantage
```

All three engines are interfaces (`search.Provider`, `eligibility.Engine`,
`ranking.Engine`) wired in `server/internal/app`, so providers, rules, and scoring
are swappable without touching the pipeline.

### The discovery pipeline

`discovery.Service.DiscoverETFs` (`server/internal/service/discovery/discovery.go`) runs
seven steps:

| # | Step | What happens |
|---|------|--------------|
| 1 | **Search** | Fan out to every registered provider in parallel; dedupe and merge by ticker/ISIN. Empty result → `NoResultsError` (HTTP 404). |
| 2 | **Eligibility** | Evaluate each candidate against the first rule whose `AppliesTo(country, accountType)` matches. Tally eligible / ineligible / unknown. |
| 3 | **Constraints** | Drop candidates failing `maxTER`, `minAUM`, `minLiquidity`, `allowedExchanges`, and the synthetic / leveraged / inverse / physical-only switches. |
| 4 | **Match score** | How well the ETF matches the requested exposure: geography 30 pts, sectors 25, asset classes 25, company holdings 20 — normalised to 0–100 (50 when no criteria given). |
| 5 | **Rank** | `ranking.Engine` scores quality; final order is `0.4 × matchScore + 0.6 × rankingScore`. Ranks assigned 1..n. |
| 6 | **Build output** | Top `maxResults` eligible ETFs become `results`; the rest become `alternatives` (capped at 5, only when `includeAlternatives` is set). |
| 7 | **Warnings** | Emits `NO_ELIGIBLE_RESULTS` and `LOW_CONFIDENCE_RESULTS` advisories. |

### Search providers

`search.AggregatedProvider` wraps the individual providers: parallel fan-out, a provider
error is logged and skipped rather than failing the request, results are deduped and
field-wise merged (first non-empty value wins; longest holdings/sector list wins), and
the merged set is cached in memory for **1 hour** keyed on country + sectors + markets.

| Provider | Source | Notes |
|----------|--------|-------|
| `LiveProvider` | Yahoo Finance `v7/finance/quote` and `v10/finance/quoteSummary` | No API key. Supplies price, AUM, holdings, fund profile. |
| `LiveProvider` | `etf.com/api/screener` | Queried only for non-`ZA` requests. |
| `LiveProvider` | JSE ETF ticker universe | Curated ticker list (STX40, STXNDQ, STX500, STXWDM, STXEMG, STXEUR, STXRES, COREEM) resolved through Yahoo with the `.JO` suffix. Used when country is `ZA` or the market mentions South Africa. |
| `AlphaVantageProvider` | `alphavantage.co` `ETF_PROFILE` + `GLOBAL_QUOTE` | Registered **only** when `ALPHA_VANTAGE_API_KEY` is set to something other than `demo`. |

For `country: "ZA"` the live provider deliberately searches **only** JSE-listed ETFs,
since non-JSE instruments cannot satisfy the TFSA rules.

### Eligibility model

`eligibility.DefaultEngine` is a registry keyed by `Name_Version`. A `Rule` declares what
it applies to and returns a `domain.EligibilityResult` with per-criterion evidence.

**Registered rule:** `TFSA_ZA` / `tfsa_za_v1.0_2025`
(`server/internal/service/eligibility/rules/tfsa_za.go`), based on Income Tax Act 1962
§12T. It checks: JSE listing, ZAR/USD denomination, not leveraged, not inverse, approved
SA provider (Satrix, CoreShares, 1nvest, Cloud Atlas, Absa, Standard Bank, Sygnia,
Ashburton), implicit SARS approval (JSE + approved provider), and replication method.

**Fallback when no rule matches:** `accountType: "standard"` → eligible with `medium`
confidence; any tax-advantaged account → `unknown` with `none` confidence (fail-safe).

Status and confidence are both explicit in the response:

| Status | Meaning |
|--------|---------|
| `eligible` | Every checked criterion passed. |
| `conditional` | Eligible, but some criteria were skipped for missing data — verify before investing. |
| `ineligible` | At least one criterion failed outright. |
| `unknown` | Too much missing data (>3 skipped criteria, or low confidence) to make a call. |

Confidence is `high` → `medium` → `low` → `none`, degrading as criteria get skipped.

### Ranking

`ranking.WeightedScorer` scores four components (each 0–1, scaled ×100 for the total):

- **fees** — `1 - TER`, zero once TER ≥ 1.0%
- **liquidity** — stepped on average daily volume (10k / 100k / 500k thresholds)
- **stability** — stepped on AUM (50M / 500M / 1B thresholds)
- **tracking** — stepped on tracking difference; 0.7 neutral when unknown

Default weights are `fees 0.4, liquidity 0.3, tracking 0.2, stability 0.1`. Supplying
`rankingPreferences.weighting` overrides them — and the handler requires those weights to
sum to 1.0 (±0.01).

---

## Getting started

### Prerequisites

- **Go 1.25+** (module declares `go 1.25.5`)
- **Node 18+** with npm (or bun — `bun.lockb` is committed)
- Optional: [`air`](https://github.com/air-verse/air) for Go live reload; `jq` for the
  example scripts

### 1. Run the API

```bash
cd server
go mod download

# Optional: a .env file (or plain environment variables). Without one the
# server runs on :8080 with defaults.
cat > .env <<'EOF'
ENVIRONMENT=development
SERVER_ADDRESS=:8080
ALPHA_VANTAGE_API_KEY=demo
JSE_API_KEY=
JSE_API_BASE_URL=https://api.jse.co.za/v1
CACHE_ENABLED=true
CACHE_TTL_SECONDS=3600
LOG_LEVEL=info
LOG_FORMAT=json
EOF

go run ./cmd/server
# or, with live reload:
air
```

The API listens on 8080 and the Vite dev server on 5173 (`gui/vite.config.ts`).

Verify:

```bash
curl http://localhost:8080/api/v1/health
```

There is also a self-documenting HTML page at `GET /`.

### 2. Run the GUI

```bash
cd gui
npm install

# The SPA calls http://localhost:8080/api/v1 by default; override with
# VITE_UPSTONK_API_URL in gui/.env if the API runs elsewhere.

npm run dev     # http://localhost:5173
```

Other scripts: `npm run build`, `npm run build:dev`, `npm run preview`, `npm run lint`.

### 3. Exercise the API directly

```bash
cd server
./scripts/test_live_data.sh      # health check + live data smoke tests
./examples/test_payloads.sh      # assorted discovery payloads
```

Both scripts assume the API is on `http://localhost:8080`.

---

## API reference

Base URL: `http://localhost:{port}/api/v1`

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/discover` | Main endpoint — full discovery request. |
| `GET` | `/discover/{type}` | Top performers shortcut. `type` ∈ `equity`, `equities`, `stock`, `stocks`, `bond`, `bonds`, `fixed income`, `etf`, `etfs`. Hardcodes a US / standard / USD profile and `maxResults: 20`. |
| `GET` | `/health` | Liveness: status, timestamp, service, version. |
| `GET` | `/` | HTML documentation page. |

### `POST /api/v1/discover`

```bash
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "ZA",
      "accountType": "tfsa",
      "currency": "ZAR",
      "riskTolerance": "moderate",
      "timeHorizonYears": 20
    },
    "exposure": {
      "assets": {
        "companies": ["Apple"],
        "sectors": ["technology"],
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["usa"],
        "emergingMarkets": false,
        "developedMarkets": true
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.75,
      "minAUM": 100000000,
      "allowedExchanges": ["JSE"],
      "excludeSyntheticETFs": true,
      "excludeLeveragedETFs": true,
      "excludeInverseETFs": true,
      "physicalOnly": false,
      "minLiquidity": 10000
    },
    "rankingPreferences": {
      "priority": ["lowest_fees", "liquidity"],
      "weighting": { "fees": 0.4, "liquidity": 0.3, "tracking": 0.2, "stability": 0.1 }
    },
    "outputOptions": {
      "maxResults": 10,
      "includeAlternatives": true,
      "includeSourceLinks": true,
      "explainEligibility": true,
      "includeWarnings": true
    }
  }'
```

Response shape (see `temp.json` in the repo root for a real captured response):

```jsonc
{
  "requestId": "4b7efb3f-…",
  "results": [{
    "ticker": "VGK",
    "ter": 0.06,
    "aum": 37100000000,
    "eligibility": {
      "status": "eligible",            // eligible | conditional | ineligible | unknown
      "isEligible": true,
      "confidence": "medium",          // high | medium | low | none
      "justification": "✓ Listed on JSE; ✓ Currency: ZAR; …",
      "rulesPassed": ["jse_listing"],  // only when explainEligibility is true
      "rulesFailed": [],
      "ruleVersion": "tfsa_za_v1.0_2025"
    },
    "matchScore": 0, "rankingScore": 70.6, "rank": 1,
    "assetBreakdown": { "equities": 0, "bonds": 0, "cash": 0, "commodities": 0, "other": 0 },
    "geographicBreakdown": { "regions": null },
    "topHoldings": [{ "name": "ASML HOLDING NV", "weight": 2.87 }],
    "dataSources": [{ "type": "API", "provider": "Alpha Vantage", "url": "…", "date": "2026-01-11" }]
  }],
  "alternatives": [],
  "summary": {
    "totalSearched": 2, "totalEligible": 2, "totalIneligible": 0, "totalUnknown": 0,
    "searchDurationMs": 0, "dataSourcesQueried": ["JSE", "FactSheets", "Provider APIs"]
  },
  "warnings": [{ "code": "LOW_CONFIDENCE_RESULTS", "message": "…", "severity": "info" }],
  "generatedAt": "2026-01-11T00:24:21Z",
  "cacheHit": false
}
```

### Validation

Struct-level (`validator/v10` tags in `server/internal/api/dto/request.go`):
`country` must be ISO-3166-1 alpha-2, `currency` ISO-4217, `riskTolerance` one of
conservative/moderate/aggressive, `timeHorizonYears` 1–50, `maxTER` 0–5,
`maxResults` 1–100, `investmentVehicles` non-empty.

Business-level (`validateBusinessRules` in the handler):

- Country + account type must be a supported pair:
  **ZA** → `tfsa`, `standard` · **US** → `ira`, `roth_ira`, `401k`, `standard` ·
  **GB** → `isa`, `standard`. Anything else is rejected.
- `investmentVehicles` ⊆ `etf`, `stock`, `bond`, `fund`.
- At least one of companies / sectors / assetClasses / markets must be specified.
- `rankingPreferences.weighting` must sum to 1.0 (±0.01) when supplied.
- `maxResults` defaults to 10 when omitted or 0.

### Errors

```json
{ "error": "CODE", "message": "…", "code": "CODE", "details": {"details": "…"}, "requestId": "…" }
```

| HTTP | Code | Cause |
|------|------|-------|
| 400 | `INVALID_JSON` | Body is not parseable JSON. |
| 400 | `VALIDATION_ERROR` | Struct validation failed; `details` carries per-field messages. |
| 400 | `INVALID_REQUEST` | Business rule failed (unsupported account type, no exposure, bad weights…). |
| 400 | `INVALID_PARAMETER` / `INVALID_TYPE` | Bad `{type}` on the top-performers route. |
| 404 | `NO_RESULTS` | Search returned zero candidates. |
| 400 | `UNSUPPORTED_COUNTRY` | Country not supported by the pipeline. |
| 503 | `DATA_SOURCE_ERROR` | Upstream data source failure. |
| 500 | `INTERNAL_ERROR` | Anything else. |

Every response carries a `requestId` (UUID v4), also attached to the request context.

### Timeouts

Per-request context 30s · server read 15s / write 30s / idle 60s · timeout middleware 30s
· graceful shutdown drains for up to 30s on SIGINT/SIGTERM.

---

## Configuration

### Server (environment variables, optionally from `server/.env`)

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDRESS` | Bind address | `:8080` |
| `ENVIRONMENT` | `development` / `staging` / `production` | `development` |
| `ALPHA_VANTAGE_API_KEY` | Enables the Alpha Vantage provider; the literal `demo` keeps it disabled | `demo` |
| `JSE_API_KEY` | JSE API key (config only — no JSE API client is wired up yet) | — |
| `JSE_API_BASE_URL` | JSE API base URL | `https://api.jse.co.za/v1` |
| `CACHE_ENABLED` | Parsed into config (see Known gaps) | `true` |
| `LOG_LEVEL` | Parsed into config | `info` |
| `LOG_FORMAT` | Parsed into config | `json` |

### GUI (`gui/.env`)

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_UPSTONK_API_URL` | Full API base URL including `/api/v1` | `http://localhost:8080/api/v1` |

---

## Frontend notes

Single-page app; `src/pages/Index.tsx` holds the whole flow as three tabs —
**Investor Profile**, **Exposure**, **Constraints** — plus a results view.

| Piece | Responsibility |
|-------|----------------|
| `stores/discoveryStore.ts` | zustand store: form state, defaults, `submitDiscovery()` which maps UI state → `DiscoveryRequest`, loading/error/response state. |
| `lib/api/upstonk.ts` | `discoverETFs()` fetch wrapper (throws a typed `APIError`), plus `formatCurrency` / `formatPercentage`. |
| `types/api.ts` | TypeScript mirror of the server DTOs, kept separate from the richer UI form types. |
| `components/ResultsTable.tsx` | Ranked results table; row click opens the detail drawer. |
| `components/ETFDetailDrawer.tsx` | Full ETF detail: breakdowns, holdings, eligibility evidence, data sources. |
| `components/EligibilityBadge.tsx` | Status + confidence badge. |
| `components/WarningsPanel.tsx` | Renders `warnings[]` from the response. |
| `components/ui/*` | shadcn/ui primitives (Radix + Tailwind + CVA). |

Stack: React 18, React Router 6, TanStack Query (provider mounted, data fetching
currently goes through the zustand store), Tailwind 3 with `tailwindcss-animate` and
typography, lucide-react icons, sonner toasts, recharts, zod.

---

## Testing & tooling

- **Go:** `go test ./...` runs offline tests at two seams: the HTTP router
  (`internal/api/router`) and `discovery.Service`, with external data replaced by
  `testutil/fakeprovider`. `go test -tags live ./...` adds tests against the real data
  sources; they check shape and provenance only, and skip when offline.
- **Live-data smoke tests:** `server/scripts/test_live_data.sh`, `server/examples/test_payloads.sh`.
- **Frontend:** `npm run lint` (ESLint 9 flat config + typescript-eslint). No test runner configured.
- **Live reload:** `air` (config in `server/.air.toml`, builds to `server/tmp/main`).

---

## Known gaps & gotchas

Things the code does *not* do, or does differently from the older docs in `server/`:

4. **Cache config is inert.** `CACHE_ENABLED` / `CACHE_TTL_SECONDS` are read into
   `config.CacheConfig` but never consumed — the only cache is the hardcoded 1-hour
   in-memory cache inside `search.AggregatedProvider`, and `discovery.Service` hardcodes
   `cacheEnabled: true` while always reporting `cacheHit: false` in the response.
   `LOG_LEVEL` / `LOG_FORMAT` are likewise unused; logging is the stdlib `log` package.
5. **No JSE API client.** `JSE_API_KEY` / `JSE_API_BASE_URL` exist in config only. JSE
   coverage comes from the curated ticker list resolved via Yahoo Finance.
6. **No Dockerfile or Makefile**, despite `server/README.md` and `server/QUICKSTART.md`
   referencing `make build`, `make docker-build`, and a Docker deployment.
7. **`server/setup.sh` is not a script** — it is a byte-identical copy of
   `server/QUICKSTART.md`. Both describe an artifact-assembly process from the original
   scaffold and no longer reflect the repo; treat `server/DOCUMENTATION.md` (endpoint
   examples) and `server/LIVEDATA_SETUP.md` (data sources) as the useful ones.
8. **GUI under-uses the API.** `submitDiscovery()` sends `constraints: {}` and omits
   `rankingPreferences`, so the constraints panel and ranking weights collected in the UI
   never reach the server.
9. **Fuzzy matching is a stub.** `discovery.contains()` only does exact equality, so
   sector / asset-class / company match scoring is effectively exact-match today.
10. **Only one eligibility rule.** Only `TFSA_ZA` is registered (in
    `server/internal/app/engines.go`); those account types validate at the handler but fall through to the
    generic standard/unknown fallback.
11. **Dead code:** `server/internal/service/search/provider_old.go` is entirely
    commented-out stub-provider code.
12. **Lovable placeholders remain.** `gui/index.html` still has the "Lovable App" title
    and lovable.dev Open Graph / Twitter image meta tags.
13. **Yahoo returns only basic quote data.** Yahoo's `v7/finance/quote` and
    `v10/finance/quoteSummary` endpoints now reject unauthenticated calls, so quotes come
    from `v8/finance/chart` (name, currency, exchange, volume). AUM, TER, provider and
    holdings are missing, which is why live TFSA results come back `unknown`. A source for
    those fields still has to be chosen.

### Adding a country rule

```go
// server/internal/service/eligibility/rules/isa_uk.go
type ISAUKRules struct{ version string }

func (r *ISAUKRules) Name() string    { return "ISA_UK" }
func (r *ISAUKRules) Version() string { return "isa_uk_v1.0_2025" }

func (r *ISAUKRules) AppliesTo(country, accountType string) bool {
    return strings.ToUpper(country) == "GB" && strings.ToLower(accountType) == "isa"
}

func (r *ISAUKRules) Evaluate(ctx context.Context, etf domain.ETF) domain.EligibilityResult {
    // Populate RulesPassed / RulesFailed / RulesSkipped, Evidence, Reasons,
    // then set Status + Confidence. Prefer `unknown` over an optimistic guess.
}
```

Then register it in `NewEligibilityEngine()` in `server/internal/app/engines.go` (the
server and the tests both use it):

```go
engine.RegisterRule(rules.NewISAUKRules())
```

### Adding a search provider

Implement `search.Provider` (a single `Search(ctx, Criteria) ([]domain.ETF, error)`),
populate `DataSources` on every ETF you return so provenance survives, and append it to
the slice in `NewSearchProvider()` in `server/internal/app/search.go`. `AggregatedProvider` handles the parallelism,
deduplication, merging, and caching for you.

---

## Disclaimer

Research tooling, not financial advice. Eligibility output is a best-effort reading of
public data and codified rules — always verify with your platform or SARS before
investing.
