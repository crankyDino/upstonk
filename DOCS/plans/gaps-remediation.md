# Gaps & Gotchas Remediation — multi-phase TDD plan

## Context
The root README's "Known gaps & gotchas" lists 13 places where UPSTONK is broken, misleading or unfinished: a CORS typo that blocks the GUI, a port collision, a mandatory `.env`, config that nothing reads, fuzzy matching that is really exact equality, a GUI that never sends constraints or ranking weights, `cacheHit` always false, stale scaffold docs, dead code, and only one eligibility rule. The user asked for a multi-phase, test-first plan to fix them, saved under `DOCS/`.

**Status:** approved 2026-11-14. Phases 0–6 done 2026-11-15; open follow-ups are in `DOCS/development/known-gaps.md`. Each phase is one branch and one PR.

## Decisions (from grilling, 2026-11-14)
| Topic | Decision |
|---|---|
| Test seams | **HTTP router** (`httptest`) and **`discovery.Service`** only. Default tests fake external data at the `search.Provider` interface. No other seams get tests. |
| Live data | An **opt-in live tier** at the `discovery.Service` seam with the real providers: `//go:build live`, run with `go test -tags live ./...`. It asserts invariants (shape, provenance, eligibility status), never exact market values. It is not part of the default `go test`. |
| Inert config | **Remove** `CACHE_*`, `LOG_*`, `JSE_*` from config, `.env` examples and docs. |
| `cacheHit` | **Make it truthful.** Real `AggregatedProvider` around a fake leaf provider; the second identical request reports `cacheHit: true`. |
| Matching | **Case-insensitive substring** for sectors, asset classes, companies; case-insensitive keys for geography. |
| Ports | **Vite → 5173**, API stays on 8080. |
| Ranking weights | **GUI adopts the server's 4**: fees, liquidity, tracking, stability (stability = AUM). The GUI "eligibility" weight is dropped. |
| Liquidity | **GUI sends average daily volume** (off / 10k / 100k / 500k) as `minLiquidity`. |
| Docs | **Consolidate** all docs into `DOCS/<subfolder>/`, fill the gaps, and **check each command or claim by running it** before writing it down. |
| ISA / IRA rules | **Own last phase:** research from primary sources first, then build the rules test-first. |

## TDD ground rules (apply to every phase)
- Vertical slices: one failing test → the minimum code to pass it → next test. Don't write all the tests first.
- Tests only at the two agreed seams. Changes outside them (config, GUI, docs) are checked by build, lint, type-check or a manual run, as each phase says.
- Expected values are literal, worked-out numbers, never recomputed the way the code computes them.
- Refactoring happens at review (`/code-review`), not inside the red → green loop.

---

## Phase 0 — Make the seams testable
Goal: get both seams under test with one tracer test each, without changing behaviour.

1. Move `setupRouter` from `server/cmd/server/main.go` into a new package, `server/internal/api/router` (`router.New(h *handlers.DiscoveryHandler) http.Handler`). `main.go` calls it.
2. Add a test-only fake provider, `server/internal/testutil/fakeprovider` (implements `search.Provider`, returns canned `[]domain.ETF`, counts calls). It is the only fake; eligibility and ranking engines are always the real ones.
3. Tracer tests:
   - Router: `GET /api/v1/health` → 200 with `"status"`.
   - Service: `DiscoverETFs` with the fake provider returning 0 ETFs → `*NoResultsError`.

4. **Live tracer** (`server/internal/service/discovery/live_test.go`, `//go:build live`): build the service the way `main.go` does (real `AggregatedProvider` + `LiveProvider`, plus Alpha Vantage if `ALPHA_VANTAGE_API_KEY` is set). A ZA/tfsa request for SA equity returns at least one result; every result has `Exchange == "JSE"`, `TER >= 0`, at least one `DataSource` with a provider and a date, and an eligibility status from the allowed set. The test skips (not fails) when the network is unreachable. This replaces the assertions in `scripts/test_live_data.sh`; the script stays as a curl-based manual smoke test.

Later phases add live assertions where they matter: Phase 2 checks that case-insensitive matching gives a non-zero `matchScore` on real sector names; Phase 6 adds one live check per new rule.

Verify: `cd server && go test ./...` is green and `go vet ./...` is clean (no network needed); `go test -tags live ./internal/service/discovery/` passes when online.

