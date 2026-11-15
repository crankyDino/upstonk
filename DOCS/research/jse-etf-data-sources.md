# JSE ETF and ETN data sources

Research notes for replacing the hardcoded JSE ticker list (`server/internal/service/search/provider.go`, see [data sources](../data-sources/README.md)) with a complete, attributed, "never guess" data set for every ETF and ETN listed on the Johannesburg Stock Exchange.

All sources were accessed on **2026-11-15**, and quotations come from the versions published that day. Where a point could not be confirmed from a primary source, these notes say so. These are research notes, not legal advice.

## Summary

- **There is no free source that is clearly permitted and covers everything.** Every candidate fails on at least one of: completeness, a machine-readable format, or terms that allow automated use.
- **Universe list.** The JSE publishes an "ETF List" and an "ETN List" as downloads on its exchange-traded products page. **I could not open them.** The JSE's `robots.txt` blocks AI agents by name (`User-agent: Claude-User` / `Disallow: /`), so I did not fetch any jse.co.za page beyond `robots.txt`. The file format, fields and update cadence are **unverified**; a person needs to download one and check (see Open questions). Secondary reports put the count at about **131 ETFs** (February 2026) and **291 ETPs** in total (ETFs + ETNs + AMCs, end of 2025). Both figures are unverified.
- **Issuers.** I confirmed these as current JSE ETF/ETN issuers: **Satrix** (35 ETFs), **10X** (which now runs the former CoreShares range), **Absa** (NewGold ETFs and NewWave ETNs), **Prescient** (AMETF platform), **FNB** (ETFs managed by Ashburton), **Sygnia**, **1nvest**, and **EasyETFs** (formerly Cloud Atlas/AMI). **Etica and Global X could not be found** as JSE issuers. I found no evidence of a JSE-listed Global X fund. CoreShares SA's own domain no longer resolves, and coreshares.com is now Leverage Shares (UK).
- **Fund facts.** Issuer MDDs (minimum disclosure documents) are **PDFs**, usually monthly with a lag of about 4 to 6 weeks; some are quarterly. They provide most of the required fields, but each issuer lays them out differently. Two issuers expose JSON behind their sites (**Absa AISS** and **10X**). **Most issuers' terms forbid automated copying without written consent** (Satrix, 10X/CoreShares, Absa, Yahoo). FNB's `robots.txt` disallows its MDD PDFs.
- **Prices and volume.** Yahoo v8 chart resolved **18 of 26** sampled codes, including every live code from every issuer sampled except FNB/Ashburton (whose current codes are unconfirmed). However, `query1.finance.yahoo.com/robots.txt` is `User-agent: *` / `Disallow: /`, and Yahoo's Terms forbid automated collection. **The service's current Yahoo usage is therefore not within terms.** Issuers publish a free daily **NAV** (Satrix, 10X, Absa), but **no free permitted source for daily traded volume** was found.
- **TFSA.** There is **no official list** of TFSA-eligible ETFs from National Treasury, SARS or ASISA. Eligibility is a property of the *product provider's account* under s12T and GN R.172 of 2015, as amended. Some issuers state it in the MDD; 10X does, but Satrix, Prescient and Absa MDDs do not.

---

## 1. The authoritative list of JSE-listed ETFs and ETNs

### 1.1 Where the JSE publishes it

| Item | Finding | Confidence |
|---|---|---|
| Page | `https://www.jse.co.za/trade/equity-market/exchange-traded-products`; ETN page `https://www.jse.co.za/exchange-traded-notes` | From search-engine results only; not opened |
| Files | "Related Documents": "ETF List v.87", "AMC List v.31", "ETN List" (search snippet). An older derivatives-market copy shows "ETF List v.20" / "ETN List v.05" | Unverified |
| Format | Unknown (the version suffix suggests a periodically replaced spreadsheet or PDF) | **Unverified** |
| Fields | Unknown. A separate search snippet suggests `jse.co.za/etf` lists "alpha code, long name, tracked index". Monthly "JSE ETF Factsheet" PDFs exist (e.g. `…/jse-etf-factsheet-june-2025/June%202025.pdf`) with new listings/delistings by code and issuer | **Unverified** |
| Update frequency | Unknown; the list is versioned rather than dated | **Unverified** |

**Why this is unverified:** jse.co.za's `robots.txt` (fetched 2026-11-15) excludes AI agents by name. I am one, so I respected it:

```
User-agent: *
Content-Signal: search=yes,ai-train=no,use=reference
Allow: /
...
User-agent: Claude-User
Disallow: /

User-agent: ChatGPT-User
Disallow: /
```

The file's preamble also says: "As a condition of accessing this website, you agree to abide by the following content signals" and "ANY RESTRICTIONS EXPRESSED VIA CONTENT SIGNALS ARE EXPRESS RESERVATIONS OF RIGHTS UNDER ARTICLE 4 OF THE EUROPEAN UNION DIRECTIVE 2019/790".

For the upstonk server itself (not an AI agent), the `User-agent: *` group is `Allow: /`. So `robots.txt` would not block a polite monthly download of the ETF list. The JSE's **website terms of use** were not reviewed and must be checked before relying on that.

