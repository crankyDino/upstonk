# Known gaps

Things the code doesn't do yet, as of 2026-11-15. The numbers match [plans/gaps-remediation.md](../plans/gaps-remediation.md); gaps 1–8, 10–12 and 21 are fixed, and 15 onwards were found during that work.

| # | Gap | Notes |
|---|---|---|
| 9 | **No source supplies regional exposure.** | `GeographicExposure.Regions` is never filled, so the geography part of `matchScore` is 0 for live data. The GUI's market codes (`US`, `CA`, …) also need mapping to whatever region names a future source uses. |
| 13 | **Yahoo serves only basic quote data.** | `v7/quote` and `v10/quoteSummary` reject unauthenticated calls, so funds have no AUM, TER, provider, holdings or sectors. ZA TFSA results are therefore `unknown`. A source for fund details still has to be chosen. See [data-sources](../data-sources/README.md). |
| 14 | **Ranking treats unknown values as real.** | An unknown TER (0) gets the best fee score; an unknown AUM gets the lowest stability score. Tracking difference already scores a neutral 0.7 when unknown; fees and stability probably should too. |
| 15 | **`maxResults` is effectively required.** | The handler defaults `maxResults` to 10 when it's 0, but struct validation (`min=1`) rejects 0 or a missing value first. Either the validation should allow 0, or the dead default should go. |
| 16 | **Accepted but ignored request fields.** | `currency`, `riskTolerance`, `timeHorizonYears`, `indices`, `excludeCountries`, `emergingMarkets`, `developedMarkets`, `priority` and `includeWarnings` are validated but never read. `investmentVehicles` only affects the cache key. |
| 17 | **`summary.dataSourcesQueried` is a fixed list.** | Always `["JSE","FactSheets","Provider APIs"]`, whatever was actually queried. |
| 18 | **Unreachable error codes.** | `UNSUPPORTED_COUNTRY`, `DATA_SOURCE_ERROR` and `INVALID_PARAMETER` are handled but never produced. Provider failures are logged and skipped, so a total upstream outage shows up as empty or `NO_RESULTS`, never as 503. |
| 19 | **etf.com screener returns 403.** | Non-ZA searches rely only on the hardcoded US ticker list via Yahoo. |
| 20 | **No ranking-weights UI.** | The GUI sends weights from its defaults; nothing in the UI changes them. |
| 22 | **No authentication or rate limiting.** | Any client can call the API, and CORS allows any origin. Fine for local use; needs both before being exposed publicly. |
| 23 | **No domicile, legal structure or FCA-register data.** | No source fills `domicile` or `legalStructure`, so the ISA rule can only use the exchange, and no ISA result can be better than `conditional`. Whether a UCITS ETF is FCA-recognised needs the FCA register. |
| 24 | **Open legal questions behind the ISA rule.** | Unresolved from primary sources (see [research](../research/etf-eligibility-isa-ira-401k.md#open-questions)): whether US ETFs count as "shares issued by a company" under reg 7(2)(a); whether an unrecognised UCITS ETF can qualify through its listing after 31 Dec 2026. The rule stays at `conditional` until these are settled. |
| 25 | **Exchange codes don't carry the market segment.** | LSE, German and Euronext codes can't tell Main Market from AIM, or Regulierter Markt from Freiverkehr, so those listings are `conditional` for ISAs. |
| 26 | **Date-bound ISA rules.** | Transitional UCITS cover ends 31 Dec 2026, and the UK retail disclosure transition ends 7 June 2027. Rule version `isa_uk_v1.0_2026` must be reviewed by then. |