## Phase 1 — Unblock running the app locally (gotchas 1, 2, 3, 13)
| Slice | Seam / check |
|---|---|
| **CORS**: red test — a request with `Origin` gets `Access-Control-Allow-Origin: *`; preflight `OPTIONS /api/v1/discover` → 200 with the header. Fix the typo in `middleware/cors.go`. | Router test |
| **Ports**: `gui/vite.config.ts` port → 5173. | Manual: `npm run dev` serves on 5173 |
| **Optional `.env`**: `config.Load()` logs and continues when `.env` is missing instead of `log.Fatal`. | Manual: start the server with no `.env`, and `/health` returns 200 |
| **Air config**: remove `*/.air.toml` from `.gitignore` and commit `server/.air.toml`, after reading it for machine-specific paths. | `git ls-files server/.air.toml` (done before Phase 0) |
| **Yahoo drift (found by the Phase 0 live tracer)**: `v7/finance/quote` and `v10/finance/quoteSummary` now refuse unauthenticated calls (429/401), so every live search returns 0 ETFs. `v8/finance/chart` still works and gives name, currency (`ZAc` = cents → ZAR), exchange (`JNB` → JSE) and daily volume. Move basic quotes to the chart endpoint, check HTTP status codes, and treat AUM/TER/holdings as missing (eligibility then reports `conditional`/`unknown`, as "never guess" requires) until a source for them is chosen. | Live tracer goes red → green |

End-to-end check: start the API on 8080 and the GUI on 5173, submit the default form, and see results in the browser with no CORS error.

## Phase 2 — Discovery correctness (gotchas 4 [cacheHit], 9)
All tests go through `discovery.Service.DiscoverETFs` with the fake provider (or the real `AggregatedProvider` wrapping it).

Slices, in order:
1. Sector `"tech"` matches an ETF with sector `"Technology"` 100% → `matchScore` is 100 when sector is the only criterion.
2. Company `"apple"` matches holding `"APPLE INC"` at weight 7.0 → `matchScore` = 7.
3. Asset class `"Equity"` matches `"equity"`.
4. Geography `"usa"` matches region key `"USA"`. First record the region vocabulary the live providers actually emit (`search/provider.go`, `aggregated.go`); if GUI codes (`US`, `CA`) don't match that vocabulary, write it down as a follow-up and don't invent a mapping.
5. **cacheHit:** two identical requests through the real `AggregatedProvider` → the first `CacheHit == false`, the second `== true`, and the fake leaf is called once. Implementation: the aggregated provider exposes hit/miss without changing `search.Provider`, either through an optional interface that `discovery.Service` type-asserts, or through a result struct. Pick whichever keeps the `Provider` interface intact.
6. Pin down the existing behaviour that Phase 4 depends on: `maxTER`, `minAUM`, `minLiquidity`, and exclusion of synthetic / leveraged / inverse each drop the right fixture ETF; custom `weighting` changes the order between two fixture ETFs (worked out by hand).

Delete `discovery.contains()` and the hardcoded `cacheEnabled` field once nothing uses them.

**Findings while doing Phase 2:**
- The cache key ignored asset classes, companies and vehicles, so an "equity" search could be answered with a cached "bond" search. Fixed: the key now covers every criterion (test: `TestRequestsForDifferentAssetClassesAreNotServedFromEachOthersCache`).
- No provider fills `GeographicExposure.Regions`, so live geography scores are always 0. Left as a follow-up (README gotcha 9): it needs a data source for regional exposure and a GUI-code → region mapping. Nothing was invented.
- The planned live check for sector matching was skipped: since the Yahoo drift, no live source returns sector data.

## Phase 3 — Remove inert config (gotchas 4 [config], 5)
- Delete `CacheConfig`, `LoggingConfig` and `JSEAPIConfig` from `server/internal/config/config.go`, along with their env reads.
- Delete `server/internal/service/search/provider_old.go` (gotcha 11).
- Update the `.env` examples in the README (and later in `DOCS/`).

Verify: `go build ./... && go vet ./... && go test ./...`; grep finds no `CACHE_`, `LOG_LEVEL`, `LOG_FORMAT` or `JSE_API` left in `server/` or the docs.

## Phase 4 — GUI sends the full request (gotchas 8, 12)
Contract check at the router seam: add the JSON body the GUI will send with its default state as a fixture, `server/internal/api/router/testdata/gui_default_request.json`. Router test: `POST /discover` with it (fake provider) → 200, not 400. Write this test first; it fails until the fixture has the full shape.

