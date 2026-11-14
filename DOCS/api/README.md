# API reference

Base URL: `http://localhost:8080/api/v1`. Every response, including errors, carries a `requestId` (UUID v4). CORS allows any origin.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/discover` | Full discovery request |
| `GET` | `/discover/{type}` | Shortcut for a US / standard / USD profile with `maxResults: 20`. `type` ∈ `equity`, `equities`, `stock`, `stocks`, `bond`, `bonds`, `fixed income`, `etf`, `etfs` |
| `GET` | `/health` | `{"service":"upstonk-api","status":"healthy","timestamp":…,"version":"1.0.0"}` |
| `GET` | `/` | HTML page describing the API |

## `POST /discover`

```bash
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": { "country": "US", "accountType": "standard", "currency": "USD" },
    "exposure": {
      "assets": { "sectors": ["technology"] },
      "geography": { "markets": ["usa"] }
    },
    "investmentVehicles": ["etf"],
    "constraints": { "maxTER": 0.5, "excludeLeveragedETFs": true },
    "rankingPreferences": { "weighting": { "fees": 0.4, "liquidity": 0.3, "tracking": 0.2, "stability": 0.1 } },
    "outputOptions": { "maxResults": 5, "includeAlternatives": true, "includeSourceLinks": true, "explainEligibility": true }
  }'
