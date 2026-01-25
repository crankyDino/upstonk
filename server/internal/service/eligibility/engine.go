package eligibility

import (
	"context"
	"fmt"
	"strings"
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

	// No rule found - for standard accounts, default to eligible
	// For tax-advantaged accounts, return unknown to be safe
	if strings.ToLower(accountType) == "standard" {
		return domain.EligibilityResult{
			Status:      domain.StatusEligible,
			IsEligible:  true,
			Confidence:  domain.ConfidenceMedium,
			Reasons:     []string{"Standard account - no specific eligibility restrictions"},
			RulesPassed: []string{"standard_account"},
			RulesFailed: []string{},
		}
	}

	// For tax-advantaged accounts without rules, return unknown
	return domain.EligibilityResult{
		Status:      domain.StatusUnknown,
		IsEligible:  false,
		Confidence:  domain.ConfidenceNone,
		Reasons:     []string{"No eligibility rules available for this country/account type combination"},
		RulesPassed: []string{},
		RulesFailed: []string{},
	}
}
