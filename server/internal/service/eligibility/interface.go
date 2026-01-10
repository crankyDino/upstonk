package eligibility

import (
	"context"
	"upstonk/internal/domain"
)

// Engine evaluates ETF eligibility based on rules
type Engine interface {
	Evaluate(ctx context.Context, etf domain.ETF, country, accountType string) domain.EligibilityResult
	RegisterRule(rule Rule)
}

// Rule represents a specific eligibility ruleset
type Rule interface {
	Name() string
	Version() string
	AppliesTo(country, accountType string) bool
	Evaluate(ctx context.Context, etf domain.ETF) domain.EligibilityResult
}
