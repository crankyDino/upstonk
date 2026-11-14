# Ubiquitous Language

The words UPSTONK uses for its domain, and what each one means. When code, docs, UI copy and conversation disagree, use the term in **bold**, and don't use the aliases listed against it.

## Investor and account

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Investor** | The person a discovery is run for, described by an investor profile. | User, client, customer |
| **Investor profile** | The investor's country of residence, account type and currency (plus risk tolerance and time horizon, which have no effect yet). | Profile, investor details |
| **Investor country** | The country whose tax rules apply to the investor (ISO-3166 alpha-2: `ZA`, `US`, `GB`). | Country (on its own), residence |
| **Account type** | The kind of investment account the investor holds, e.g. `tfsa`, `isa`, `ira`, `roth_ira`, `401k`, `standard`. | Account, wrapper, account kind |
| **Supported pair** | A combination of investor country and account type that the server accepts. | Supported country, valid account |
| **Tax-advantaged account** | Any account type other than `standard`; holdings are restricted by law. | Tax-free account, registered account |
| **Standard account** | A taxable brokerage account with no legal restrictions on which ETFs it holds. | Normal account, taxable account |
| **TFSA** | South African Tax-Free Savings Account (Income Tax Act §12T). Never the Canadian account of the same name. | Tax-free account |
| **Account provider** | The broker, platform or plan administrator that runs the investor's account and decides which products it offers. | Provider (on its own), platform, broker |
| **Plan menu** | The fixed list of investments a 401(k) plan offers; because only it decides, 401(k) results are always `unknown`. | Fund list, plan options |
| **Recognised stock exchange** | A market HMRC designates under Income Tax Act 2007 s1005; listing on one is one route for an ISA-qualifying investment. | Approved exchange, regulated market |
| **UCITS** | An EU/UK-regulated retail fund structure; a UK or FCA-recognised UCITS qualifies for an ISA. | European fund |
| **Product summary / KID** | The retail disclosure document UK firms must have before selling a fund to a retail investor; US ETFs normally lack one. | Factsheet, KIID (older) |

