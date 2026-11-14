# Known gaps

Things the code doesn't do yet, as of 2026-11-15. The numbers match [plans/gaps-remediation.md](../plans/gaps-remediation.md); gaps 1–8, 11 and 12 are fixed, and 15 onwards were found during that work.

| # | Gap | Notes |
|---|---|---|
| 9 | **No source supplies regional exposure.** | `GeographicExposure.Regions` is never filled, so the geography part of `matchScore` is 0 for live data. The GUI's market codes (`US`, `CA`, …) also need mapping to whatever region names a future source uses. |
| 10 | **Only one eligibility rule.** | Only `TFSA_ZA` is registered. US `ira`/`roth_ira`/`401k` and GB `isa` pass validation but get the fallback: `unknown`, confidence `none`. Planned as Phase 6. |
| 13 | **Yahoo serves only basic quote data.** | `v7/quote` and `v10/quoteSummary` reject unauthenticated calls, so funds have no AUM, TER, provider, holdings or sectors. ZA TFSA results are therefore `unknown`. A source for fund details still has to be chosen. See [data-sources](../data-sources/README.md). |
| 14 | **Ranking treats unknown values as real.** | An unknown TER (0) gets the best fee score; an unknown AUM gets the lowest stability score. Tracking difference already scores a neutral 0.7 when unknown; fees and stability probably should too. |
| 15 | **`maxResults` is effectively required.** | The handler defaults `maxResults` to 10 when it's 0, but struct validation (`min=1`) rejects 0 or a missing value first. Either the validation should allow 0, or the dead default should go. |
| 16 | **Accepted but ignored request fields.** | `currency`, `riskTolerance`, `timeHorizonYears`, `indices`, `excludeCountries`, `emergingMarkets`, `developedMarkets`, `priority` and `includeWarnings` are validated but never read. `investmentVehicles` only affects the cache key. |
| 17 | **`summary.dataSourcesQueried` is a fixed list.** | Always `["JSE","FactSheets","Provider APIs"]`, whatever was actually queried. |
| 18 | **Unreachable error codes.** | `UNSUPPORTED_COUNTRY`, `DATA_SOURCE_ERROR` and `INVALID_PARAMETER` are handled but never produced. Provider failures are logged and skipped, so a total upstream outage shows up as empty or `NO_RESULTS`, never as 503. |
| 19 | **etf.com screener returns 403.** | Non-ZA searches rely only on the hardcoded US ticker list via Yahoo. |
| 20 | **No ranking-weights UI.** | The GUI sends weights from its defaults; nothing in the UI changes them. |
| 21 | **Rule selection order isn't fixed.** | `eligibility.DefaultEngine` iterates a Go map. Harmless with one rule, but once there are several, overlapping rules would be picked at random. Planned for Phase 6. |
| 22 | **No authentication or rate limiting.** | Any client can call the API, and CORS allows any origin. Fine for local use; needs both before being exposed publicly. |
