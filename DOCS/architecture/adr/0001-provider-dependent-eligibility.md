# ADR 0001: Eligibility constraints that depend on the account provider

- **Status:** accepted
- **Date:** 2026-11-15
- **Context for:** Phase 6 of [gaps-remediation](../../plans/gaps-remediation.md), the UK ISA and US IRA / Roth IRA / 401(k) rules

## Context

An eligibility rule answers "may this ETF be held in this account?". For South African TFSAs, most of the answer depends on the ETF (JSE listing, approved provider, not leveraged). For the next account types, much of it depends on **who runs the account**:

- A 401(k) can only hold what the employer's plan menu offers.
- UK and US brokers decide which funds they let retail clients buy (for example, whether a product needs a key information document, or whether leveraged products are offered). These are platform policies, not tax law.

UPSTONK knows nothing about the investor's platform or plan. On top of that, today's data sources don't fill `Domicile` or `LegalStructure` for any fund, and leveraged/inverse flags are guessed from the fund name (see [known gaps](../../development/known-gaps.md)).

The project's rule is "never guess": missing data must not become `eligible`.

## Decision

1. **Separate law from platform.** A rule checks only what the law says about the ETF. Constraints that depend on the account provider or plan are never evaluated as pass/fail.
2. **Provider-dependent constraints make an otherwise-passing result `conditional`**, never `eligible`. Each one adds a reason starting with `⚠` that names the constraint, e.g. "⚠ 401(k) holdings are limited to your plan's investment menu — check that this fund is offered". The reason surfaces in `eligibility.warnings` when `explainEligibility` is set.
3. **`ineligible` only on a definite legal failure** that the rule can show from data we actually have (e.g. a listing that the law excludes). A missing attribute is recorded in `RulesSkipped`, never treated as failing.
4. **Missing ETF data follows the existing TFSA convention:** each check with no data is skipped, and confidence drops. More than 3 skipped checks, or `low` confidence, gives `unknown`.
5. **Exception — menu-based accounts are `unknown`.** For a 401(k), the plan menu is the *whole* question, and `conditional` sets `isEligible: true`, which would make every fund look holdable. So the 401(k) rule always returns `unknown` (confidence `none`) with a plan-menu reason. Decided 2026-11-15.
6. **Every rule cites its sources.** The rule's doc comment links to the primary sources in `DOCS/research/`, and the rule version string changes when the legal reading changes.

## Consequences

- Implemented in Phase 6 as `ISA_UK`, `IRA_US` (traditional and Roth) and `401K_US`, sharing `rules/result.go`.
- Results for these accounts will mostly be `conditional` (or `unknown` with today's sparse data), not `eligible`. That's correct: the honest answer is "probably, but check with your provider/plan".
- `tfsaEligibleOnly: true` keeps `conditional` results (they have `isEligible: true`), so these funds still appear, with the warning attached.
- A rule can become stricter later (more `ineligible` outcomes) once sources supply domicile and legal structure, without changing this policy.
- Platform data (e.g. a specific broker's tradable list, or a plan menu upload) would be a separate future input, not something rules guess at.
