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
         │  ├─ Live        │ │  ├─ TFSA_ZA    │ │                │
         │  └─ AlphaVantage│ │  ├─ ISA_UK     │ │                │
         │                 │ │  ├─ IRA_US     │ │                │
         │                 │ │  └─ 401K_US    │ │                │
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
| 1 | **Search** | Query every provider in parallel, then dedupe and merge by ticker/ISIN. Every provider failing → `DataSourceError` (HTTP 503); no candidates → `NoResultsError` (HTTP 404). Reports whether the result came from cache. |
| 2 | **Eligibility** | Evaluate each candidate against the rule whose `AppliesTo(country, accountType)` matches, and tally eligible / ineligible / unknown. |
| 3 | **Constraints** | Drop candidates that fail `maxTER`, `minAUM`, `minLiquidity`, `allowedExchanges`, or the synthetic / leveraged / inverse / physical-only switches. With `tfsaEligibleOnly`, only `eligible` and `conditional` funds are kept; otherwise only `ineligible` ones are dropped. **Unknown AUM or TER (reported as 0) is not treated as failing** `minAUM`/`maxTER`. |
| 4 | **Match score** | How well the fund matches the requested exposure, normalised to 0–100 (50 when no criteria are given): geography 30 pts, sectors 25, asset classes 25, company holdings 20. Sectors, asset classes and companies match case-insensitively as substrings ("tech" matches "Technology"). Regions match by whole name, ignoring case. |
| 5 | **Rank** | `ranking.Engine` gives a quality score; the final order is `0.4 × matchScore + 0.6 × rankingScore`. Ranks are assigned 1..n. |
| 6 | **Build output** | The top `maxResults` eligible funds become `results`. The rest become `alternatives` (at most 5, only with `includeAlternatives`). |
| 7 | **Warnings** | `NO_ELIGIBLE_RESULTS`, `LOW_CONFIDENCE_RESULTS` (any medium/low-confidence result), and `UNVERIFIED_AUM` / `UNVERIFIED_TER` (results kept despite unknown values under an active constraint). |

## Search providers

`search.AggregatedProvider` wraps the individual providers:
- It queries them in parallel. A provider error is logged and skipped rather than failing the request; only when **every** provider fails does the search fail (an outage, never cached). `LiveProvider` applies the same rule to its own sub-sources.
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

`eligibility.DefaultEngine` is a registry of `Rule`s, checked in registration order (the first one that applies wins). A rule declares which country/account pair it applies to, and returns a `domain.EligibilityResult` with per-criterion evidence (passed / failed / skipped).

Structure flags (`isPhysical`, `isSynthetic`, `isLeveraged`, `isInverse`) are tri-state. A flag no source reported is unknown, never `false`: the TFSA leverage/inverse criteria are skipped, and structure constraints keep the fund and raise `UNVERIFIED_STRUCTURE`.

| Rule | Applies to | Version | Outcome in short |
|---|---|---|---|
| `TFSA_ZA` | ZA `tfsa` | `tfsa_za_v1.0_2025` | Details below |
| `ISA_UK` | GB `isa` | `isa_uk_v1.0_2026` | Qualifying route (UK/recognised UCITS, NURS, or recognised-exchange listing) can't be fully confirmed from our data → `conditional`; no listing or structure data → `unknown`; `ineligible` only for a known non-UCITS fund quoted only off-exchange (e.g. OTC). US-listed funds warn that UK platforms can't sell them to retail investors without a product summary/KID. |
| `IRA_US` | US `ira`, `roth_ira` | `ira_us_v1.0_2026` | No ETF fails IRA law → always `conditional` (custodian decides), with notes for metal-backed (collectibles), non-US-domiciled, partnership and leveraged/inverse funds. |
| `401K_US` | US `401k` | `k401_us_v1.0_2026` | Always `unknown`, confidence `none`: only the plan menu decides. |

The ISA, IRA and 401(k) rules follow [ADR 0001](adr/0001-provider-dependent-eligibility.md) and cite [the research](../research/etf-eligibility-isa-ira-401k.md).

**`TFSA_ZA`**, version `tfsa_za_v1.0_2025` (`server/internal/service/eligibility/rules/tfsa_za.go`), based on Income Tax Act 1962 §12T. It checks:
1. JSE listing
2. ZAR or USD denomination
3. Not leveraged or inverse (skipped when no source reports it)
4. An approved SA provider: Satrix, CoreShares, 1nvest, Cloud Atlas, Absa, Standard Bank, Sygnia, Ashburton (matched against `etf.Provider`)
5. Implicit SARS approval (JSE + approved provider)
6. Replication method

**When no rule matches:** a `standard` account → `eligible`, `medium` confidence; any other account → `unknown`, `none` confidence. Every supported tax-advantaged account now has a rule.

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
| `fees` | `1 − TER` (TER in percent); 0 once TER ≥ 1.0%; **0.5 when TER is unknown** |
| `liquidity` | Average daily volume: <10k → 0.3, <100k → 0.6, <500k → 0.8, otherwise 1.0 |
| `stability` | AUM: unknown → 0.5, <50M → 0.3, <500M → 0.6, <1B → 0.8, otherwise 1.0 |
| `tracking` | Tracking difference: unknown → 0.7, <0.1 → 1.0, <0.3 → 0.8, <0.5 → 0.6, otherwise 0.4 |

Default weights: `fees 0.4, liquidity 0.3, tracking 0.2, stability 0.1`. `rankingPreferences.weighting` replaces them; the handler requires the weights to sum to 1.0 (±0.01). A component missing from a custom weighting contributes 0.

Unknown values never count as good or bad: unknown TER and AUM score a neutral 0.5, and unknown tracking difference 0.7.
