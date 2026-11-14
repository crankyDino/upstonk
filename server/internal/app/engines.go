// Package app holds the engine wiring shared by the server binary and tests,
// so both always run the same eligibility rules and ranking engine.
package app

import (
	"upstonk/internal/service/eligibility"
	"upstonk/internal/service/eligibility/rules"
	"upstonk/internal/service/ranking"
)

// NewEligibilityEngine returns an engine with every production rule registered.
func NewEligibilityEngine() eligibility.Engine {
	engine := eligibility.NewEngine()
	engine.RegisterRule(rules.NewTFSASouthAfricaRules())
	return engine
}

// NewRankingEngine returns the production ranking engine.
func NewRankingEngine() ranking.Engine {
	return ranking.NewWeightedScorer()
}
