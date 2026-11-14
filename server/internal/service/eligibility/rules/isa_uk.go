package rules

import (
	"context"
	"strings"

	"upstonk/internal/domain"
)

// ISAUKRules: UK Stocks & Shares ISA.
// Law: Individual Savings Account Regulations 1998 (SI 1998/1870) reg 7,
// https://www.legislation.gov.uk/uksi/1998/1870/regulation/7, and Income Tax
// Act 2007 s1005. Retail distribution: FCA DISC 2A.3.1R / PS25/20.
// Research and citations: DOCS/research/etf-eligibility-isa-ira-401k.md §1.
// Provider-dependent constraints follow ADR 0001.
type ISAUKRules struct{ version string }

func NewISAUKRules() *ISAUKRules { return &ISAUKRules{version: "isa_uk_v1.0_2026"} }

func (r *ISAUKRules) Name() string    { return "ISA_UK" }
func (r *ISAUKRules) Version() string { return r.version }

func (r *ISAUKRules) AppliesTo(country, accountType string) bool {
	return strings.EqualFold(country, "GB") && strings.EqualFold(accountType, "isa")
}

func (r *ISAUKRules) Evaluate(_ context.Context, etf domain.ETF) domain.EligibilityResult {
	b := newResult(r.version)
	r.checkQualifyingInvestment(b, etf)
	r.checkRetailDisclosure(b, etf)
	r.checkLeverageRestrictions(b, etf)
	return b.finish()
}

// checkQualifyingInvestment: reg 7(2) lets an ETF qualify as a UK UCITS, a
// recognised UCITS, a qualifying NURS, or as shares officially listed on a
// recognised stock exchange.
func (r *ISAUKRules) checkQualifyingInvestment(b *resultBuilder, etf domain.ETF) {
	const criterion = "isa_qualifying_investment"
	const expected = "UK/recognised UCITS, qualifying NURS, or listed on a recognised stock exchange"

	structure := strings.ToUpper(etf.LegalStructure)
	isUCITS := strings.Contains(structure, "UCITS") && !strings.Contains(structure, "NON-UCITS")
	isNURS := strings.Contains(structure, "NURS") || strings.Contains(structure, "NON-UCITS RETAIL")
	listed := classifyListing(etf.Exchange)

	switch {
	case isUCITS && strings.EqualFold(etf.Domicile, "GB"):
		b.skip(criterion, expected, "UK UCITS",
			"UK UCITS: qualifies if FCA-authorised; confirm on the FCA register", false)
	case isUCITS:
		b.skip(criterion, expected, "UCITS domiciled in "+orUnknown(etf.Domicile),
			"Overseas UCITS qualifies only if FCA-recognised; transitional cover for pre-October-2024 recognition ends 31 Dec 2026 (reg 7(2D))", false)
	case isNURS:
		b.skip(criterion, expected, "non-UCITS retail scheme",
			"NURS qualifies only if units can be redeemed at least every two months", false)
	case listed == listingRecognised:
		b.skip(criterion, expected, "listed on "+etf.Exchange+" (recognised stock exchange)",
			"Listed on a recognised stock exchange; HMRC guidance doesn't confirm that US-registered ETF shares count as \"shares issued by a company\" (reg 7(2)(a))", false)
	case listed == listingSegmentAmbiguous:
		b.skip(criterion, expected, "quoted on "+etf.Exchange+" (segment unknown)",
			"Quoted on "+etf.Exchange+"; only some of its market segments count as a recognised listing", false)
	case etf.LegalStructure != "" && listed == listingNotRecognised:
		b.fail(criterion, expected, etf.LegalStructure+" quoted only on "+etf.Exchange,
			"Not a UCITS or NURS and only quoted on "+etf.Exchange+", which is not a recognised stock exchange")
	default:
		b.skip(criterion, expected, "structure and listing not reported",
			"Neither legal structure nor a recognised listing is known, so no qualifying route can be confirmed", true)
	}
}

func orUnknown(s string) string {
	if s == "" {
		return "an unreported country"
	}
	return s
}

// checkRetailDisclosure: UK firms may not sell a fund to a retail investor
// without a UK product summary (or, until 7 June 2027, a KID). No US fund is
// an FCA-recognised scheme, so US ETFs are generally unavailable to retail
// ISA investors. This is a rule on the firm, not ISA law.
func (r *ISAUKRules) checkRetailDisclosure(b *resultBuilder, etf domain.ETF) {
	if strings.EqualFold(etf.Domicile, "US") || isUSListing(etf.Exchange) {
		b.providerWarning("US ETFs normally have no UK product summary or KID, so UK platforms can't sell them to retail investors (FCA DISC 2A.3.1R)")
	}
}

// checkLeverageRestrictions: ISA law doesn't exclude leveraged or inverse
// ETFs, but many ISA managers don't offer them.
func (r *ISAUKRules) checkLeverageRestrictions(b *resultBuilder, etf domain.ETF) {
	if domain.IsTrue(etf.IsLeveraged) || domain.IsTrue(etf.IsInverse) {
		b.providerWarning("Leveraged and inverse ETFs are allowed in an ISA by law, but many ISA managers don't offer them")
	}
}
