# ETF eligibility: UK Stocks & Shares ISA, US IRA / Roth IRA, US 401(k)

Research notes for the eligibility rules engine (`server/internal/service/eligibility/rules/`).
All sources were accessed on **2026-11-15**. Quotations are taken from the versions published on that date.
These are research notes, not legal advice. Where a point could not be confirmed from a primary source, the notes say so.

## Summary

- **UK Stocks & Shares ISA.** The law decides which ETFs qualify, and part of that test can be run on ETF data. An ETF qualifies if it is a UK UCITS or an FCA-recognised UCITS. It also qualifies if it is a qualifying non-UCITS retail scheme, or if its shares are officially listed on (or, in the UK/EEA, admitted to trading on) an HMRC-designated "recognised stock exchange". An ETF that fails every one of those routes can be marked `ineligible`. Separately, FCA disclosure rules stop UK firms from selling an ETF to a retail client unless a product summary (or, until 7 June 2027, a KID) exists. US-domiciled ETFs have no such document, so in practice they are unavailable to retail investors. That is a rule on the firm, not an ISA tax rule, so it should produce `conditional`.
- **US traditional and Roth IRA.** The Internal Revenue Code has no list of permitted investments. It forbids life insurance, treats collectibles as distributions, and forbids prohibited transactions. None of these makes an ordinary exchange-traded ETF ineligible, so no ETF attribute should produce `ineligible`. In practice, whether you can hold a given ETF depends on the custodian. Non-US-domiciled ETFs are generally not offered in the US because of securities law, not IRA law. Leveraged, inverse, commodity and partnership ETFs carry custodian-dependent or tax caveats.
- **US 401(k).** Neither IRC §401(k) nor ERISA lists permitted investments. A participant can hold only what the plan's fiduciaries put on the menu, plus whatever a brokerage window allows, if the plan has one. ETF data alone can never settle 401(k) eligibility. The honest default is `unknown`, or `conditional` if the service's semantics allow it (see Open questions).

## Terms

- **UCITS** ("Undertakings for Collective Investment in Transferable Securities"): a fund regime originally from EU Directive 2009/65/EC. In the UK, a **UK UCITS** is a fund authorised by the FCA that complies with the UCITS requirements in the FCA's Collective Investment Schemes sourcebook (COLL). A **recognised UCITS** is an overseas UCITS (for example, an Irish or Luxembourg ETF) that the FCA has recognised so that it can be marketed in the UK.
- **Non-UCITS retail scheme (NURS):** a UK authorised fund that is open to retail investors but does not meet the UCITS rules.
- **Recognised stock exchange:** a market that HMRC has designated under section 1005 of the Income Tax Act 2007. Being "listed" on it has a specific statutory meaning (see 1.2).
- **KID** (Key Information Document): the pre-sale disclosure document required by the PRIIPs Regulation (PRIIPs means packaged retail and insurance-based investment products). In the UK, PRIIPs was replaced on 6 April 2026 by the **Consumer Composite Investments (CCI)** regime, which requires a **product summary** instead.
- **Reporting fund:** an offshore fund that has HMRC "reporting status", so that UK investors outside an ISA are taxed on gains as capital gains rather than income.
- **Collectible** (US): tangible personal property listed in IRC §408(m), such as art, metals and coins.
- **Prohibited transaction** (US): a transaction listed in IRC §4975 between a plan or IRA and a "disqualified person" (for example the account owner or their family).

---

## 1. UK Stocks & Shares ISA

### 1.1 The statutory list of qualifying investments

Regulation 7 of the Individual Savings Account Regulations 1998 (SI 1998/1870) sets out "the kind of investments ('qualifying investments for a stocks and shares component') which may be purchased, made or held under a stocks and shares component". On 2026-11-15, legislation.gov.uk showed the regulation as up to date with changes in force on or before 14 November 2026. Source: <https://www.legislation.gov.uk/uksi/1998/1870/regulation/7>

Four limbs of regulation 7(2) can apply to an ETF:

- **(a) Listed shares:** "shares, not being shares in an investment trust — (i) issued by a company wherever incorporated, and (ii) … either officially listed on a recognised stock exchange or, in the United Kingdom or the European Economic Area, admitted to trading on a recognised stock exchange".
- **(f) UCITS units:** "units in, or shares of, a UK UCITS or recognised UCITS". Note that this limb has **no listing condition**.
- **(g) NURS:** "qualifying units in or shares of a non-UCITS retail scheme".
- **(ga) Listed UCITS/NURS shares:** "shares of a UK UCITS, a recognised UCITS or a non-UCITS retail scheme that are listed on or admitted to trading on a recognised stock exchange".

Limbs (b) (qualifying securities), (d) (investment trusts) and (ha) (depositary interests) cover neighbouring products such as exchange-traded commodities (ETCs) issued as debt securities, and depositary interests. They are not ETFs and are not analysed further here. One exclusion matters for "ETP" data feeds. Regulation 7(2G) says: "A UK cryptoasset exchange traded note is not a qualifying security for the purposes of paragraph (2)(b)." This took effect on 6 April 2026, and holdings acquired before that date are grandfathered under reg 7(2H).