GUI changes (checked with `npm run lint`, `npx tsc --noEmit` and a manual run, since the store isn't a test seam):
- `gui/src/types/api.ts`: `ConstraintsRequest` and `RankingPreferencesRequest` mirror `server/internal/api/dto/request.go`.
- `gui/src/stores/discoveryStore.ts` `submitDiscovery()` maps:
  - `excludeSynthetic → excludeSyntheticETFs`, `excludeLeveraged → excludeLeveragedETFs`, `excludeInverse → excludeInverseETFs`, `physicalReplicationOnly → physicalOnly`, `liquidityThreshold → minLiquidity`, plus `tfsaEligibleOnly`, `maxTER` and `minAUM` as they are.
  - Ranking weights → `rankingPreferences.weighting` `{fees, liquidity, tracking, stability}`, normalised to sum to 1.0.
  - Also send `riskTolerance`, `timeHorizonYears`, `sectors`, `companies`, `includeAlternatives` and `explainEligibility`, which the form already collects.
- `ConstraintsPanel.tsx`: replace the liquidity % slider with volume steps (off / 10k / 100k / 500k).
- Ranking weights: four (`fees`, `liquidity`, `tracking`, `stability`) in the store defaults; `aum` and `eligibility` removed. (No ranking panel exists in the UI.)
- `gui/index.html`: real title and meta tags, with the lovable.dev images removed.

End-to-end check: in the browser, setting max TER to 0.1% visibly shrinks the results; changing the weights changes the order.

**Findings and decisions while doing Phase 4 (2026-11-15):**
- The GUI offered countries (CA/AU/DE/FR) and account values (`TFSA`, `ISA`, `IRA`, `retirement_annuity`) the server rejects, and labelled TFSA as Canadian. Decision: **offer only the server's supported pairs** (ZA/US/GB), in lower case, with the account list depending on the country.
- There is no ranking-weights panel in the GUI; the four weights are sent from defaults only.
- Live data has no AUM or TER, so the default `minAUM` dropped every live ETF. Decision: **unknown AUM passes `minAUM` and is flagged `UNVERIFIED_AUM`**. The same rule was applied to TER (`UNVERIFIED_TER`). The planned "max TER 0.1% shrinks results" check therefore can't show a change with live data today.
- The contract fixtures are generated from the GUI's real code (`npm run contract:fixture`), not written by hand. The router tests post the default request and every account pair the GUI offers.
- Follow-up (README gotcha 14): ranking scores unknown TER as free and unknown AUM as tiny.

## Phase 5 — Consolidate docs into `DOCS/` (gotchas 6, 7)
Rule: **before a command, endpoint or behaviour goes into the docs, run it and confirm it works.** Anything that can't be confirmed is left out or marked as not working.

Proposed layout:
```
DOCS/
├── README.md                index + reading order
├── getting-started/         prerequisites, run API, run GUI, env vars
├── architecture/            pipeline, providers, eligibility model, ranking
├── api/                     endpoint reference, request/response, validation, errors
├── data-sources/            from server/LIVEDATA_SETUP.md, re-verified
├── development/             testing (seams, fakes), air, scripts, adding a rule/provider
└── plans/                   this plan + future plans
```
Sources to merge, compare and then delete: `server/README.md`, `server/QUICKSTART.md`, `server/setup.sh` (copy of QUICKSTART), `server/DOCUMENTATION.md`, `server/LIVEDATA_SETUP.md`, `gui/README.md` (Lovable boilerplate), and the long sections of the root README. Where sources disagree, the code decides. The root README becomes a short overview that links into `DOCS/`. Fix the hardcoded port in `server/examples/test_payloads.sh` and `scripts/test_live_data.sh`, and confirm both run. No Makefile or Dockerfile, so every reference to them is removed.

Also: update the HTML page at `GET /` (`serveDocumentation` in main.go) so its country list matches reality.

**Findings while doing Phase 5 (2026-11-15):** each documented request and error case was run against a live server. Things the old docs got wrong, now recorded in `DOCS/development/known-gaps.md` (15–22):
- `maxResults` can't be left out (validation runs before the default);
- nine request fields are validated but never read;
- `dataSourcesQueried` is a fixed list;
- `UNSUPPORTED_COUNTRY` and `DATA_SOURCE_ERROR` are never produced;
- etf.com returns 403;
- no auth or rate limiting.

None of the old TFSA examples returns results with today's live data. The scripts now take `API_URL` and both run cleanly.

## Phase 6 — UK ISA and US IRA / Roth IRA / 401k rules (gotcha 10)
1. **Research** (`/research` skill): collect the primary-source rules for each account type (HMRC ISA regulations; IRS Publication 590-A/B and 401(k) plan rules) into `DOCS/research/`. Which facts can be checked against ETF data we actually have (exchange, domicile, structure, currency)?
2. **ADR** in `DOCS/architecture/adr/`: how to handle rules whose real constraint is the account provider rather than the ETF. Default: `conditional` plus an explanation, following "never guess".
3. **Make rule selection deterministic**: `eligibility.DefaultEngine` currently iterates a Go map. Once there is more than one rule, the matching rule has to be selected in a fixed order.
4. **TDD each rule** at the `discovery.Service` seam (real engine, fake provider): for example, a GB/isa request with a UCITS ETF on the LSE → expected status and confidence; a leveraged ETF → ineligible. Register the rules in `main.go`.
5. Update the docs and remove gotcha 10.

**Done 2026-11-15.** Research: `DOCS/research/etf-eligibility-isa-ira-401k.md`. ADR: `DOCS/architecture/adr/0001-provider-dependent-eligibility.md`. Decisions made along the way:
- **401(k) is always `unknown`** with a plan-menu reason, an exception to the ADR, because `conditional` would set `isEligible`.
- **Structure flags became tri-state (`*bool`)**, because the research showed missing data was being read as `false`. Consequences: TFSA leverage/inverse checks are skipped without data; structure constraints keep and flag (`UNVERIFIED_STRUCTURE`); merges take the first known flag; providers stop inventing flags (every live ETF had been "physical").
- **Rules built now with fixture data**, so they tighten automatically once a source supplies domicile and legal structure.

Remaining open items are known gaps 23–26.

## Phase 7 — Small open gaps (known gaps 14, 15, 17, 18, 20)
Added 2026-11-15; done on the single `dev` branch.

| Gap | Change | Seam / check |
|---|---|---|
| 14 | Unknown TER or AUM (0) scores a **neutral 0.5** for fees / stability, instead of best / worst. | `discovery.Service` |
| 15 | `maxResults` may be omitted or 0; the handler default (10) then applies. | Router |
| 17 | `summary.dataSourcesQueried` lists the providers that actually contributed data (from each candidate's provenance), not a fixed list. | `discovery.Service` |
| 18 | When every data source fails, respond `503 DATA_SOURCE_ERROR` instead of an empty or 404 result. Remove the unreachable `UNSUPPORTED_COUNTRY` and `INVALID_PARAMETER` paths. | Router |
| 20 | Ranking-weights panel in the GUI: four 0–100 sliders (fees, liquidity, tracking, fund size) showing each one's resulting share; the builder already normalises. | lint + type-check + contract fixture |

Decisions (2026-11-15): neutral 0.5 for unknown values; sliders, auto-normalised.

---

## Gotcha → phase map
| # | Gotcha | Phase |
|---|---|---|
| 1 | CORS typo | 1 |
| 2 | Port collision | 1 |
| 3 | `.env` mandatory | 1 |
| 4 | Inert cache/log config, false `cacheHit` | 2 (cacheHit), 3 (config) |
| 5 | No JSE client (JSE config) | 3 |
| 6 | Makefile/Docker references | 5 |
| 7 | setup.sh / stale docs | 5 |
| 8 | GUI drops constraints/ranking | 4 |
| 9 | Fuzzy matching stub | 2 |
| 10 | Only one eligibility rule | 6 |
| 11 | `provider_old.go` dead code | 3 |
| 12 | Lovable placeholders | 4 |
| 13 | `.air.toml` gitignored | 1 |

## Verification (whole plan)
- `cd server && go test ./... && go vet ./...` is green after every phase.
- `go test -tags live ./...` passes when online, run at the end of each phase. It doesn't block when upstream sources are down; failures there are triaged as data-source drift.
- `cd gui && npm run lint && npx tsc --noEmit` is clean after Phase 4.
- Manual end-to-end after Phases 1 and 4: API on :8080, GUI on :5173, form submission returns results that respect the constraints.
- The README's gotcha list shrinks phase by phase; after Phase 6 only deliberate limitations remain.
