# Architecture

```
                    ┌──────────────────────────────────────────┐
  Browser  ───────► │  gui/  Vite + React + zustand            │
                    │  discoveryStore → buildDiscoveryRequest  │
                    └────────────────────┬─────────────────────┘
                                         │ POST /api/v1/discover
                    ┌────────────────────▼─────────────────────┐
                    │  internal/api/router (gorilla/mux)       │
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
         │ Aggregated      │ │ Engine         │ │ WeightedScorer │
         │  ├─ Live        │ │  └─ TFSA_ZA    │ │                │
         │  └─ AlphaVantage│ │                │ │                │
         │  + 1h cache     │ │                │ │                │
         └────────┬────────┘ └────────────────┘ └────────────────┘
                  │
         Yahoo Finance · etf.com · Alpha Vantage
```

The three engines are interfaces (`search.Provider`, `eligibility.Engine`, `ranking.Engine`). They're wired in `server/internal/app`, which both `cmd/server/main.go` and the tests use, so the server and the tests always run the same rules and scorer.

## The discovery pipeline

`discovery.Service.DiscoverETFs` (`server/internal/service/discovery/discovery.go`) runs seven steps:

| # | Step | What happens |
|---|---|---|
| 1 | **Search** | Query every provider in parallel, then dedupe and merge by ticker/ISIN. No candidates → `NoResultsError` (HTTP 404). Reports whether the result came from cache. |
| 2 | **Eligibility** | Evaluate each candidate against the rule whose `AppliesTo(country, accountType)` matches, and tally eligible / ineligible / unknown. |
| 3 | **Constraints** | Drop candidates that fail `maxTER`, `minAUM`, `minLiquidity`, `allowedExchanges`, or the synthetic / leveraged / inverse / physical-only switches. With `tfsaEligibleOnly`, only `eligible` and `conditional` funds are kept; otherwise only `ineligible` ones are dropped. **Unknown AUM or TER (reported as 0) is not treated as failing** `minAUM`/`maxTER`. |
| 4 | **Match score** | How well the fund matches the requested exposure, normalised to 0–100 (50 when no criteria are given): geography 30 pts, sectors 25, asset classes 25, company holdings 20. Sectors, asset classes and companies match case-insensitively as substrings ("tech" matches "Technology"). Regions match by whole name, ignoring case. |
| 5 | **Rank** | `ranking.Engine` gives a quality score; the final order is `0.4 × matchScore + 0.6 × rankingScore`. Ranks are assigned 1..n. |
| 6 | **Build output** | The top `maxResults` eligible funds become `results`. The rest become `alternatives` (at most 5, only with `includeAlternatives`). |
| 7 | **Warnings** | `NO_ELIGIBLE_RESULTS`, `LOW_CONFIDENCE_RESULTS` (any medium/low-confidence result), and `UNVERIFIED_AUM` / `UNVERIFIED_TER` (results kept despite unknown values under an active constraint). |

## Search providers

`search.AggregatedProvider` wraps the individual providers:
- It queries them in parallel. A provider error is logged and skipped rather than failing the request.
- Results are deduped and merged field by field: the first non-empty value wins, and the longest holdings/sector list wins.
- The merged set is cached in memory for **1 hour**, keyed on every search criterion (country, markets, sectors, asset classes, companies, vehicles).

It implements `search.CacheReporter`, which is how the response's `cacheHit` stays truthful.

| Provider | Used when | Details |
|---|---|---|
| `LiveProvider` – JSE | country `ZA`, or a market containing "south africa" | A hardcoded list of JSE ETF tickers, chosen by market and asset class, looked up on Yahoo with the `.JO` suffix. For `ZA`, **only** JSE funds are searched, because nothing else can satisfy the TFSA rules. |
| `LiveProvider` – global | country is not `ZA` | etf.com screener plus a hardcoded US ticker list looked up on Yahoo |
| `AlphaVantageProvider` | `ALPHA_VANTAGE_API_KEY` is set and isn't `demo` | `ETF_PROFILE` + `GLOBAL_QUOTE` |

The current status of each upstream source is in [data-sources](../data-sources/README.md).

`LiveProvider` also pre-filters by asset class, sector and market. When a fund has **no data** for one of these, it is let through rather than treated as a mismatch, and match scoring ranks it.

## Eligibility model

`eligibility.DefaultEngine` is a registry of `Rule`s. A rule declares which country/account pair it applies to, and returns a `domain.EligibilityResult` with per-criterion evidence (passed / failed / skipped).

**Registered rule:** `TFSA_ZA`, version `tfsa_za_v1.0_2025` (`server/internal/service/eligibility/rules/tfsa_za.go`), based on Income Tax Act 1962 §12T. It checks:
1. JSE listing
2. ZAR or USD denomination
3. Not leveraged or inverse
4. An approved SA provider: Satrix, CoreShares, 1nvest, Cloud Atlas, Absa, Standard Bank, Sygnia, Ashburton (matched against `etf.Provider`)
5. Implicit SARS approval (JSE + approved provider)
6. Replication method

**When no rule matches:** a `standard` account → `eligible`, `medium` confidence. Any tax-advantaged account (`isa`, `ira`, `roth_ira`, `401k`) → `unknown`, `none` confidence.

| Status | Meaning |
|---|---|
| `eligible` | Every checked criterion passed. |
| `conditional` | Nothing failed, but some criteria were skipped for missing data. Verify before investing. |
| `ineligible` | At least one criterion failed. |
| `unknown` | More than 3 criteria skipped, or confidence fell to `low`. |

Confidence runs `high` → `medium` → `low` → `none`, dropping as criteria are skipped.

## Ranking

`ranking.WeightedScorer` scores four components, each 0–1. The total is scaled ×100.

| Component | Score |
|---|---|
| `fees` | `1 − TER` (TER in percent); 0 once TER ≥ 1.0% |
| `liquidity` | Average daily volume: <10k → 0.3, <100k → 0.6, <500k → 0.8, otherwise 1.0 |
| `stability` | AUM: <50M → 0.3, <500M → 0.6, <1B → 0.8, otherwise 1.0 |
| `tracking` | Tracking difference: unknown → 0.7, <0.1 → 1.0, <0.3 → 0.8, <0.5 → 0.6, otherwise 0.4 |

Default weights: `fees 0.4, liquidity 0.3, tracking 0.2, stability 0.1`. `rankingPreferences.weighting` replaces them; the handler requires the weights to sum to 1.0 (±0.01). A component missing from a custom weighting contributes 0.

Unknown TER and AUM are scored as if they were real values (see [known gaps](../development/known-gaps.md)).
