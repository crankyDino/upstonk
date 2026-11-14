package rules

import (
	"context"
	"strings"

	"upstonk/internal/domain"
)

// K401USRules: US 401(k). What a participant can hold is the plan's
// designated investment menu (or a brokerage window), chosen by the plan
// fiduciary: ERISA §404(a), §404(c); 29 CFR 2550.404a-5,
// https://www.law.cornell.edu/uscode/text/29/1104. ETF data can never decide
// it, so the verdict is always unknown. This is a deliberate exception to
// ADR 0001's "provider-dependent means conditional": conditional sets
// isEligible, which would overstate availability for a menu-based account.
// Research and citations: DOCS/research/etf-eligibility-isa-ira-401k.md §3.
type K401USRules struct{ version string }

func NewK401USRules() *K401USRules { return &K401USRules{version: "k401_us_v1.0_2026"} }

func (r *K401USRules) Name() string    { return "401K_US" }
func (r *K401USRules) Version() string { return r.version }

func (r *K401USRules) AppliesTo(country, accountType string) bool {
	return strings.EqualFold(country, "US") && strings.EqualFold(accountType, "401k")
}

func (r *K401USRules) Evaluate(_ context.Context, etf domain.ETF) domain.EligibilityResult {
	b := newResult(r.version)
	b.skip("plan_menu", "Offered by your 401(k) plan", "plan menu not known",
		"401(k) holdings are limited to your plan's investment menu (or its brokerage window); check whether this ETF is offered", true)
	checkCollectibles(b, etf)
	checkUSAccess(b, etf)
	result := b.finish()
	result.Confidence = domain.ConfidenceNone
	return result
}
