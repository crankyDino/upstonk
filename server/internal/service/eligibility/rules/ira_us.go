package rules

import (
	"context"
	"strings"

	"upstonk/internal/domain"
)

// IRAUSRules: US traditional IRA and Roth IRA.
// Law: IRC §408 and §408A (no list of permitted investments; §408(m)
// collectibles; §4975 prohibited transactions), IRS Pub 590-A/590-B,
// https://www.law.cornell.edu/uscode/text/26/408. Nothing in IRA law makes an
// ordinary ETF ineligible; what's offered is up to the custodian (ADR 0001).
// Research and citations: DOCS/research/etf-eligibility-isa-ira-401k.md §2.
type IRAUSRules struct{ version string }

func NewIRAUSRules() *IRAUSRules { return &IRAUSRules{version: "ira_us_v1.0_2026"} }

func (r *IRAUSRules) Name() string    { return "IRA_US" }
func (r *IRAUSRules) Version() string { return r.version }

func (r *IRAUSRules) AppliesTo(country, accountType string) bool {
	a := strings.ToLower(accountType)
	return strings.EqualFold(country, "US") && (a == "ira" || a == "roth_ira")
}

func (r *IRAUSRules) Evaluate(_ context.Context, etf domain.ETF) domain.EligibilityResult {
	b := newResult(r.version)
	b.providerWarning("IRA law allows ETFs generally; whether this one can be bought depends on your IRA custodian's product range")
	checkCollectibles(b, etf)
	checkUSAccess(b, etf)
	checkTaxCaveats(b, etf)
	if domain.IsTrue(etf.IsLeveraged) || domain.IsTrue(etf.IsInverse) {
		b.providerWarning("Leveraged and inverse ETFs are legal in an IRA, but many custodians restrict them")
	}
	return b.finish()
}

// checkCollectibles: §408(m) treats metals as collectibles. Only
// non-precedential private letter rulings (e.g. PLR 201446030) say shares of a
// physically backed metal ETF aren't. Shared with the 401(k) note.
func checkCollectibles(b *resultBuilder, etf domain.ETF) {
	commodity := strings.Contains(strings.ToLower(etf.AssetClass), "commodit")
	if commodity && !domain.IsFalse(etf.IsPhysical) {
		b.providerWarning("Possibly a physically backed metal ETF: only non-precedential IRS private letter rulings say these aren't collectibles under §408(m); confirm with your custodian")
	}
}

// checkUSAccess: Investment Company Act §7(d) keeps unregistered non-US funds
// off US platforms. Securities law, not IRA law.
func checkUSAccess(b *resultBuilder, etf domain.ETF) {
	if etf.Domicile != "" && !strings.EqualFold(etf.Domicile, "US") {
		b.providerWarning("Domiciled in " + etf.Domicile + ": non-US funds are generally not offered to US investors (Investment Company Act §7(d))")
	}
}

// checkTaxCaveats: partnership-structured funds can generate UBTI inside an
// IRA (Pub 590-B). Not a bar to holding them.
func checkTaxCaveats(b *resultBuilder, etf domain.ETF) {
	if strings.Contains(strings.ToLower(etf.LegalStructure), "partnership") {
		b.providerWarning("Partnership structure: may generate unrelated business taxable income (UBTI) and a Form 990-T filing inside an IRA")
	}
}