### 1.2 Official JSE data products

- The JSE sells market data under a Data Agreement. The "JSE Market Data Policies v5.0" PDF (`jse.co.za/sites/default/files/media/documents/jse-market-data-policies/JSE%20Market%20Data%20Policies%20v5.0.pdf`) and an "Indices Data Agreement" appear in search results. I did not open them, for the robots reason above.
- **I could not confirm any free JSE end-of-day price feed.** Treat JSE EOD prices and volumes as a licensed product until someone reads the Market Data Policies.

### 1.3 Fallback for the universe: union of issuer lists

These issuer sources were read and can enumerate each issuer's range:

| Issuer | Enumerable list | Count seen |
|---|---|---|
| Satrix | `https://satrix.co.za/api/products/search/filter?fundType=ETF&categoryId=1` (HTML fragment) and `https://satrix.co.za/daily-fund-prices?fundtype=ETF` (HTML table) | 35 ETFs in the price table |
| 10X | `https://assetmanagement.10x.co.za/funds/?view=list` | ETFs, AMETFs and unit trusts mixed together |
| Absa | `https://aiss.absa.africa/api/client/products/` (JSON) | 21 listings across 8 exchanges; JSE: GLD, NGPLT, NGPLD, NEWPLT, NEWUSD, NEWGBP, NEWEUR, NEWSLV, plus AMCs ABAM1/ABAM2 |
| Prescient | `https://www.prescient.co.za/funds/prescient-ametfs/` | 4 Prescient-branded AMETFs. Third-party co-named AMETFs (e.g. etfSA Oyster, Vunani) are separate |
| FNB | `https://www.fnb.co.za/share-investing/exchange-traded-funds.html` | 8 MDD links |

A union of issuer lists can never prove completeness. Only the JSE list can do that.

---

## 2. Issuers

### 2.1 Per-issuer table

