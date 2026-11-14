# Data sources

All data comes from public sources at request time. Each fund carries `dataSources` (when `includeSourceLinks` is set), so you can see where its data came from.

## Status (checked 2026-11-15)

| Source | Endpoint | Used for | Status |
|---|---|---|---|
| Yahoo Finance chart | `query1.finance.yahoo.com/v8/finance/chart/{ticker}?range=3mo&interval=1d` | Name, currency, exchange, 3-month average daily volume | **Works** without auth |
| Yahoo Finance quote | `…/v7/finance/quote` | Formerly price, market cap, volume | **Blocked**: 429/401 without a session crumb. No longer called. |
| Yahoo Finance quoteSummary | `…/v10/finance/quoteSummary/{ticker}?modules=fundProfile,topHoldings,fundPerformance` | Provider (fund family), TER, holdings, sectors | **Blocked**: 401. Still called; the failure is treated as missing data. |
| etf.com screener | `www.etf.com/api/screener?q=…` | Global fund screening (non-ZA requests) | **Blocked**: 403. Errors are ignored. |
| Alpha Vantage | `alphavantage.co` `ETF_PROFILE`, `GLOBAL_QUOTE` | Profiles and quotes when `ALPHA_VANTAGE_API_KEY` is set (not `demo`) | **Not checked**: no key was available. Free-tier keys: <https://www.alphavantage.co/support/#api-key> |

What this means in practice:
- Funds come back with **no AUM, TER, provider, holdings, sectors or regions**.
- Constraints on AUM and TER can't be checked; the affected funds are flagged.
- The TFSA rule can't confirm the provider, so ZA TFSA results are `unknown`.

## Where tickers come from

There is no live "list all ETFs" source. Tickers are chosen from hardcoded lists in `server/internal/service/search/provider.go`:

- **JSE** (`getJSETickersForCriteria`), used for country `ZA` or markets mentioning South Africa: STX40, STXNDQ, STX500, STXWDM, STXEMG, STXEUR, STXRES, COREEM, picked by market and asset class. They're looked up on Yahoo as `{ticker}.JO`, and get a second `ExchangeListing` data source pointing at jse.co.za. **No JSE API or website is called.**
- **Global** (`getETFTickersForCriteria`): technology → QQQ, XLK, VGT, SOXX; usa/us → SPY, VOO, IVV, VTI; emerging → EEM, VWO, IEMG; china → FXI, MCHI, ASHR; india → INDA, EPI, INDY; healthcare → XLV, VHT, IHI. When nothing matches, it falls back to popular funds by asset class.

Market and sector names in these lists are matched ignoring case, but otherwise exactly. So `usa` and `US` pick the US list, while `United States` doesn't.

## Normalisation

- Yahoo reports JSE prices in cents (`ZAc`) and LSE prices in pence (`GBp`). Currencies are mapped to `ZAR` / `GBP`.
- Funds from several providers are merged by ticker/ISIN; see [architecture](../architecture/README.md#search-providers).
- Results are cached in memory for one hour per set of search criteria. `cacheHit: true` in the response means no source was contacted.

## Checking a source yourself

```bash
# Works: basic quote data
curl -s -A "Mozilla/5.0" "https://query1.finance.yahoo.com/v8/finance/chart/STX40.JO?range=5d&interval=1d" | jq '.chart.result[0].meta | {symbol, longName, currency, exchangeName}'

# Blocked: fund details (prints the HTTP status)
curl -s -o /dev/null -w "%{http_code}\n" -A "Mozilla/5.0" "https://query1.finance.yahoo.com/v10/finance/quoteSummary/STX40.JO?modules=fundProfile"
```

The live test tier (`go test -tags live ./...`, see [development](../development/README.md#tests)) checks that the live sources still return usable, attributed data.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `results` empty for a ZA TFSA search, warning `NO_ELIGIBLE_RESULTS` | Eligibility is `unknown` because the provider is missing. Use `tfsaEligibleOnly: false` + `includeAlternatives: true` to see candidates. |
| `UNVERIFIED_AUM` / `UNVERIFIED_TER` warnings | Expected while quoteSummary is blocked |
| Server log shows `yahoo chart …: HTTP 429` | Yahoo rate limiting; retry later. The 1-hour cache cuts repeat calls. |
| Second identical request is instant, with `cacheHit: true` | Served from the cache |