```

On 2026-11-15 this returned five US funds (IVV, VGT, SOXX, SPY, VOO), all `eligible`, with `UNVERIFIED_TER` and `LOW_CONFIDENCE_RESULTS` warnings. TER isn't available from live data at the moment; see [data-sources](../data-sources/README.md).

### Request fields

"Effect" says whether the field changes the result today. Fields marked *none* are accepted and validated, but nothing reads them yet.

**`investorProfile`**

| Field | Type | Rules | Effect |
|---|---|---|---|
| `country` | string | required, ISO-3166-1 alpha-2, and must be `ZA`, `US` or `GB` | Picks providers and eligibility rule |
| `accountType` | string | required; must be supported for the country (below) | Picks eligibility rule |
| `currency` | string | required, ISO-4217 | *none* |
| `riskTolerance` | string | optional: `conservative`, `moderate`, `aggressive` | *none* |
| `timeHorizonYears` | int | optional, 1–50 | *none* |

Supported pairs: **ZA** → `tfsa`, `standard` · **US** → `ira`, `roth_ira`, `401k`, `standard` · **GB** → `isa`, `standard`. Values are case-sensitive and lower case.

**`exposure`**. At least one of companies, sectors, assetClasses or markets is required.

| Field | Type | Effect |
|---|---|---|
| `assets.companies` | string[] | Match score (case-insensitive substring of holding names) |
| `assets.sectors` | string[] | Provider pre-filter, ticker selection, match score |
| `assets.assetClasses` | string[] | Provider pre-filter, ticker selection, match score |
| `assets.indices` | string[] | *none* |
| `geography.markets` | string[] | Ticker selection, provider pre-filter, match score |
| `geography.emergingMarkets` | bool | *none* |
| `geography.developedMarkets` | bool | *none* |
| `geography.excludeCountries` | string[] | *none* |

**`investmentVehicles`**. Required, non-empty, each one of `etf`, `stock`, `bond`, `fund`. It's only part of the cache key; the search doesn't filter on it.

**`constraints`**

| Field | Type | Effect |
|---|---|---|
| `tfsaEligibleOnly` | bool | `true`: keep only `eligible`/`conditional`. `false`: drop only `ineligible`. |
| `maxTER` | float, 0–5 | Drop funds with a known TER above it (percent). Unknown TER passes and is flagged `UNVERIFIED_TER`. |
| `minAUM` | float | Drop funds with a known AUM below it. Unknown AUM passes and is flagged `UNVERIFIED_AUM`. |
| `minLiquidity` | float | Drop funds whose average daily volume (shares) is below it |
| `allowedExchanges` | string[] | Keep only these exchanges (case-insensitive) |
| `excludeSyntheticETFs` / `excludeLeveragedETFs` / `excludeInverseETFs` | bool | Drop those structures |
| `physicalOnly` | bool | Keep only physical replication |

**`rankingPreferences`**

| Field | Type | Effect |
|---|---|---|
| `weighting` | object | Replaces the default weights. Keys: `fees`, `liquidity`, `tracking`, `stability`. Must sum to 1.0 (±0.01). |
| `priority` | string[] | Validated (`lowest_fees`, `tracking_accuracy`, `liquidity`, `diversification`, `tax_efficiency`); *none* |

**`outputOptions`**

| Field | Type | Effect |
|---|---|---|
| `maxResults` | int, 1–100 | **Required.** Leaving it out or sending 0 fails validation. |
| `includeAlternatives` | bool | Include up to 5 non-result funds as `alternatives` |
| `includeSourceLinks` | bool | Include `dataSources` on each fund |
| `explainEligibility` | bool | Include `rulesPassed`, `rulesFailed` and eligibility `warnings` |
| `includeWarnings` | bool | *none* (top-level warnings are always returned) |

### Response

A real response (2026-11-15) to the GUI's default request (`server/internal/api/router/testdata/gui_default_request.json`), cut down to one result:

```jsonc
{
  "requestId": "89fb6e6a-70b3-43af-89c0-79646a66fb74",
  "results": [{
    "ticker": "STX500", "name": "Satrix S&P 500 ETF", "isin": "", "exchange": "JSE", "provider": "",
    "assetClass": "", "trackingIndex": "", "geographicFocus": "",
    "ter": 0, "aum": 0,                       // 0 = not reported by any source
    "currency": "ZAR", "averageDailyVolume": 188645,
    "eligibility": {
      "status": "eligible",                   // eligible | conditional | ineligible | unknown
      "isEligible": true,
      "confidence": "medium",                 // high | medium | low | none
      "justification": "Standard account - no specific eligibility restrictions; ",
      "rulesPassed": ["standard_account"]     // rulesPassed/rulesFailed/warnings only with explainEligibility;
                                              // empty lists and ruleVersion are omitted
    },
    "matchScore": 0, "rankingScore": 81, "rank": 1,
    "assetBreakdown": { "equities": 0, "bonds": 0, "cash": 0, "commodities": 0, "other": 0 },
    "geographicBreakdown": { "regions": null },
    // "topHoldings" is omitted when empty
    "dataSources": [                          // only with includeSourceLinks
      { "type": "API", "provider": "Yahoo Finance", "url": "https://finance.yahoo.com/quote/STX500.JO", "date": "2026-11-15" },
      { "type": "ExchangeListing", "provider": "JSE", "url": "https://www.jse.co.za/trade/etfs", "date": "2026-11-15" }
    ]
  }],
  // "alternatives" is omitted when empty
  "summary": {
    "totalSearched": 2, "totalEligible": 2, "totalIneligible": 0, "totalUnknown": 0,
    "searchDurationMs": 787,
    "dataSourcesQueried": ["JSE", "FactSheets", "Provider APIs"]   // fixed list, not the sources actually used
  },
  "warnings": [
    { "code": "UNVERIFIED_AUM", "message": "Fund size (AUM) is unavailable for STX500, STXNDQ, so the minimum AUM constraint could not be checked for them.", "severity": "warning" },
    { "code": "UNVERIFIED_TER", "message": "Total expense ratio (TER) is unavailable for STX500, STXNDQ, so the maximum TER constraint could not be checked for them.", "severity": "warning" },
    { "code": "LOW_CONFIDENCE_RESULTS", "message": "2 results have medium or low confidence. Verify eligibility with your platform before investing.", "severity": "info" }
  ],
  "generatedAt": "2026-11-14T22:04:47Z",
  "cacheHit": false                           // true when served from the 1-hour cache
}
```

| Warning code | Severity | When |
|---|---|---|
| `NO_ELIGIBLE_RESULTS` | warning | `results` is empty |
| `LOW_CONFIDENCE_RESULTS` | info | Any result has `medium` or `low` confidence |
| `UNVERIFIED_AUM` | warning | `minAUM` set and some results have unknown AUM |
| `UNVERIFIED_TER` | warning | `maxTER` set and some results have unknown TER |

## Errors

```json
{ "error": "CODE", "message": "…", "code": "CODE", "details": { "details": "…" }, "requestId": "…" }
```

Every code below was triggered against a running server on 2026-11-15, except `NO_RESULTS` and `INTERNAL_ERROR`. `NO_RESULTS` is covered by tests: live searches always fall back to some tickers, so it's hard to trigger.

| HTTP | Code | Cause (observed message) |
|---|---|---|
| 400 | `INVALID_JSON` | Body isn't JSON ("Failed to parse request body") |
| 400 | `VALIDATION_ERROR` | Struct validation; `details` maps Go field names to messages, e.g. `{"Country":"This field is required"}`, `{"MaxResults":"Must be at least 1"}` |
| 400 | `INVALID_REQUEST` | Business rules: "account type 'ira' is not supported for country 'ZA'", "at least one exposure criterion must be specified", "unsupported investment vehicle: crypto", "ranking weights must sum to 1.0, got 0.50" |
| 400 | `INVALID_TYPE` | Unknown `{type}` on `GET /discover/{type}` |
| 404 | `NO_RESULTS` | The search returned no candidates |
| 500 | `INTERNAL_ERROR` | Anything else, e.g. a search failure |

`UNSUPPORTED_COUNTRY`, `DATA_SOURCE_ERROR` and `INVALID_PARAMETER` exist in the handler, but nothing currently produces them. Unsupported countries come back as `INVALID_REQUEST`, and upstream failures are logged and skipped.

## Timeouts

Per-request context 30 s · server read 15 s / write 30 s / idle 60 s · timeout middleware 30 s · graceful shutdown waits up to 30 s on SIGINT/SIGTERM.
