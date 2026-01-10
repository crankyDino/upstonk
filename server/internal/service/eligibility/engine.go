package eligibility

import (
	"context"
	"fmt"
	"upstonk/internal/domain"
)

type DefaultEngine struct {
	rules map[string]Rule
}

func NewEngine() Engine {
	return &DefaultEngine{
		rules: make(map[string]Rule),
	}
}

func (e *DefaultEngine) RegisterRule(rule Rule) {
	key := fmt.Sprintf("%s_%s", rule.Name(), rule.Version())
	e.rules[key] = rule
}

func (e *DefaultEngine) Evaluate(ctx context.Context, etf domain.ETF, country, accountType string) domain.EligibilityResult {
	// Find applicable rule
	for _, rule := range e.rules {
		if rule.AppliesTo(country, accountType) {
			return rule.Evaluate(ctx, etf)
		}
	}

	// No rule found - return unknown status
	return domain.EligibilityResult{
		Status:      domain.StatusUnknown,
		IsEligible:  false,
		Confidence:  domain.ConfidenceNone,
		Reasons:     []string{"No eligibility rules available for this country/account type combination"},
		RulesPassed: []string{},
		RulesFailed: []string{},
	}
}