## Funds and their data

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **ETF** | An exchange-traded fund; the only investment vehicle UPSTONK actually searches. | Fund (on its own), product, instrument |
| **Ticker** | An ETF's trading symbol without exchange suffix, e.g. `STX40` (Yahoo's `STX40.JO` is an internal lookup detail). | Symbol, code |
| **Issuer** | The company that manages an ETF, e.g. Satrix, CoreShares, Vanguard. | Provider, fund family, manager |
| **Exchange** | Where an ETF is listed, e.g. `JSE`, `NYSE`. | Market, venue, listing |
| **Domicile** | The country where an ETF is legally established. | Country, home |
| **TER** | Total expense ratio: the fund's annual cost as a percentage of assets. | Fees, expense ratio, cost |
| **AUM** | Assets under management: the total market value the fund holds. | Fund size, size, market cap |
| **Average daily volume** | The mean number of shares traded per day (over 3 months, from Yahoo's chart data). | Liquidity, volume, turnover |
| **Unknown value** | An ETF attribute that no data source reported; stored as 0, empty or `nil` (for structure flags) and never treated as a real value. | Missing, default, zero, false |
| **Data source** | An external service UPSTONK reads fund data from, e.g. Yahoo Finance, etf.com, Alpha Vantage. | Provider, feed, API |
| **Provenance** | The list of data sources (with URL and date) recorded against each ETF. | Sources, attribution, source links |

## Discovery request

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Discovery** | One run of the pipeline: an investor profile, exposure and constraints in, ranked ETFs out. | Search, query, lookup |
| **Exposure** | What the investor wants to own: markets, sectors, asset classes and companies. | Criteria, targets, allocation |
| **Market** | A geographic area the investor wants exposure to, e.g. `usa`, `emerging markets`. | Region, geography, country |
| **Region** | A geographic bucket in an ETF's own breakdown, matched against the requested markets. | Market |
| **Sector** | An industry grouping, e.g. technology, healthcare. | Industry |
| **Asset class** | A broad type of holding, e.g. equity, bond, commodity. | Asset type, category |
| **Constraint** | A hard filter that removes ETFs, e.g. max TER, min AUM, minimum average daily volume, excluded structures. | Filter, criterion, restriction |
| **Ranking weights** | The share each ranking component contributes to the ranking score; must sum to 1.0 (the GUI's sliders are normalised to that). | Preferences, priorities, weighting |

## Eligibility

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Eligibility rule** | The codified legal test for one or more supported pairs, e.g. `TFSA_ZA`, `ISA_UK`, `IRA_US`, `401K_US`, identified by name and version. | Ruleset, policy, rules |
| **Criterion** | One check within an eligibility rule, e.g. "JSE-listed"; it passes, fails or is skipped. | Rule (on its own), check, condition |
| **Skipped criterion** | A criterion that couldn't be checked because an unknown value was involved. | Unknown rule, missing rule |
| **Eligibility status** | The rule's verdict: `eligible`, `conditional`, `ineligible` or `unknown`. | Result, outcome, eligibility (on its own) |
| **Eligible** | Every criterion passed. | Allowed, qualifying, approved |
| **Conditional** | Nothing failed, but some criteria were skipped or depend on the account provider; verify before investing. | Likely eligible, eligible with warnings |
| **Ineligible** | At least one criterion failed on data UPSTONK actually has. | Not allowed, rejected |
| **Unknown** (status) | Too many criteria skipped, or confidence too low, to make a call. | Undetermined, n/a |
| **Holdable** | Eligibility status of `eligible` or `conditional` (the API's `isEligible: true`). | Eligible (when conditional is included) |
| **Confidence** | How much of the verdict rests on verified data: `high`, `medium`, `low`, `none`. | Certainty, reliability |
| **Justification** | The human-readable reasons behind an eligibility status. | Explanation, reasoning |
| **Never guess** | The principle that missing data must lead to `unknown`, `conditional` or a flag, never to `eligible` or a silent default. | Fail-safe, conservative mode |
| **Provider-dependent constraint** | A restriction set by the account provider rather than by law (e.g. plan menus, broker product lists); it can only make a result conditional. | Broker rule, platform rule |

## Scoring and output

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **Match score** | 0–100: how well an ETF fits the requested exposure. | Relevance, fit, exposure score |
| **Ranking score** | 0–100: the ETF's quality from its ranking components and ranking weights. | Quality score, rating |
| **Combined score** | `0.4 × match score + 0.6 × ranking score`; decides the order of results. | Total score, final score |
| **Ranking component** | One input to the ranking score: `fees` (from TER), `liquidity` (from average daily volume), `tracking` (from tracking difference), `stability` (from AUM). | Factor, metric |
| **Candidate** | An ETF returned by the search, before eligibility and constraints. | Match, hit |
| **Result** | A holdable candidate that survived the constraints and made the top `maxResults`. | Recommendation, pick |
| **Alternative** | A surviving candidate shown alongside the results (up to 5) because it wasn't holdable or didn't make the cut. | Suggestion, runner-up |
| **Warning** | A response-level advisory with a code, e.g. `UNVERIFIED_AUM`, `NO_ELIGIBLE_RESULTS`. | Alert, notice |
| **Eligibility warning** | A `⚠` reason inside one ETF's justification. | Warning (on its own) |
| **Unverified constraint** | A constraint that couldn't be checked for an ETF because of an unknown value; the ETF is kept and flagged. | Skipped filter, ignored constraint |
| **Cache hit** | A discovery whose search was served from the one-hour cache without contacting any data source. | Cached result |

## Relationships

- An **Investor** has exactly one **Investor profile**, which holds one **Investor country** and one **Account type**; together these must form a **Supported pair**.
- Each **Supported pair** is judged by at most one **Eligibility rule** (the first registered rule that applies). Without one, a **Standard account** gets `eligible` and any **Tax-advantaged account** gets `unknown`.
- An **Eligibility rule** has many **Criteria**. Each **Criterion** passes, fails or becomes a **Skipped criterion**.
- An **ETF** has one **Issuer**, one **Exchange**, one **Domicile** and one or more **Data sources** (its **Provenance**).
- A **Discovery** produces many **Candidates**. Each **Candidate** becomes a **Result**, an **Alternative**, or is dropped.
- A **Constraint** that hits an **Unknown value** becomes an **Unverified constraint** and raises one **Warning** listing the affected **Tickers**.

## Example dialogue

> **Dev:** "A ZA **TFSA** **discovery** came back with no **results**. Did the **constraints** remove everything?"
>
> **Domain expert:** "No, look at the **alternatives**. Every **candidate** is `unknown`: the **issuer** criterion is a **skipped criterion**, because no **data source** reported who issues the fund, and that drops **confidence** to `low`."
>
> **Dev:** "But the min-AUM **constraint** should still have filtered them. **AUM** is zero."
>
> **Domain expert:** "Zero is an **unknown value**, not a tiny fund. Under **never guess** it's an **unverified constraint**: the **ETF** stays and the response carries an `UNVERIFIED_AUM` **warning**."
>
> **Dev:** "And for a 401(k), even with full data, the best we'd say is **conditional**?"
>
> **Domain expert:** "Right. Whether the fund is on the **plan menu** is a **provider-dependent constraint**. The **account provider** decides, not the law, so the fund is **holdable** but never plain **eligible**."

## Flagged ambiguities

- **"provider"** means three different things. In the code, `search.Provider` and `LiveProvider` are **data sources**, `ETF.Provider` is the **issuer**, and the ADR's "account provider" is the **Account provider**. Always say which. Recommendation: rename `ETF.Provider` → `Issuer` in a future refactor.
- **"eligible"** is used both for the status `eligible` and for the API flag `isEligible`, which is also `true` for `conditional`. Use **Holdable** for the flag's meaning; reserve **Eligible** for the status.
- **`tfsaEligibleOnly`** filters to **holdable** ETFs for *every* account type, not just TFSAs. The name is misleading; read it as "holdable only". Recommendation: rename in a future API version.
- **"liquidity"** has meant three things: the **average daily volume** constraint (`minLiquidity`), the **liquidity** ranking component (a 0–1 score from volume), and the GUI's old 0–100% "liquidity score". Only the first two exist now; say "minimum average daily volume" for the constraint.
- **"stability"** is the ranking component derived from **AUM**; the GUI labels it "Fund Size (AUM)". Same concept, so prefer "stability (AUM)" in docs.
- **"country"** has meant the **investor country**, a requested **market**, an ETF's **domicile** and its exchange country. Always qualify it.
- **"market"** and **"region"**: the request uses markets (`usa`, `US`); ETF breakdowns use regions. Matching compares them by name, and the two vocabularies don't line up yet (known gap 9).
- **"warning"** covers both response-level **warnings** (with codes) and per-ETF **eligibility warnings** (`⚠` reasons). Qualify which.
- **"unknown"** is both an **eligibility status** and the general state of an **unknown value**. Say "`unknown` status" or "unknown value".
- **"rule"** has meant both a whole **eligibility rule** (`TFSA_ZA`) and a single **criterion** (the API's `rulesPassed` / `rulesFailed` list criteria). Prefer **criterion** for the single check.
- **"TFSA"** was labelled as a Canadian account in the GUI. In UPSTONK it is always the South African account.
