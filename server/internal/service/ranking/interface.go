package ranking

import (
	"upstonk/internal/api/dto"
	"upstonk/internal/domain"
)

// Engine scores and ranks ETFs
type Engine interface {
	Score(etf domain.ETF, preferences dto.RankingPreferences) domain.RankingScore
}