There is **no currency condition**, **no leverage or inverse condition**, **no issuer condition** and **no reporting-fund condition** anywhere in regulation 7. These conclusions come from reading the full text of the regulation, which names none of them.

The definitions are in regulation 2 (<https://www.legislation.gov.uk/uksi/1998/1870/regulation/2>):

- "UK UCITS" means "a collective investment scheme with Part 4A permission under FISMA 2000, which complies with the requirements to be a 'UCITS scheme' for the purposes of the COLL", including sub-funds of an umbrella.
- "recognised UCITS" means a Gibraltar UCITS, or "a recognised scheme by virtue of section 271A or 272 of FISMA 2000 that is a UCITS within the meaning given by section 236A of that Act". Section 271A is the Overseas Funds Regime (OFR) recognition route.
- "qualifying units in or shares of a non-UCITS retail scheme" requires that the scheme's instrument "secures that redemption … shall take place no less frequently than bi-monthly".
- "recognised stock exchange" "has the same meaning as in section 1005 of ITA 2007".

HMRC's guidance for ISA managers restates these rules ("Stocks and shares ISA investments for ISA managers", last updated 6 April 2026): <https://www.gov.uk/guidance/stocks-and-shares-investments-for-isa-managers>. It says that qualifying shares are those "issued by a company that is incorporated anywhere in the world" and "either officially listed on a recognised stock exchange or are admitted to trading on a recognised stock exchange in the UK or the EEA". It also says that "Units or shares in a UK UCITS, or a part of a UK UCITS, are qualifying investments". It lists futures, share options, warrants and nil-paid rights as non-qualifying. **It does not mention ETFs by name.** Leveraged products are also not mentioned.

### 1.2 "Recognised stock exchange" and "listed"

Section 1005 of the Income Tax Act 2007 (<https://www.legislation.gov.uk/ukpga/2007/3/section/1005>) defines a recognised stock exchange as a market that HMRC designates by order. Subsection (3) defines "listed": securities "listed on a recognised stock exchange" are securities "(a) which are admitted to trading on that exchange, and (b) which are included in the official UK list or are officially listed in a qualifying country outside the United Kingdom in accordance with provisions corresponding to those generally applicable in EEA states".

HMRC publishes the designations as tables:

- By country ("shorter tables", updated 31 July 2026): <https://www.gov.uk/government/publications/recognised-stock-exchanges-section-1005-income-tax-act-2007/shorter-tables>
  - **United States:** "Any exchange registered with the Securities and Exchange Commission of the United States (SEC) as a national securities exchange under section 6 of the Securities Exchange Act of 1934 is a recognised stock exchange for UK tax purposes". The named exchanges include NYSE, NYSE Arca, NYSE American, Nasdaq and "BATs Exchange Inc" (now Cboe BZX). Their main markets are shown as "Listed". OTCBB and Pink Sheets are expressly "not a recognised stock exchange".
  - **Ireland:** Euronext Dublin is listed. Euronext Growth and Euronext Access Dublin are "not listed".
  - **Germany:** the Frankfurt Stock Exchange and other German exchanges are listed for their "Regulierter Markt". The "Freiverkehr" (open) markets are "not listed".
  - **France:** Euronext Paris is listed. Euronext Growth and Access are not.
- By designated exchange (updated 21 April 2026): <https://www.gov.uk/government/publications/designated-recognised-stock-exchanges-section-1005-income-tax-act-2007/designated-recognised-stock-exchanges-section-1005-income-tax-act-2007-v3>
  - **London Stock Exchange:** the Main Market is listed. AIM and the Specialist Fund Market are "not listed".
  - **SIX Swiss Exchange:** listed for securities under its International Reporting Standard, Swiss Reporting Standard or Bonds Standard.

HMRC adds that "whether a share or security is 'listed' is a question of fact, HMRC does not designate 'listed' status".

**What this means for the rules engine.** The test depends on both the exchange and the **market segment**. NYSE Arca's main market passes. A German Freiverkehr quotation does not. The `exchange` / `exchangeCountry` attributes are only enough if they identify the segment.

### 1.3 UCITS recognition: a deadline at the end of 2026

Since 1 October 2024, an EEA UCITS ETF (for example, an Irish-domiciled iShares or Vanguard UCITS ETF) is a "recognised UCITS" only if it is recognised under FSMA s271A (OFR) or s272. A transitional rule, reg 7(2D), treats investments that were recognised UCITS immediately before 1 October 2024 (or sub-funds of such umbrellas) as qualifying "until the end of 31st December 2026". Reg 7(2E) also ends the older Brexit grandfathering in reg 7(2A) "at the end of 31st December 2026". Source: <https://www.legislation.gov.uk/uksi/1998/1870/regulation/7>.

The FCA ran the Temporary Marketing Permissions Regime (TMPR) "landing slots" for EEA funds to apply for OFR recognition. The final slot ran from 1 July to 30 September 2026. A fund that did not apply "will lose [its] recognition under the TMPR immediately after the end of [its] landing slot". The page also says that loss of recognition "may also have personal taxation consequences for UK investors". Source: <https://www.fca.org.uk/firms/temporary-permissions-regime/landing-slots-funds-temporary-marketing-permissions-regime> (page last updated 16 January 2025).

**What this means for the rules engine.** `domicile = IE/LU` plus `legalStructure = UCITS` is **not enough** to prove ISA qualification under limb (f). The fund must also be on the FCA register as a recognised scheme. From 1 January 2027, the transitional cover in reg 7(2D) ends. A UCITS ETF that is not recognised can still qualify under limb (a) or (ga) if its shares are officially listed on a recognised stock exchange. Whether an Irish ICAV or plc sub-fund's shares count as "shares … issued by a company" for limb (a) is not addressed in HMRC's guidance (see Open questions).

### 1.4 US-domiciled ETFs: tax eligibility versus retail access

**Tax side.** A US ETF is not a UK UCITS. It is not a recognised UCITS either, because the FCA says "There are currently no US funds that are recognised schemes in the UK" (PS25/20, para 2.8, below). It can therefore qualify only under limb (a), as shares "issued by a company wherever incorporated" that are officially listed on a recognised stock exchange. US national securities exchanges are recognised stock exchanges (1.2). **No primary source found says whether a US ETF (often a Delaware statutory trust or a Maryland corporation registered under the Investment Company Act) counts as "shares … issued by a company" for limb (a).** This is unverified.

**Distribution side.** This is a rule on the firm, not the ISA:

- The UK PRIIPs regime was replaced on **6 April 2026** by the Consumer Composite Investments (CCI) regime. The commencement legislation is the Consumer Composite Investments (Designated Activities) (Amendment) Order 2025 (<https://www.legislation.gov.uk/uksi/2025/1347/made>), and the FCA rules are in instrument FCA 2025/52 (<https://api-handbook.fca.org.uk/files/instrument/GLOSSARY-GEN-COBS-BCOBS-PROD-ESG-DEPP-COLL-FUND-PRM-DISC-ENFG/FCA%202025/52-2026-04-06.pdf>).
- DISC 2A.3.1R (in FCA 2025/52) says: "A firm must not distribute a consumer composite investment to a retail investor unless" it ensures "an up-to-date product summary for the consumer composite investment is available" and makes it available to the investor. In DISC, "distribute" includes "dealing, or arranging a deal in an investment with or for an investor where the investor is to enter into the deal as buyer". That covers execution-only purchases.
- **Transition:** DISC TP 2 says "The transitional period begins on 6 April 2026 and ends on 7 June 2027". During that period a manufacturer may produce either a product summary or the document it previously had to produce: a KID, a KIID (the UCITS Key Investor Information Document) or an EEA KIID. DISC TP 2.5R prohibits distribution to a retail investor without that disclosure document. A separate exemption for TMPR-recognised manufacturers that keep producing a KIID "expires on 1 January 2027" (DISC TP 2, the provision just before TP 2.4G).
- In **PS25/20** (<https://www.fca.org.uk/publication/policy/ps25-20.pdf>, para 2.8) the FCA addressed US ETFs directly. Respondents said the KID requirement "prevented UK-based retail consumers from accessing US ETFs and this would persist". The FCA replied: "An overseas fund can only be marketed to UK retail investors if it is a 'recognised scheme' … There are currently no US funds that are recognised schemes in the UK. … The decision to enter the UK retail investment market is a commercial one for those firms." The same document says "Distributors must not direct or sell any CCI products that do not have an up-to-date product summary to a retail client." The PS25/20 web page says "The rules in this PS will come into effect on 8 June 2027" (<https://www.fca.org.uk/publications/policy-statements/ps25-20-supporting-informed-decision-making-final-rules-consumer-composite-investments>).
- **Still changing:** FCA CP26/24 "Simplifying Consumer Investment Disclosures" (July 2026, comments closed 21 August 2026) proposes further changes to COBS cost disclosure to align it with CCI (<https://www.fca.org.uk/publication/consultation/cp26-24.pdf>). From the parts read, it does not change the product-summary requirement for overseas ETFs. Not fully verified.

**Result.** For a retail client, a UK ISA manager may not sell a US ETF unless the manufacturer has produced a product summary (or, until 7 June 2027, a KID). Professional clients are outside DISC's retail rule. The ETF data model has no "UK product summary / KID available" attribute. A `domicile = US` ETF should therefore be `conditional`: possibly ISA-qualifying in law, but generally not available to retail clients. It should not be `ineligible`, because the ISA Regulations themselves do not exclude it.

### 1.5 Reporting-fund status is not an ISA criterion

Regulation 7 does not mention reporting status. HMRC helpsheet HS265 (last updated 6 April 2026) says: "Investments in non-reporting and reporting offshore funds can be held within the ISA wrapper" and "If your investment in the fund is held through an ISA, then you do not need to declare the income or gains from these funds." Source: <https://www.gov.uk/government/publications/offshore-funds-self-assessment-helpsheet-hs265/hs265-offshore-funds>. Reporting status matters only for investors holding the fund outside an ISA.

### 1.6 Provider-side constraints (not law)

- **Product range.** Nothing in the ISA Regulations obliges an ISA manager to offer every qualifying investment. Whether a given ETF is offered is the provider's choice.
- **Product summary / KID availability** for overseas funds (1.4).
- **Leveraged and inverse ETFs.** Neither the ISA Regulations nor the HMRC guidance excludes them. Any restriction comes from the provider (product governance, appropriateness checks). No primary source states a legal ISA ban, and none was found.

### 1.7 Recent and scheduled ISA changes

- **6 April 2026** (SI 2026/248, <https://www.legislation.gov.uk/uksi/2026/248/made>): long-term asset funds added (reg 7(2)(s)), and UK cryptoasset ETNs removed from stocks and shares ISAs (reg 7(2G)).
- **31 December 2026:** the recognised-UCITS transitional rules in reg 7(2D) and 7(2A) end (1.3).
- **SI 2026/1018** (<https://www.legislation.gov.uk/uksi/2026/1018/made>), due in force on 6 April 2027, would add reg 7(2)(t) and (10A). Under (10A), "100% of the value of the investments, other than cash, held under the stocks and shares component, must not be money market funds". The same instrument introduced the £12,000 cash ISA limit. HMRC's Tax-free savings newsletter 24 (October 2026, <https://www.gov.uk/government/publications/tax-free-savings-newsletter-24/tax-free-savings-newsletter-24-october-2026>) says these regulations "will be revoked" and that "the annual cash ISA limit of £20,000 will be retained". **Unverified:** whether the revoking instrument has been made, and whether the money-market-fund condition is revoked along with the cash limit. This matters for money-market ETFs.

### 1.8 ISA summary table

| Criterion | Legal source (URL) | Checkable from ETF data? | Suggested rule outcome when data is missing |
|---|---|---|---|
| ETF is a UK UCITS (FCA-authorised UCITS scheme) | ISA Regs reg 7(2)(f), reg 2 — <https://www.legislation.gov.uk/uksi/1998/1870/regulation/7> | Partly: `legalStructure = UCITS` plus `domicile = GB`. Full proof needs FCA register status. | Data missing: `unknown` (cannot tell which route applies). UCITS + GB but FCA status not checked: `conditional` ("UK UCITS; confirm on FCA register"). |
| ETF is a recognised UCITS (s271A/s272 FSMA) | reg 7(2)(f), reg 2; transition reg 7(2D) to 31 Dec 2026 — same URL | Partly: `legalStructure = UCITS` plus non-UK `domicile` suggests it. Recognition is an FCA register fact, not an ETF attribute. | `conditional` ("EEA UCITS; ISA-qualifying only if FCA-recognised; transitional cover ends 31 Dec 2026"). Never `eligible` from domicile alone. |
| ETF is a qualifying NURS (redemption at least bi-monthly) | reg 7(2)(g), reg 2 — same URL | Partly: `legalStructure` would need a NURS value. Redemption terms are not in the data. | `conditional` |
| ETF shares officially listed on a recognised stock exchange (or admitted to trading on one in the UK/EEA) | reg 7(2)(a), (ga); ITA 2007 s1005 — <https://www.legislation.gov.uk/ukpga/2007/3/section/1005>; HMRC tables — <https://www.gov.uk/government/publications/recognised-stock-exchanges-section-1005-income-tax-act-2007/shorter-tables> | Yes, from `exchange` / `exchangeCountry`, if the market segment is known (for example Main Market vs AIM, Regulierter Markt vs Freiverkehr). | Exchange missing: `unknown`. Exchange known but segment ambiguous: `conditional`. |
| **Fails every route**: not UK/recognised UCITS, not NURS, and only quoted on a non-recognised market or segment (for example OTC/Pink Sheets) | reg 7(2) taken as a whole | Yes, but only when `legalStructure` **and** the listing are both known. | `ineligible`. This is the only legal `ineligible` path for an ETF. Require positive evidence on both limbs. If either is missing: `unknown`. |
| US ETF counts as "shares issued by a company" (limb (a)) | reg 7(2)(a) — not addressed by HMRC guidance | No (legal-characterisation question) | `conditional` (unverified legal point; see Open questions) |
| Product summary / KID exists, so the firm may sell to a retail client | FCA DISC 2A.3.1R, DISC TP 2.5R (FCA 2025/52); PS25/20 para 2.8 — <https://www.fca.org.uk/publication/policy/ps25-20.pdf> | No attribute. `domicile = US` is a strong signal that none exists. | `conditional` ("not normally available to UK retail investors: no UK product summary/KID"). Firm-level rule, not ISA law, so not `ineligible`. |
| UK cryptoasset ETN (not an ETF) | reg 7(2G) | Only if the data distinguishes ETNs from ETFs | `ineligible` for new purchases since 6 Apr 2026, if identified |
| Currency | none in reg 7 | n/a | No rule (does not affect eligibility) |
| Leveraged / inverse | none in reg 7 or HMRC guidance | Yes (`isLeveraged`, `isInverse`) | `conditional` ("legally permitted; many providers restrict"). Never `ineligible`. |
| Provider / issuer | none in reg 7 | Yes (`provider`) | No legal rule. Provider range is `conditional`. |
| Reporting-fund status | not a reg 7 criterion; HS265 — <https://www.gov.uk/government/publications/offshore-funds-self-assessment-helpsheet-hs265/hs265-offshore-funds> | n/a | No rule |
| Money-market ETF (proposed cap) | SI 2026/1018 reg 9 (6 Apr 2027; announced for revocation) | Partly (`assetClass`) | No rule until the position is confirmed; at most `conditional` |

---

## 2. US traditional IRA and Roth IRA

### 2.1 The statute contains no list of permitted investments

IRC §408(a) defines an individual retirement account as "a trust created or organized in the United States for the exclusive benefit of an individual or his beneficiaries". Its requirements include that the trustee be a bank or a person approved by the Secretary (§408(a)(2)) and that "No part of the trust funds will be invested in life insurance contracts" (§408(a)(3)). Source: <https://www.law.cornell.edu/uscode/text/26/408>.

A Roth IRA "shall be treated for purposes of this title in the same manner as an individual retirement plan", except as §408A provides (§408A(a), <https://www.law.cornell.edu/uscode/text/26/408A>). §408A's differences concern contributions, deductibility, conversions and distributions, not permitted investments. Everything below therefore applies equally to traditional and Roth IRAs.

The IRS states that "there is no list of approved investments for retirement plans". It adds that IRAs "are not permitted to invest in life insurance" and that "both participant-directed accounts and IRAs cannot invest in collectibles". Source: "Retirement plan investments FAQs", reviewed 8 April 2026, <https://www.irs.gov/retirement-plans/retirement-plan-investments-faqs>. Publication 590-A (2025) gives the example "your IRA may purchase shares of stock" (<https://www.irs.gov/publications/p590a>).

### 2.2 Collectibles (§408(m))

Under §408(m)(1), acquiring a collectible in an IRA "shall be treated as a distribution". Under (m)(2), collectibles are "any work of art", "any rug or antique", "any metal or gem", "any stamp or coin", "any alcoholic beverage", and "any other tangible personal property specified by the Secretary". Under (m)(3), certain coins are excluded, as is bullion meeting futures-contract fineness "if such bullion is in the physical possession of a trustee described under subsection (a)". Source: <https://www.law.cornell.edu/uscode/text/26/408>. Publication 590-B (2025) repeats this under "Investment in Collectibles" (<https://www.irs.gov/publications/p590b>).

**ETFs that hold physical metal.** An ETF share is a security, not tangible personal property. The only IRS material found is non-precedential. In PLR 201446030 (18 August 2014, <https://www.irs.gov/pub/irs-wd/201446030.pdf>) the IRS concluded that "the acquisition of Shares by the trustee or custodian of an Account will not constitute the acquisition of a collectible within the meaning of section 408(m)". It added that if shares "are exchanged for Gold, such exchange would constitute the acquisition of a collectible … except to the extent the Gold acquired by the Account satisfies section 408(m)(3)". The letter itself warns that under §6110(k)(3) it "may not be used or cited as precedent". **No precedential ruling (revenue ruling or regulation) on metal-backed ETFs was found.**

### 2.3 Prohibited transactions (§4975) and pledging

§4975(c)(1) prohibits "any direct or indirect" sale, exchange or lease of property, lending, furnishing of goods or services, or transfer or use of plan assets between a plan and a "disqualified person". It also prohibits fiduciary self-dealing. §4975(e)(1)(B) includes "an individual retirement account described in section 408(a)" in the definition of "plan". Source: <https://www.law.cornell.edu/uscode/text/26/4975>. If the owner engages in a prohibited transaction, "the account stops being an IRA as of the first day of that year" (Pub 590-B, <https://www.irs.gov/publications/p590b>; §408(e)(2)). Using the account as security for a loan is treated as a distribution of that portion (§408(e)(4)).

Pub 590-B warns that the prohibited-transaction risk "may be increased" for "nonpublicly traded assets or assets that you directly control". Buying a publicly traded ETF on an exchange is not, in itself, any of the listed transactions. These rules are about **who the counterparty is and how the account is used**, so they cannot be evaluated from ETF attributes. (An edge case: an issuer that is itself a disqualified person, such as a fund company the owner controls. That cannot be detected from `provider`.)

### 2.4 Tax consequences that do not affect eligibility

- **Unrelated business taxable income (UBTI).** Pub 590-B: "An IRA is subject to tax on unrelated business income if it carries on an unrelated trade or business … regularly carried on by the IRA or by a partnership of which it is a member". An IRA with $1,000 or more of such income must file Form 990-T. This can affect ETFs structured as partnerships (some commodity-pool and MLP products). It is a tax cost, not a bar.
- **Passive foreign investment company (PFIC) reporting** for non-US funds. Treas. Reg. §1.1298-1 (<https://www.law.cornell.edu/cfr/text/26/1.1298-1>) and the Form 8621 instructions (<https://www.irs.gov/instructions/i8621>) relieve tax-exempt holders, including IRAs, of the annual filing in most cases. **Not fully verified:** the current list of exempt accounts in the 2025 instructions was not read in full.

### 2.5 Domicile: securities law, not IRA law

Nothing in §408 or §408A requires a US-domiciled fund. However, Investment Company Act §7(d) (15 U.S.C. 80a-7(d), <https://www.law.cornell.edu/uscode/text/15/80a-7>) provides that "No investment company, unless organized or otherwise created under the laws of the United States or of a State … shall make use of the mails or any means or instrumentality of interstate commerce … to offer for sale, sell, or deliver after sale, in connection with a public offering, any security of which such company is the issuer". In practice, non-US ETFs (for example UCITS ETFs) are not publicly offered to US investors, and US custodians generally do not make them available. This is an availability constraint (`conditional`), not an IRA-law ineligibility.

### 2.6 Provider-side constraints (not law)

- **What the custodian offers.** The IRA trustee or custodian decides which investments it will hold. No IRS text was found that requires a custodian to offer any particular investment.
- **Leveraged and inverse ETFs.** Nothing in §408, §408A, §408(m) or §4975 mentions them. Buying a leveraged ETF does not mean the IRA borrows (the leverage is inside the fund), so §408(e)(4) is not triggered. Any restriction comes from the custodian or broker. **No IRS source addresses leveraged ETFs directly.** This conclusion rests on the absence of any such rule in the statute.
- **Margin and short selling** inside an IRA are limited because lending to or by the account and pledging it are prohibited (§4975(c)(1)(B), §408(e)(4)). This affects trading strategies, not which ETF you may buy.

### 2.7 IRA summary table

| Criterion | Legal source (URL) | Checkable from ETF data? | Suggested rule outcome when data is missing |
|---|---|---|---|
| Not a life insurance contract | IRC §408(a)(3) — <https://www.law.cornell.edu/uscode/text/26/408> | Effectively yes: an ETF is never a life insurance contract | No rule needed. Always satisfied for ETFs. |
| Not a collectible | IRC §408(m) — same URL; Pub 590-B — <https://www.irs.gov/publications/p590b> | Partly: `assetClass` / `assetExposure.commodities` and `isPhysical` flag metal-backed ETFs | Ordinary ETF: satisfied. Physically backed precious-metal ETF: `conditional` ("only non-precedential PLRs say shares are not collectibles; in-kind redemption for metal would be a collectible"). Never `ineligible`. |
| No prohibited transaction / pledging | IRC §4975 — <https://www.law.cornell.edu/uscode/text/26/4975>; §408(e) | No (depends on parties and conduct, not the ETF) | Not an ETF rule. At most an informational note. |
| Domicile (US vs non-US) | Not an IRA rule; Investment Company Act §7(d) — <https://www.law.cornell.edu/uscode/text/15/80a-7> | Yes (`domicile`) | Non-US: `conditional` ("generally not offered to US investors"). Domicile missing: `unknown`. |
| Exchange / listing | None in IRA law | Yes (`exchange`) | No legal rule. Unlisted or OTC: `conditional` (custodian may refuse; prohibited-transaction risk is higher for non-public assets per Pub 590-B). |
| Legal structure (partnership vs '40 Act fund/trust) | UBTI: Pub 590-B; IRC §511 et seq. | Partly (`legalStructure`, if it records partnership status) | Partnership: `conditional` ("may generate UBTI / Form 990-T"). Not a bar. |
| Leveraged / inverse | None in IRA law | Yes (`isLeveraged`, `isInverse`) | `conditional` ("legal; custodian may restrict") |
| Currency | None | n/a | No rule |
| Provider / issuer | None (except the §4975 edge case) | Yes (`provider`) | No rule |
| Custodian actually offers the ETF | Provider-side | No | `conditional` |
| **Anything that yields `ineligible`** | — | — | **Nothing from ETF data alone.** |

---

## 3. US 401(k)

### 3.1 Law: prudence and the plan menu

The IRC sets no investment list for 401(k) plans. The IRS states that "there is no list of approved investments for retirement plans" and that plan fiduciaries are "required, in investing plan assets, to exercise the judgment that a prudent investor would use" (<https://www.irs.gov/retirement-plans/retirement-plan-investments-faqs>). §401(k) defines the cash-or-deferred arrangement. It does not regulate which securities the trust holds.

ERISA §404(a)(1) (29 U.S.C. 1104, <https://www.law.cornell.edu/uscode/text/29/1104>) requires fiduciaries to act for the exclusive purpose of providing benefits; "with the care, skill, prudence, and diligence under the circumstances then prevailing"; to diversify "so as to minimize the risk of large losses, unless under the circumstances it is clearly not prudent to do so"; and "in accordance with the documents and instruments governing the plan". These are standards of conduct for the people who build the menu, not a test applied to individual securities.

Under ERISA §404(c), a participant who exercises control over their account "shall not be deemed to be a fiduciary by reason of such exercise". The regulation at 29 CFR 2550.404c-1 (<https://www.law.cornell.edu/cfr/text/29/2550.404c-1>) requires that participants be able to "Choose from at least three investment alternatives", each diversified. It defines a designated investment alternative as one "identified by a plan fiduciary as an available investment alternative under the plan".

29 CFR 2550.404a-5(h)(4) (<https://www.law.cornell.edu/cfr/text/29/2550.404a-5>) defines a "designated investment alternative" as one "designated by the plan into which participants and beneficiaries may direct the investment". It excludes "'brokerage windows,' 'self-directed brokerage accounts,' or similar plan arrangements that enable participants and beneficiaries to select investments beyond those designated by the plan".

**Consequence:** a participant can hold an ETF only if (a) the plan designates it, or (b) the plan offers a brokerage window and the window's terms allow it. Both depend on the plan.

### 3.2 Other rules

- **Collectibles.** §408(m)(1) also applies to "any individually-directed account under a plan described in section 401(a)", so the IRA collectibles analysis in 2.2 carries over to self-directed 401(k) accounts (<https://www.law.cornell.edu/uscode/text/26/408>).
- **Prohibited transactions.** §4975 applies to qualified plans, and ERISA §406 is its parallel. As with IRAs, these rules concern parties and conduct, not ETF attributes.
- **Foreign custody.** ERISA §404(b) restricts holding "indicia of ownership" of plan assets outside US jurisdiction. 29 CFR 2550.404b-1 (<https://www.law.cornell.edu/cfr/text/29/2550.404b-1>) permits securities of issuers "not organized under the laws of the United States or a State" when they are held by a qualifying US custodian (for example a US bank, or a broker-dealer with its "principal place of business in the United States"). This is a plan-operations rule, not an ETF bar. Non-US ETFs also face the Investment Company Act §7(d) constraint described in 2.5.

### 3.3 Recent and pending changes

The DOL proposed "Fiduciary Duties in Selecting Designated Investment Alternatives" on 31 March 2026 (RIN 1210-AC38, implementing Executive Order 14330 on alternative assets in 401(k) plans). It would create process-based safe harbours for choosing menu options. Sources: <https://www.federalregister.gov/documents/2026/03/31/2026-06178/fiduciary-duties-in-selecting-designated-investment-alternatives> and <https://www.dol.gov/newsroom/releases/ebsa/ebsa20260330>. The 2026 Unified Agenda (August 2026) still lists it at the proposed-rule stage. **No final rule was found** as of 2026-11-15. It concerns how fiduciaries choose the menu and would not change the fact that holdings are limited to the menu.

### 3.4 401(k) summary table

| Criterion | Legal source (URL) | Checkable from ETF data? | Suggested rule outcome when data is missing |
|---|---|---|---|
| ETF is on the plan's designated menu, or reachable through a brokerage window | ERISA §404(a)(1)(D), §404(c) — <https://www.law.cornell.edu/uscode/text/29/1104>; 29 CFR 2550.404a-5(h)(4) — <https://www.law.cornell.edu/cfr/text/29/2550.404a-5> | No (plan data, not ETF data) | `unknown` by default (most plans do not offer arbitrary ETFs, so "eligible with warnings" overstates it). `conditional` only if the service's semantics treat conditional as "depends on your plan" (see Open questions). Never `eligible` or `ineligible` from ETF data. |
| Fiduciary prudence / diversification | ERISA §404(a)(1)(B)-(C) | No (judgement about the menu as a whole) | Not an ETF rule |
| Not a collectible (individually-directed accounts) | IRC §408(m)(1) — <https://www.law.cornell.edu/uscode/text/26/408> | Partly (metal-backed: `assetClass`, `isPhysical`) | As for IRA: `conditional` for physically backed metal ETFs |
| Foreign-domiciled ETF | ERISA §404(b); 29 CFR 2550.404b-1 — <https://www.law.cornell.edu/cfr/text/29/2550.404b-1>; ICA §7(d) | Yes (`domicile`) | `conditional` |
| Leveraged / inverse | None in ERISA or IRC | Yes | `conditional` (plan or window terms decide; fiduciaries may exclude) |
| Exchange, currency, provider | None | Yes | No legal rule |
| **Anything that yields `ineligible`** | — | — | **Nothing from ETF data alone.** |

---

## Open questions

1. **US ETFs under ISA limb (a).** Does a US-registered ETF (Delaware statutory trust, Massachusetts business trust or Maryland corporation) issue "shares … issued by a company wherever incorporated" for reg 7(2)(a)? HMRC's ISA guidance does not say. An HMRC manual or a direct HMRC query would be needed before any `eligible` outcome for US ETFs.
2. **UCITS ETFs under ISA limb (a)/(ga) after 31 Dec 2026.** If an EEA UCITS ETF is not FCA-recognised, can it still qualify through its LSE/Euronext listing under limb (a)? Limb (ga) expressly requires the fund to be a UK UCITS, a recognised UCITS or a NURS, which suggests that listed fund shares are meant to qualify only through those routes. Not resolved from primary sources.
3. **FCA register data.** Recognised-scheme status (s271A/s272) is the deciding fact for most UCITS ETFs and is not in the ETF model. Can the service source it (the FCA Financial Services Register lists recognised schemes)? Until then, EEA UCITS ETFs should not be better than `conditional`.
4. **SI 2026/1018 revocation.** Has the revoking instrument been made, and does it also remove the money-market-fund condition (reg 7(10A))? HMRC newsletter 24 says only that the regulations "will be revoked".
5. **Execution-only access to US ETFs after 8 June 2027.** PS25/20 and DISC show the product-summary requirement applies to "distribution", which includes dealing as buyer. No FCA statement was found on whether a UK firm can let a retail client buy an overseas fund that has no product summary on a purely client-initiated basis. As read, DISC 2A.3.1R does not allow it.
6. **Semantics of `conditional` in the codebase.** `tfsa_za.go` sets `IsEligible = true` for `conditional` ("eligible but with warnings"). For a 401(k), where availability depends entirely on the plan menu, that may overstate eligibility. Consider whether `unknown` or a distinct reason code is more truthful.
7. **Boolean flags cannot express "unknown".** `domain.ETF.IsLeveraged`, `IsInverse` and `IsPhysical` are `bool`, so "data missing" and "false" look the same. Under "never guess", rules that depend on these flags (leveraged/inverse provider restrictions, metal-backed collectibles caveat) cannot tell missing data apart from a real `false`.
8. **Market segment in exchange data.** The ISA "listed" test depends on the market segment (LSE Main Market vs AIM; Regulierter Markt vs Freiverkehr). The `exchange` field may not carry the segment.
9. **Metal-backed ETFs in IRAs/401(k)s.** Only non-precedential PLRs address them. Is there a revenue ruling or other precedential guidance? None was found.
10. **PFIC filing for IRAs holding non-US funds.** The exemption list in the 2025 Form 8621 instructions was not read in full.

## Sources consulted (accessed 2026-11-15)

- ISA Regulations 1998, regs 2 and 7: <https://www.legislation.gov.uk/uksi/1998/1870/regulation/2>, <https://www.legislation.gov.uk/uksi/1998/1870/regulation/7>
- SI 2026/248: <https://www.legislation.gov.uk/uksi/2026/248/made>; SI 2026/1018: <https://www.legislation.gov.uk/uksi/2026/1018/made>
- ITA 2007 s1005: <https://www.legislation.gov.uk/ukpga/2007/3/section/1005>
- HMRC, Stocks and shares ISA investments for ISA managers: <https://www.gov.uk/guidance/stocks-and-shares-investments-for-isa-managers>
- HMRC recognised stock exchange tables: <https://www.gov.uk/government/publications/recognised-stock-exchanges-section-1005-income-tax-act-2007/shorter-tables>, <https://www.gov.uk/government/publications/designated-recognised-stock-exchanges-section-1005-income-tax-act-2007/designated-recognised-stock-exchanges-section-1005-income-tax-act-2007-v3>
- HMRC HS265 Offshore funds: <https://www.gov.uk/government/publications/offshore-funds-self-assessment-helpsheet-hs265/hs265-offshore-funds>
- HMRC Tax-free savings newsletter 24: <https://www.gov.uk/government/publications/tax-free-savings-newsletter-24/tax-free-savings-newsletter-24-october-2026>
- FCA TMPR landing slots: <https://www.fca.org.uk/firms/temporary-permissions-regime/landing-slots-funds-temporary-marketing-permissions-regime>
- FCA PS25/20 and FCA 2025/52 (DISC): <https://www.fca.org.uk/publication/policy/ps25-20.pdf>, <https://api-handbook.fca.org.uk/files/instrument/GLOSSARY-GEN-COBS-BCOBS-PROD-ESG-DEPP-COLL-FUND-PRM-DISC-ENFG/FCA%202025/52-2026-04-06.pdf>
- CCI commencement order SI 2025/1347: <https://www.legislation.gov.uk/uksi/2025/1347/made>
- FCA CP26/24: <https://www.fca.org.uk/publication/consultation/cp26-24.pdf>
- IRC §§408, 408A, 4975: <https://www.law.cornell.edu/uscode/text/26/408>, <https://www.law.cornell.edu/uscode/text/26/408A>, <https://www.law.cornell.edu/uscode/text/26/4975>
- IRS Pubs 590-A and 590-B (2025): <https://www.irs.gov/publications/p590a>, <https://www.irs.gov/publications/p590b>
- IRS Retirement plan investments FAQs: <https://www.irs.gov/retirement-plans/retirement-plan-investments-faqs>
- IRS PLR 201446030: <https://www.irs.gov/pub/irs-wd/201446030.pdf>
- 26 CFR 1.1298-1: <https://www.law.cornell.edu/cfr/text/26/1.1298-1>; Form 8621 instructions: <https://www.irs.gov/instructions/i8621>
- Investment Company Act §7(d): <https://www.law.cornell.edu/uscode/text/15/80a-7>
- ERISA §404 (29 U.S.C. 1104): <https://www.law.cornell.edu/uscode/text/29/1104>; 29 CFR 2550.404a-5, 2550.404b-1, 2550.404c-1: <https://www.law.cornell.edu/cfr/text/29/2550.404a-5>, <https://www.law.cornell.edu/cfr/text/29/2550.404b-1>, <https://www.law.cornell.edu/cfr/text/29/2550.404c-1>
- DOL proposed rule, 31 March 2026: <https://www.federalregister.gov/documents/2026/03/31/2026-06178/fiduciary-duties-in-selecting-designated-investment-alternatives>; press release: <https://www.dol.gov/newsroom/releases/ebsa/ebsa20260330>
