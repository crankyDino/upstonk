// service/search/interface.go
package search

import (
	"context"
	"upstonk/internal/domain"
)

// Provider defines the contract for ETF search implementations
type Provider interface {
	Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error)
}

type Criteria struct {
	Markets      []string
	Sectors      []string
	AssetClasses []string
	Companies    []string
	Country      string
	Vehicles     []string
}