| Issuer (JSE brand) | Factsheet/MDD location and URL pattern | Format | Fields seen in a sampled MDD | Update cadence | API/JSON behind site | robots.txt (verbatim, relevant lines) | Terms on automated access |
|---|---|---|---|---|---|---|---|
| **Satrix** | `https://satrix.co.za/fund/mdd/{JSE_CODE}` redirects to `/media/{id}?inline=true` | PDF (2 pages) | As-of date, JSE code, ISIN, benchmark, inception, ASISA class, distribution frequency, portfolio value (AUM), NAV, TER/TC/TIC (1y and 3y), top 10 holdings, sector allocation, "direct replication", "registered as a Collective Investment Scheme", issue date. **No TFSA wording.** | "monthly Minimum Disclosure Document". 30 Sep 2026 MDD issued 26 Oct 2026 | `/api/products/search/filter` returns HTML, not JSON. Bare `/api/products` returns an auth error. Daily NAV and TER are in a server-rendered HTML table (`lastUpdatedDate = '12/11/2026'`) | **None**: `/robots.txt` returns 404 | Terms PDF (`/media/129`): "Content and design of the Website are protected by copyright. Any copying of the Website or of its content requires the preliminary written consent of Satrix Managers (Pty) Ltd." **Consent required.** |
| **10X** (includes former **CoreShares** SA range: CSP500, CTOP50, CSPROP, CSGOVI, GLODIV, GLPROP, WNXT40, WTOP20 …) | `https://assetmanagement.10x.co.za/wp-content/uploads/{YYYY}/{MM}/{Fund-Name}-{Month}-{YYYY}-MDD-Class-A.pdf`, linked from `/fund/{slug}/`. The name slug is not predictable from the code | PDF | As-of date, JSE code, benchmark, ASISA class, fees, TER/TC/TIC, fund size, launch dates, distribution frequency, market maker, NAV, structure (for example "Feeder Fund … consist solely of participatory interests in the Vanguard S&P 500 UCITS ETF"), **explicit TFSA statement** | Monthly (Sep 2026 MDD uploaded Oct 2026) | **Yes.** `POST /wp-admin/admin-ajax.php` with `action=get_fund_details_with_ajax&post_id={id}` returns JSON `{"etf":{"nav":"R 127.48","fundSize":"R 3 347 768 329.95","indexValue":…,"distributableAmount":…},"shares":…}`. Holdings: `action=cs_get_daily_holdings_content` (HTML, daily and month-end) | `assetmanagement.10x.co.za`: `User-agent: *` / `Disallow:` (allows all). `www.10x.co.za`: `User-agent: *`, `Disallow: https://www.10x.co.za/compliance/*` …, `Allow: /` | ToU (`/wp-content/uploads/2021/12/CSH-Terms-of-Use.pdf`, still in CoreShares' name) 2.10: "A part from bona-fide search engine operators … no person may use or attempt to use any technology or applications (including web crawlers, robots or web spiders) to search, collect or copy content from the Website for any purpose whatsoever, without the prior written consent from CoreShares." 2.2: "Content from the Website may not be used or exploited by Users for any commercial and non-private purposes without the prior written consent". **Consent required.** |
| **Absa** (NewGold ETFs, NewWave ETNs; Absa Index & Structured Solutions) | Listing: `https://aiss.absa.africa/api/client/downloads/` (JSON, 2,610 docs). File: `https://aiss.absa.africa/api/client/downloads/file/{downloadFileId}` | JSON list + PDF | GLD MDD (30 Sep 2026): manager, issuer, market maker, exchange, code, ISIN, benchmark, listing date, securities issued, NAV (ZAR), NAV per share, **"Annual Sales Charge 0.30%"** (not labelled TER), ounces held, returns, "structured as a debenture", "fully backed by holdings of gold bullion", Shariah statement. No TFSA wording | GLD data sheets appear quarterly (Jun 2026, Sep 2026). Sep data sheets published 21 Oct 2026 | **Yes.** `GET /api/client/products/` returns JSON per listing: `isin`, `name`, `type` (ETF/ETN/AMC), `exchangeCode`, `stockExchange`, `issuer`, `manager`, `launchDate`, `benchmark`, about 250 daily `prices`, `totalExpenseRatio`, `marketCapitalisation`, `distributes`, `bloombergCode` | `aiss.absa.africa`: `User-agent: *` / `Allow: /` | AISS Disclaimer PDF: "no part of this document may be reproduced or transmitted in any form or by any means … without the prior permission in writing from Absa." **Consent required.** |
| **Prescient** (AMETF platform; also manages co-named AMETFs) | `https://www.prescient.co.za/media/{id}/{fund-slug}-{month}-{yyyy}-{code}.pdf`, linked from `/funds/prescient-ametfs/` and `/news-and-resources/minimum-disclosure-documents/` | PDF ("MINIMUM DISCLOSURE DOCUMENT & GENERAL INVESTOR REPORT") | PIPETF (30 Sep 2026): JSE code, ISIN, benchmark, fund size, inception date, TER, distributions, risk profile, Regulation 28 status. No TFSA wording | Every MDD on the index page on 2026-11-15 was "September 2026", so the cadence looks quarterly (unconfirmed) | Umbraco site search only (`/umbraco/api/searchapi/SiteSearch`, under a disallowed path). No fund-data JSON found | `User-agent: *` / `Disallow: /umbraco/` | `/legal-information/` page rendered no terms text; **terms not located** |
| **FNB** (ETFs managed by Ashburton: FNB Top 40, Midcap, Inflation, World Government Bond, Global 1200 FoF, MSCI World Feeder, MSCI EM Feeder, S&P …) | `https://www.fnb.co.za/downloads/unittrust/exchange-traded-funds/MDD/{Name}.pdf` (fixed URL per fund, e.g. `FNBTop40ETF.pdf`) | PDF | **Not fetched** (robots) | Unknown | Ashburton's Next.js fund page (`ashburtoninvestments.com/funds-and-performance/passives-strategies/exchange-traded-fund/{ISIN}`) carries fund JSON with most fields "N/A" or `$undefined`. Live data likely comes from `/api/`, which robots disallows | `fnb.co.za`: `User-agent: *` … `Disallow: /*.pdf` / `Disallow: /downloads/`. `ashburtoninvestments.com`: `Disallow: /api/` / `Disallow: /inward-listed-etns` | Not reviewed. **robots.txt already disallows the MDD PDFs and the Ashburton API.** |
| **Sygnia** (Itrix) | `https://www.sygnia.co.za/wp-content/uploads/{YYYY}/{MM}/{YYYY}-{MON}-{CODE}-{Fund-Name}-ETF-Fund-Fact-Sheet.pdf` (from search results; one guessed URL returned 404) | PDF | Not read | Monthly, judging by filenames (e.g. 2026-APR, 2026-JUL) | Not inspected: the site serves an **Imperva/Incapsula JavaScript challenge** to non-browser clients (`/_Incapsula_Resource`) | `Crawl-delay: 10` (placed **before** any `User-agent` line, so it belongs to no group), then `User-agent: *` / `Disallow:` | Not reviewed (blocked by bot challenge) |
| **1nvest** (Standard Bank) | Unknown on the issuer site. Copies exist on resources.easyequities.co.za | PDF (third-party copy) | Not read | Unknown | Not inspected: **Cloudflare challenge, HTTP 403**, for curl and WebFetch alike | `User-agent: *` / `Allow: /` | Not reviewed (blocked) |
| **EasyETFs** (formerly Cloud Atlas / AMI) | `https://cloudatlas.easyequities.co.za/hubfs/CloudAtlas/SENS/AMIB50/Big-50-ex-SA-ETF-factsheet.pdf` | PDF | "As at end of February 2023". AMIB50 page says "STATUS: Suspended", ISIN ZAE000242582. SENS links include "AMIB50 suspension - 23 March 2023" and "AMIB50 wind up - 5 February 2024" | Stale (Feb 2023) | HubSpot CMS, no data API seen | `User-agent: *` / `Disallow: /_hcms/preview/` … (allows fund pages) | Not reviewed |
| **Etica** | **Not found.** No JSE-listed Etica ETF surfaced in searches | n/a | n/a | n/a | n/a | n/a | n/a |
| **Global X** | **Not found.** Search returned only Canadian, Australian, Hong Kong and EU Global X sites. No JSE SENS for a Global X listing | n/a | n/a | n/a | n/a | n/a | n/a |
| **CoreShares** | `coreshares.co.za` **does not resolve** (DNS). `www.coreshares.com` is run by "Investium Limited and Leverage Shares Management Company Limited" (London) and lists Leverage Shares ETCs (e.g. CULS, ISIN XS2702787875), not JSE ETFs. The SA CoreShares range now appears under 10X (above) | n/a | n/a | n/a | coreshares.com has Inertia JSON with `holdings.csvUrl`, but for the wrong products | coreshares.com: comment-only file with no `User-agent` rules | coreshares.com ToU: "No information contained on this site may be reproduced … or otherwise used for commercial purposes without the prior consent of Leverage Shares." |

**Other issuers, not reviewed.** These appear in JSE listing news (via search results) and should appear in the JSE list: 27four (Global Shariah AMETF, listed 8 Oct 2025), Amplify (AMETF, Feb 2026), Benguela/EasyETFs (BGFMAI, Aug 2026), Ninety One (two ETFs, Nov 2025), Cartesian (Balanced ETF, Jan 2026), etfSA-branded Prescient AMETFs, and STANLIB (appears in the ETF500 long name "1NVEST S&P 500 Index STANLIB Feeder ETF"). **The full issuer count can only be settled from the JSE list.** On what I saw, it is roughly 15 to 20 brands.

**Absa NewFunds (NFEMOM, NFEDEF, NFTRCI…).** These are absent from the Absa AISS product API and from Yahoo (404). They may have been delisted or transferred, but **I could not confirm this.**

### 2.2 Data-quality traps found ("never guess")

- **Absa API zeros are placeholders, not values.** Every Absa product returns `"totalExpenseRatio": 0` and `"marketCapitalisation": 0`. The GLD MDD shows an "Annual Sales Charge" of 0.30% and a Total NAV of R35.7 bn. Map `0` in these fields to **unknown**, never to 0%.
- **Labels differ.** Absa uses "Annual Sales Charge", not TER. Satrix reports TER, TC and TIC over 1 and 3 years. 10X reports a separate management fee, VAT, other costs, TER, TC and TIC. Store the issuer's own label, and only map it to `ter` where the label actually says TER.
- **Web page and MDD disagree.** For CSP500 on 2026-11-15, the 10X web page said "TER 0.38%" and "Market maker Sanlam Private Wealth (SPW)". The 30 Sep 2026 MDD says TER 0.37% and market maker "Jane Street". Keep the source and as-of date with every value, and prefer the dated MDD for fund facts.
- **Domicile is rarely stated outright.** Ashburton's fund JSON has `"Domicile","value":"N/A"`. A `ZAE…` ISIN shows where the security was issued, which is not necessarily the fund's domicile. Record domicile only where the document states it, or where it says "registered as a Collective Investment Scheme" under CISCA.
- **Cents.** Yahoo reports `ZAc`, and Satrix's NAV column is "NAV PER UNIT (C)". 10X and Absa give rands.

---

## 3. Daily price and volume

### 3.1 Yahoo Finance v8 chart coverage

Request: `https://query1.finance.yahoo.com/v8/finance/chart/{CODE}.JO?range=5d&interval=1d`, `User-Agent: Mozilla/5.0`. This was made as a one-off verification on 2026-11-15, 2 seconds apart, at the caller's explicit request.

| Code | Issuer (per primary source) | HTTP | Yahoo name | Last bar | Last volume |
|---|---|---|---|---|---|
| STX40 | Satrix | 200 | Satrix 40 ETF | 2026-11-13 | 123178 |
| STXNDQ | Satrix | 200 | Satrix Nasdaq 100 ETF | 2026-11-13 | 53360 |
| STXGLP | Satrix | 200 | Satrix Global Property Feeder ETF | 2026-11-13 | 2152 |
| STXWIS | Satrix | 200 | Satrix MSCI World Islamic Feeder ETF | 2026-11-13 | 19204 |
| CSP500 | 10X | 200 | 10X S&P 500 ETF | 2026-11-13 | 2453 |
| CTOP50 | 10X | 200 | 10X S&P SA Top 50 ETF | 2026-11-13 | 33521 |
| GLODIV | 10X | 200 | 10X S&P Global Dividend Aristocrats ETF | 2026-11-13 | 141824 |
| WNXT40 | 10X | 200 | 10X Wealth Next40 Equal Weighted ETF | 2026-11-13 | 89923 |
| PIPETF | Prescient | 200 | Prescient Income Provider Feeder Actively Managed ETF (A) | 2026-11-13 | 186184 |
| GLD | Absa (NewGold) | 200 | NewGold Issuer Limited | 2026-11-13 | 454217 |
| NGPLT | Absa (NewGold) | 200 | NewGold Platinum ETF | 2026-11-13 | 106 |
| NEWUSD | Absa (ETN) | 200 | NewWave USD Currency Exchange Traded Note | 2026-11-13 | 905488 |
| SYG500 | Sygnia | 200 | Sygnia Itrix S&P 500 ETF | 2026-11-13 | 21253 |
| SYGWD | Sygnia | 200 | Sygnia Itrix MSCI World ETF | 2026-11-13 | 344889 |
| SYGT40 | Sygnia | 200 | Sygnia Itrix Top 40 ETF | 2026-11-13 | 1617 |
| ETFGLD | 1nvest | 200 | 1nvest Gold ETF | 2026-11-13 | 1677 |
| ETF500 | 1nvest | 200 | 1NVEST S&P 500 Index STANLIB Feeder ETF | 2026-11-13 | 625 |
| BGFMAI | Benguela EasyETFs | 200 | Benguela EasyETFs Global AI Revolution Actively Managed ETF | 2026-11-13 | 345 |
| AMIB50 | EasyETFs (suspended) | 404 | "No data found, symbol may be delisted" | – | – |
| NFEMOM | Absa NewFunds (status unconfirmed) | 404 | same | – | – |
| NFEDEF | Absa NewFunds (status unconfirmed) | 404 | same | – | – |
| NFTRCI | Absa NewFunds (status unconfirmed) | 404 | same | – | – |
| ASHT40 | Ashburton legacy code (unconfirmed) | 404 | same | – | – |
| ASHMID | Ashburton legacy code (unconfirmed) | 404 | same | – | – |
| ASHINF | Ashburton legacy code (unconfirmed) | 404 | same | – | – |
| ASHWGB | Ashburton legacy code (unconfirmed) | 404 | same | – | – |

18 of 26 resolved, and every code confirmed live by an issuer document resolved. The 8 failures are a suspended fund and codes whose current status is unconfirmed. **FNB's current JSE codes were not confirmed**, so FNB coverage is untested.

### 3.2 Yahoo's terms

- `https://query1.finance.yahoo.com/robots.txt`, verbatim and complete:

  ```
  User-agent: *
  Disallow: /
  ```

- `https://finance.yahoo.com/robots.txt` names AI agents (`User-agent: ChatGPT-User`, `User-agent: ClaudeBot`, `User-agent: anthropic-ai` … `Disallow: /`). For `*` it disallows `/quote/*` subpaths such as `Disallow: /quotes/*` and `Disallow: /lookup/`.
- Yahoo Terms of Service (`https://legal.yahoo.com/us/en/yahoo/terms/otos/index.html`) say you may not "access or collect data, or attempt to access or collect data, from our Services using any automated means, devices, programs, algorithms or methodologies, including but not limited to robots, spiders, scrapers, data mining tools, or data gathering or extraction tools, for any purpose without our express, prior permission". Nor may you "use any material or content from, including without limitation any data, (a) to create any database, archive, mobile application, data feed, widget or any other aggregated data source that competes with or constitutes a material substitute for the Services".

**Conclusion:** the service's current use of the Yahoo chart endpoint is not allowed by Yahoo's `robots.txt` or Terms. It works technically; it is not permitted.

### 3.3 Other free end-of-day sources

| Source | What it gives | Access | Terms/robots |
|---|---|---|---|
| Satrix daily fund prices (`/daily-fund-prices?fundtype=ETF`) | Daily **NAV** (cents) + rolling 1-year TER for 35 Satrix ETFs. No traded price or volume | HTML table (and a client-side XLSX export) | No `robots.txt`. Terms require written consent for "any copying" |
| 10X fund AJAX | Daily NAV, fund size, shares in issue | JSON (admin-ajax) | robots allows. ToU 2.10 bans crawlers without consent |
| Absa AISS products API | About 250 daily prices per listing (NAV-type price) | JSON | robots `Allow: /`. Disclaimer bans reproduction without written permission |
| Sharenet | Not reviewed. Its terms page is itself disallowed in robots (`User-agent: *` … `Disallow: /v3/terms.php`, `Disallow: /jsepdf/`) | – | Terms not read; robots disallows the terms page |
| etfSA (`etfsa.co.za`) | Aggregator; same Cloudflare AI block as the JSE (`User-agent: Claude-User` / `Disallow: /`) | – | Not fetched |
| JSE | EOD data is sold under the Market Data Policies (not read) | – | See §1 |

**No free source permitted for automated use was found for daily traded volume.**

---

## 4. TFSA eligibility in South Africa

### 4.1 Is there an official list?

**No.** I found no list from National Treasury, SARS or ASISA. SARS's page (`https://www.sars.gov.za/individuals/tax-free-investments/`) says: "Tax Free Investments are approved account types and investment vehicles offered by authorised financial service providers and recognised by SARS." It names provider types, not funds. It also states the current limit: "an annual limit (R46 000 with effect from 1 March 2026) as well as a lifetime limit (R500 000)". The etfSA "Tax Free – qualifying ETFs" PDF (v11, Mar 2024) is an aggregator document and is dated.

### 4.2 What the law requires

- **Income Tax Act s12T** creates the exemption. Regulations under **s12T(8)** set the requirements. I did not read the consolidated text of s12T from an official source in this session; see Open questions.
- **GN R.172 of 25 February 2015, Government Gazette 38509** (`https://www.treasury.gov.za/Legislation/regulations/38509_25-2_NationalTreasCV01.pdf`), read in full:
  - Reg 2: an instrument "may only be issued by" (a) a bank … (c) "a manager as defined in section 1 of the Collective Investment Schemes Control Act, 2002 … other than a manager of a collective investment scheme in participation bonds" …
  - Reg 3(1): "A product provider must advertise a tax free investment as being a tax free investment by virtue of section 12T of the Income Tax Act". Reg 3(2): "the designation … must contain the words 'tax free'."
  - Reg 13 sets composition limits: "not more than 10 per cent … from shares in any single company"; "not less than 80 per cent of any shares must be listed on a recognised exchange"; for an index, a single company may not exceed "120 per cent of that company's weighting in that index, subject to a maximum of 35 per cent". Reg 13(3): "not more than 10 per cent of the value of that tax free investment may be derived from that commodity". **Reg 13(6): "This regulation does not apply in respect of any financial instrument in respect of a collective investment scheme."**
  - Reg 14: "A derivative instrument may be utilised … only to the extent that the derivative is utilised for the purposes of reducing the risk of loss or reducing cost (without any increase in risk of loss)". This is relevant to leveraged and inverse products.
  - Reg 17: "Any financial instrument or policy that does not comply with these regulations is not a tax free investment for the purposes of section 12T".
- **Amendments.** GN 54 of 18 February 2016 amended reg 9 (`treasury.gov.za/legislation/GovernmentNotices/54-16 …pdf`, not opened). A 2017 amendment followed, which secondary sources call GG 40758 of 31 March 2017 (number unconfirmed). National Treasury's 28 Feb 2017 media statement (read) says: "Performance fees are not permitted in tax free savings accounts whether charged as part of the tax free savings account or in an underlying fund into which tax free savings account contributions are invested."

**Implications for the rules engine (my reading; not legal advice).** A CISCA-registered ETF escapes the reg 13 composition limits (13(6)). It must still have no performance fee (2017 amendment) and must use derivatives only to reduce risk or cost (reg 14). Whether a given ETF is TFSA-eligible *in practice* depends on the investor's TFSA **account provider** offering it. ETNs and debenture-structured products such as NewGold ("structured as a debenture") are not CIS participatory interests, so the reg 13 limits would apply to them, including the 10% single-commodity limit. **I have not confirmed how providers treat these in practice.**

### 4.3 Where issuers say it

| Issuer | Statement found |
|---|---|
| 10X | MDD: "Participatory interests in the portfolios issued by the Manager qualify as investment instruments for the purposes of Tax-Free Savings and Investment Account by virtue of Section 12T of the Income Tax Act (available on certain LISPs)." Manager-wide, not per fund |
| Satrix | Not in the STX40 MDD. Has a general "Tax Free Investing" page with no per-fund flag found |
| Prescient | Not in the PIPETF MDD |
| Absa | Not in the GLD MDD. Absa Bank's own "Tax-Free ETF Savings Account" page (seen in search only) offers "select" ETFs |
| Others | Not checked |

---

## 5. Recommendation

### 5.1 Field-by-field

| field | primary source | access method (API/download/scrape) | allowed by terms/robots? | refresh | fallback |
|---|---|---|---|---|---|
| Universe (code, name, ISIN, issuer, board) | JSE "ETF List" / "ETN List" (exchange-traded-products page) | Download | robots: yes for a non-AI client (`User-agent: *` / `Allow: /`). **Terms: unverified** | Per JSE version bump (cadence unknown) | Union of issuer lists (§1.3), marked "possibly incomplete" |
| Daily price | Yahoo v8 chart | Undocumented API | **No** (query1 `Disallow: /`; ToS bans automated collection) | Daily | Issuer daily **NAV** (Satrix HTML, 10X JSON, Absa JSON). All three need written consent under their terms. Label it as NAV, not price |
| Daily volume | Yahoo v8 chart | Undocumented API | **No** | Daily | **None free and permitted.** Licence the JSE EOD feed or ask Yahoo for permission |
| TER / TIC | Issuer MDD | PDF download + parse | robots: yes for Satrix, 10X, Absa, Prescient; **no for FNB** (`Disallow: /*.pdf`). Terms: consent needed for Satrix, 10X, Absa | Monthly (Satrix, 10X); quarterly (Absa, likely Prescient) | Satrix daily table (rolling 1-yr TER). Never Absa API `totalExpenseRatio` (placeholder 0) |
| Fund size (AUM) | Issuer MDD | PDF | As above | Monthly/quarterly | 10X AJAX `fundSize` (daily) |
| Top holdings | Issuer MDD (Satrix top 10) | PDF | As above | Monthly | 10X daily holdings AJAX (HTML) |
| Sector exposure | Issuer MDD (Satrix "Portfolio allocation") | PDF | As above | Monthly | none |
| Region/country exposure | Issuer MDD where present (not in sampled Satrix STX40, a domestic fund) | PDF | As above | Monthly | **none confirmed** |
| Tracked index | MDD "Benchmark" | PDF | As above | Monthly | Absa API `benchmark.name`; Satrix product cards |
| Domicile | MDD statement of CISCA registration | PDF | As above | Rarely changes | **none**; ISIN prefix is not proof |
| Legal structure | MDD text ("registered as a Collective Investment Scheme", "structured as a debenture", ETN) | PDF | As above | Rarely changes | Absa API `type` (ETF/ETN/AMC) |
| Replication (physical/synthetic) | MDD text ("direct replication", "fully backed by … bullion", "Feeder Fund") | PDF, free text | As above | Rarely changes | none; free text needs per-issuer parsing, otherwise unknown |
| Leverage / inverse | MDD text | PDF, free text | As above | Rarely changes | **none**; no sampled source has a flag |
| Issuer / manager | MDD | PDF | As above | Rarely changes | Absa API `issuer`/`manager` |
| ISIN | MDD | PDF | As above | Static | Absa API `isin`; Ashburton page URL contains the ISIN |
| Distribution policy | MDD ("Distribution Frequency") | PDF | As above | Rarely changes | Absa API `distributes`, `distributionFrequency` |
| Inception date | MDD | PDF | As above | Static | Absa API `launchDate` (per listing, not per fund) |
| As-of date | MDD header ("30 September 2026") + issue date | PDF | As above | Per MDD | none; without it, the value is unknown |
| TFSA eligibility | Issuer MDD statement (10X); otherwise none | PDF | As above | Per MDD | **No official list.** Default `unknown` |

### 5.2 Fields no free, permitted source provides

1. **Daily traded volume**, for any issuer.
2. **Daily market (traded) price.** Only NAV is available free, and even that needs issuer consent.
3. **A verified complete universe.** Pending a human check of the JSE list download and the JSE terms.
4. **Explicit leverage/inverse flag** and **explicit domicile**. These exist only as free text, when they exist at all.
5. **Regional/country exposure** for most funds (seen in no sampled MDD).
6. **An official TFSA list.**
7. **Anything for FNB ETFs**, because robots disallows the MDD PDFs and the Ashburton API.
8. **Machine-accessible data for Sygnia and 1nvest** (bot challenges).

### 5.3 Practical path

- **Ask issuers for written consent or a data feed.** Satrix, 10X, Absa and Sygnia all state that copying needs written consent. Several already have structured endpoints (Absa JSON, 10X AJAX), so asking costs little.
- **Price and volume:** budget for a licensed JSE EOD feed, or written permission from a vendor. Until then, keep Yahoo-derived values flagged and attributed, and decide (see Open questions) whether to keep calling it at all.
- **Keep every value as `{value, source URL, as-of date, label as printed}`.** Map placeholders (Absa `0`) and "N/A" to unknown.

---

## Open questions

1. **JSE ETF/ETN List:** what format is it, which columns does it have (ISIN? issuer? board/sector?), how often is it versioned, and is the URL stable? A person (not an AI agent) needs to download it from `jse.co.za/trade/equity-market/exchange-traded-products`.
2. **JSE website terms of use and Market Data Policies v5.0:** do they allow a monthly automated download of the ETF list, and is any EOD price or volume free?
3. **Should the service keep calling Yahoo**, given `query1` `robots.txt` `Disallow: /` and the ToS? If yes, under what written permission?
4. **FNB ETF current JSE codes**, and whether FNB will grant access to its MDD PDFs (`Disallow: /*.pdf`).
5. **Absa NewFunds (NFEMOM, NFEDEF, NFTRCI):** delisted, transferred, or renamed? Check the JSE list or SENS.
6. **Etica and Global X:** are any of their funds on the JSE? Nothing was found.
7. **Sygnia and 1nvest terms of use** (not readable because of bot challenges). Would they supply a feed?
8. **Prescient MDD cadence** (quarterly?) and Prescient's website terms (not located).
9. **Section 12T consolidated text** from an official source (SARS's consolidated Income Tax Act), and the **2017 amendment** gazette number and text (reported as GG 40758, unconfirmed). Do the amended regs change reg 13 or 14?
10. **TFSA treatment of ETNs and debenture ETFs** (e.g. NewGold) under reg 13(3)'s 10% single-commodity limit, as applied by account providers in practice.
11. **Satrix's 47 MDD codes against 35 ETFs:** some codes are unit-trust classes (e.g. SABIF, SATTI). The ETF/UT split needs to come from the price table, not the MDD link list.

---

## Sources

All accessed 2026-11-15.

**JSE** (robots.txt only fetched; other URLs from search results, not opened)
- <https://www.jse.co.za/robots.txt>
- <https://www.jse.co.za/trade/equity-market/exchange-traded-products> (search result)
- <https://www.jse.co.za/exchange-traded-notes> (search result)
- <https://jse.co.za/sites/default/files/media/documents/jse-market-data-policies/JSE%20Market%20Data%20Policies%20v5.0.pdf> (search result)

**Satrix**
- <https://satrix.co.za/robots.txt> (404)
- <https://satrix.co.za/api/products/search/filter?fundType=ETF&categoryId=1>
- <https://satrix.co.za/fund/mdd/STX40> (Satrix 40 ETF MDD, 30 September 2026, issue date 26 Oct 2026)
- <https://satrix.co.za/daily-fund-prices?fundtype=ETF>
- <https://satrix.co.za/assets/products.js>, <https://satrix.co.za/assets/daily-fund.js>
- <https://satrix.co.za/media/129> (Terms & Conditions)
- <https://satrix.co.za/tax-free-investments>

**10X / former CoreShares**
- <https://assetmanagement.10x.co.za/robots.txt>, <https://www.10x.co.za/robots.txt>
- <https://assetmanagement.10x.co.za/funds/>, <https://assetmanagement.10x.co.za/funds/?view=list>
- <https://assetmanagement.10x.co.za/fund/10x-sp-500-etf/>
- <https://assetmanagement.10x.co.za/wp-content/uploads/2026/10/10X-SP-500-Index-Feeder-ETF-September-2026-MDD-Class-A.pdf>
- <https://assetmanagement.10x.co.za/wp-admin/admin-ajax.php> (`action=get_fund_details_with_ajax`)
- <https://assetmanagement.10x.co.za/wp-content/themes/10x/assets/dist/js/main.min.js>
- <https://assetmanagement.10x.co.za/wp-content/uploads/2021/12/CSH-Terms-of-Use.pdf>
- <https://www.coreshares.com/robots.txt>, <https://www.coreshares.com/core-products/explore>, <https://www.coreshares.com/terms-of-use> (Leverage Shares)

**Absa**
- <https://aiss.absa.africa/robots.txt>
- <https://aiss.absa.africa/api/client/products/>
- <https://aiss.absa.africa/api/client/downloads/>
- <https://aiss.absa.africa/api/client/downloads/file/01a15dd0-7367-7e45-a28b-dd889078f23d> (NewGold ETF MDD, 30 September 2026)
- <https://aiss.absa.africa/assets/files/Disclaimer.pdf>
- <https://www.absa.co.za/robots.txt>

**Prescient**
- <https://www.prescient.co.za/robots.txt>, <https://www.prescient.co.za/sitemap.xml>
- <https://www.prescient.co.za/funds/prescient-ametfs/>
- <https://www.prescient.co.za/news-and-resources/minimum-disclosure-documents/>
- <https://www.prescient.co.za/media/qvlho5xn/prescient-income-provider-feeder-ametf-a-september-2026-petfipa.pdf>
- <https://www.prescient.co.za/legal-information/>

**FNB / Ashburton**
- <https://www.fnb.co.za/robots.txt>, <https://www.fnb.co.za/share-investing/exchange-traded-funds.html>
- <https://www.ashburtoninvestments.com/robots.txt>, <https://www.ashburtoninvestments.com/sitemap.xml>
- <https://www.ashburtoninvestments.com/funds-and-performance/passives-strategies/exchange-traded-fund>

**Sygnia, 1nvest, EasyETFs/Cloud Atlas**
- <https://www.sygnia.co.za/robots.txt>, <https://www.sygnia.co.za/sitemap_index.xml>, <https://www.sygnia.co.za/fund-sitemap.xml> (fund pages: Incapsula challenge)
- <https://1nvest.co.za/robots.txt> (site: Cloudflare 403)
- <https://cloudatlas.easyequities.co.za/robots.txt>, <https://cloudatlas.easyequities.co.za/amib50>
- <https://cloudatlas.easyequities.co.za/hubfs/CloudAtlas/SENS/AMIB50/Big-50-ex-SA-ETF-factsheet.pdf>
- <https://cloudatlas.co.za/> (unrelated business; not the ETF issuer)

**Price sources**
- <https://query1.finance.yahoo.com/robots.txt>, <https://finance.yahoo.com/robots.txt>
- `https://query1.finance.yahoo.com/v8/finance/chart/{CODE}.JO?range=5d&interval=1d` (26 codes, §3.1)
- <https://legal.yahoo.com/us/en/yahoo/terms/otos/index.html>
- <https://www.sharenet.co.za/robots.txt>, <https://www.etfsa.co.za/robots.txt>

**TFSA**
- <https://www.treasury.gov.za/Legislation/regulations/38509_25-2_NationalTreasCV01.pdf> (GN R.172, GG 38509, 25 Feb 2015)
- <https://www.treasury.gov.za/comm_media/press/2017/2017022801%20-%20Media%20Statement%20Tax%20Free%20Savings%20Accounts.pdf>
- <https://www.sars.gov.za/individuals/tax-free-investments/>
- <https://www.treasury.gov.za/robots.txt>, <https://www.sars.gov.za/robots.txt>, <https://www.asisa.org.za/robots.txt>
- `https://www.treasury.gov.za/legislation/GovernmentNotices/54-16 Amendment to Regulations in terms of section 12T 18 February 2016.pdf` (search result, not opened)
