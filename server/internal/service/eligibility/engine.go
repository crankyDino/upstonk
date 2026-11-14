package eligibility

import (
	"context"
	"fmt"
	"strings"
	"upstonk/internal/domain"
)

// DefaultEngine evaluates an ETF against the first registered rule that
// applies. Rules are checked in registration order so the outcome is
// deterministic when rules overlap.
type DefaultEngine struct {
	rules []Rule
	index map[string]int // Name_Version -> position in rules
}

func NewEngine() Engine {
	return &DefaultEngine{index: make(map[string]int)}
}

// RegisterRule adds a rule. Registering the same name and version again
// replaces the earlier rule in place.
func (e *DefaultEngine) RegisterRule(rule Rule) {
	key := fmt.Sprintf("%s_%s", rule.Name(), rule.Version())
	if i, ok := e.index[key]; ok {
		e.rules[i] = rule
		return
	}
	e.index[key] = len(e.rules)
	e.rules = append(e.rules, rule)
}

func (e *DefaultEngine) Evaluate(ctx context.Context, etf domain.ETF, country, accountType string) domain.EligibilityResult {
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
