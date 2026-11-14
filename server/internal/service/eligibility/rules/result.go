package rules

import (
	"time"

	"upstonk/internal/domain"
)

// resultBuilder accumulates criteria for rules that follow ADR 0001
// (DOCS/architecture/adr/0001-provider-dependent-eligibility.md): legal
// criteria pass, fail or are skipped; provider-dependent constraints only add
// warnings, which cap the outcome at conditional.
type resultBuilder struct {
	r                domain.EligibilityResult
	providerWarnings int
}

func newResult(version string) *resultBuilder {
	return &resultBuilder{r: domain.EligibilityResult{
		RuleVersion:  version,
		EvaluatedAt:  time.Now(),
		Confidence:   domain.ConfidenceHigh,
		RulesPassed:  []string{},
		RulesFailed:  []string{},
		RulesSkipped: []string{},
		Evidence:     []domain.EligibilityEvidence{},
		Reasons:      []string{},
	}}
}

func (b *resultBuilder) pass(criterion, reason string) {
	b.r.RulesPassed = append(b.r.RulesPassed, criterion)
	b.r.Reasons = append(b.r.Reasons, "✓ "+reason)
}

func (b *resultBuilder) fail(criterion, expected, actual, reason string) {
	b.r.RulesFailed = append(b.r.RulesFailed, criterion)
	b.r.Reasons = append(b.r.Reasons, "✗ "+reason)
	b.r.Evidence = append(b.r.Evidence, domain.EligibilityEvidence{
		Criterion: criterion, Expected: expected, Actual: actual, Result: "fail",
	})
}

// skip records a legal criterion that can't be decided from the data.
// lowConfidence marks data so thin the verdict should become unknown.
func (b *resultBuilder) skip(criterion, expected, actual, reason string, lowConfidence bool) {
	b.r.RulesSkipped = append(b.r.RulesSkipped, criterion)
	b.r.Reasons = append(b.r.Reasons, "⚠ "+reason)
	b.r.Evidence = append(b.r.Evidence, domain.EligibilityEvidence{
		Criterion: criterion, Expected: expected, Actual: actual, Result: "unknown",
	})
	switch {
	case lowConfidence:
		b.r.Confidence = domain.ConfidenceLow
	case b.r.Confidence == domain.ConfidenceHigh:
		b.r.Confidence = domain.ConfidenceMedium
	}
}

// providerWarning records a constraint set by the account provider or plan
// rather than the law. It never fails the ETF.
func (b *resultBuilder) providerWarning(reason string) {
	b.providerWarnings++
	b.r.Reasons = append(b.r.Reasons, "⚠ "+reason)
}

func (b *resultBuilder) finish() domain.EligibilityResult {
	switch {
	case len(b.r.RulesFailed) > 0:
		b.r.Status, b.r.IsEligible = domain.StatusIneligible, false
	case len(b.r.RulesSkipped) > 3 || b.r.Confidence == domain.ConfidenceLow:
		b.r.Status, b.r.IsEligible = domain.StatusUnknown, false
	case len(b.r.RulesSkipped) > 0 || b.providerWarnings > 0:
		b.r.Status, b.r.IsEligible = domain.StatusConditional, true
	default:
		b.r.Status, b.r.IsEligible = domain.StatusEligible, true
	}
	if b.r.Status == domain.StatusConditional && b.r.Confidence == domain.ConfidenceHigh {
		b.r.Confidence = domain.ConfidenceMedium
	}
	return b.r
}
