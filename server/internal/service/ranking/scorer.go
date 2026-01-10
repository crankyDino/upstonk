package ranking

import (
	"upstonk/internal/api/dto"
	"upstonk/internal/domain"
)

type WeightedScorer struct{}

func NewWeightedScorer() Engine {
	return &WeightedScorer{}
}

func (s *WeightedScorer) Score(etf domain.ETF, preferences dto.RankingPreferences) domain.RankingScore {
	scores := make(map[string]float64)

	// Fee score (inverse - lower is better)
	scores["fees"] = s.scoreFees(etf.TER)

	// Liquidity score
	scores["liquidity"] = s.scoreLiquidity(etf.AverageDailyVolume)

	// Size/stability score
	scores["stability"] = s.scoreAUM(etf.AUM)

	// Tracking score
	scores["tracking"] = s.scoreTracking(etf.TrackingDifference)

	// Apply weights
	weights := preferences.Weighting
	if len(weights) == 0 {
		// Default weights
		weights = map[string]float64{
			"fees":      0.4,
			"liquidity": 0.3,
			"tracking":  0.2,
			"stability": 0.1,
		}
	}

	totalScore := 0.0
	for component, score := range scores {
		if weight, exists := weights[component]; exists {
			totalScore += score * weight
		}
	}

	return domain.RankingScore{
		TotalScore:      totalScore * 100, // Scale to 0-100
		ComponentScores: scores,
		Explanation:     "Weighted score based on fees, liquidity, tracking, and stability",
	}
}

func (s *WeightedScorer) scoreFees(ter float64) float64 {
	// Lower TER = higher score
	if ter >= 1.0 {
		return 0.0
	}
	return 1.0 - ter
}

func (s *WeightedScorer) scoreLiquidity(avgVolume float64) float64 {
	// Logarithmic scoring for volume
	if avgVolume < 10000 {
		return 0.3
	} else if avgVolume < 100000 {
		return 0.6
	} else if avgVolume < 500000 {
		return 0.8
	}
	return 1.0
}

func (s *WeightedScorer) scoreAUM(aum float64) float64 {
	// Larger AUM = more stable
	if aum < 50000000 {
		return 0.3
	} else if aum < 500000000 {
		return 0.6
	} else if aum < 1000000000 {
		return 0.8
	}
	return 1.0
}

func (s *WeightedScorer) scoreTracking(trackingDiff float64) float64 {
	// Lower tracking difference = better
	if trackingDiff == 0 {
		return 0.7 // Unknown, neutral score
	}
	if trackingDiff < 0.1 {
		return 1.0
	} else if trackingDiff < 0.3 {
		return 0.8
	} else if trackingDiff < 0.5 {
		return 0.6
	}
	return 0.4
}
