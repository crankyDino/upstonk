# Development

## Repo layout

```
upstonk/
├── DOCS/                this documentation
├── server/              Go REST API (module `upstonk`, Go 1.25.5)
│   ├── cmd/server/      main.go — config load, server start, graceful shutdown
│   ├── internal/
│   │   ├── api/
│   │   │   ├── dto/         request/response contracts + validation tags
│   │   │   ├── handlers/    DiscoveryHandler (discover, top performers, health)
│   │   │   ├── middleware/  logging, recovery, CORS, timeout
│   │   │   └── router/      route table + HTML page at / (test seam)
│   │   ├── app/         engine + search provider wiring shared by server and tests
│   │   ├── config/      SERVER_ADDRESS, ALPHA_VANTAGE_API_KEY (optional .env)
│   │   ├── domain/      ETF, EligibilityResult, RankingScore, DataSource
│   │   ├── service/
│   │   │   ├── discovery/    7-step pipeline (test seam)
│   │   │   ├── search/       Provider interface, Live, AlphaVantage, Aggregated (+ cache)
│   │   │   ├── eligibility/  rule engine + rules/tfsa_za.go
│   │   │   └── ranking/      WeightedScorer
│   │   └── testutil/fakeprovider/   test double for search.Provider
│   ├── examples/test_payloads.sh    curl payloads against a running server
│   ├── scripts/test_live_data.sh    live smoke test against a running server
│   └── .air.toml                    air live-reload config (builds to tmp/main)
└── gui/                 Vite + React 18 + TypeScript SPA (port 5173)
    ├── scripts/write-contract-fixture.ts
    └── src/
        ├── components/      feature components (RankingWeightsPanel = weight sliders); shadcn/ui primitives in ui/
        ├── lib/api/         upstonk.ts (fetch client), buildRequest.ts (UI state → request)
        ├── lib/discoveryOptions.ts   supported country/account pairs, defaults
        ├── stores/discoveryStore.ts  zustand: form state + submit
        ├── types/api.ts     TS mirror of the server DTOs
        └── pages/           Index.tsx (the whole app), NotFound.tsx
```

## Tests

```bash
cd server
go test ./...              # offline; no network needed
go test -tags live ./...   # adds tests against the real data sources
go vet ./... && go vet -tags live ./...
```

Tests only run at two **seams** (public entry points):

| Seam | Location | What it covers |
|---|---|---|
| HTTP router | `internal/api/router/router_test.go` | Real router, handler, service and engines through `httptest`: health, CORS, preflight, and the GUI contract |
| `discovery.Service` | `internal/service/discovery/discovery_test.go` | `DiscoverETFs` end to end: matching, constraints, unknown-value flags, ranking order, cache behaviour |

External data is replaced by `testutil/fakeprovider`, which returns fixed ETFs and counts calls. Everything else is real: eligibility rules, scorer, and the aggregated provider and its cache. Expected values in tests are hand-worked literals.

**Live tier** (`internal/service/discovery/live_test.go`, build tag `live`): runs the production wiring against the real sources. It checks only things that don't change day to day (JSE-only results for ZA, data sources present, valid eligibility statuses), never market values, and skips itself when the network is unreachable. A failure here usually means an upstream source changed; see [data-sources](../data-sources/README.md).

**GUI:** no test runner. Check with `npm run lint` and `npx tsc --noEmit -p tsconfig.app.json`. The lint errors left in `components/ui/*` and `tailwind.config.ts` are shadcn boilerplate.

## GUI ↔ server contract

`gui/src/lib/api/buildRequest.ts` turns form state into the request, and `gui/src/lib/discoveryOptions.ts` lists the country/account pairs the GUI offers. `npm run contract:fixture` (in `gui/`) runs both with the real defaults and writes:

- `server/internal/api/router/testdata/gui_default_request.json`
- `server/internal/api/router/testdata/gui_account_pairs.json`

The router tests POST these and fail if the server rejects the default request or any pair the GUI offers. **Rerun `npm run contract:fixture` whenever you change the builder, the defaults or the supported pairs, and commit the fixtures.**

## Adding a country rule

Research the law first (primary sources, into `DOCS/research/`), then follow `rules/isa_uk.go` as the model:

```go
func (r *MyRules) Evaluate(_ context.Context, etf domain.ETF) domain.EligibilityResult {
    b := newResult(r.version)          // rules/result.go
    // Legal criteria: b.pass / b.fail (only on data we have) / b.skip (missing data).
    // Platform or plan constraints: b.providerWarning — caps the result at conditional.
    return b.finish()
}
```

Structure flags are `*bool`: use `domain.IsTrue` / `domain.IsFalse`, and treat `nil` as unknown. Write the tests first at the `discovery.Service` seam (see `discovery_test.go`, e.g. the ISA tests), and add a live check in `live_test.go`.

Register it in `NewEligibilityEngine()` in `server/internal/app/engines.go`; the server and the tests both pick it up. The country/account pair must also be in `isAccountTypeSupported` (`internal/api/handlers/discovery.go`) and in `SUPPORTED_ACCOUNTS` (`gui/src/lib/discoveryOptions.ts`). The contract test catches the GUI and server disagreeing.

## Adding a search provider

Implement `search.Provider` (`Search(ctx, Criteria) ([]domain.ETF, error)`) and set `DataSources` on every ETF you return, so provenance survives. Then add it to the list in `NewSearchProvider()` in `server/internal/app/search.go`. `AggregatedProvider` handles the parallel calls, deduplication, merging and caching.

Report missing values as zero/empty, never as made-up defaults. The pipeline treats zero AUM/TER as unknown.
