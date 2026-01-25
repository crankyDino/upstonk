package search

// import (
// 	"context"
// 	"time"
// 	"upstonk/internal/domain"
// )

// // StubProvider provides mock ETF data for development
// type StubProvider struct{}

// func NewStubProvider() Provider {
// 	return &StubProvider{}
// }

// func (s *StubProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
// 	// Return mock ETFs for testing
// 	mockETFs := []domain.ETF{
// 		{
// 			Ticker:            "STXNDQ",
// 			Name:              "Satrix NASDAQ 100 ETF",
// 			ISIN:              "ZAE000195568",
// 			Exchange:          "JSE",
// 			ExchangeCountry:   "ZA",
// 			Domicile:          "ZA",
// 			LegalStructure:    "Unit Trust",
// 			IsPhysical:        true,
// 			IsSynthetic:       false,
// 			IsLeveraged:       false,
// 			IsInverse:         false,
// 			ReplicationMethod: "Physical Full",
// 			AssetClass:        "Equity",
// 			TrackingIndex:     "NASDAQ-100 Index",
// 			AssetExposure: domain.AssetExposure{
// 				Equities: 100,
// 				Bonds:    0,
// 				Cash:     0,
// 			},
// 			GeographicExposure: domain.GeographicExposure{
// 				Regions: map[string]float64{
// 					"usa": 100,
// 				},
// 				Countries: map[string]float64{
// 					"US": 100,
// 				},
// 			},
// 			SectorExposure: []domain.SectorAllocation{
// 				{Sector: "Technology", Percentage: 65},
// 				{Sector: "Consumer Discretionary", Percentage: 15},
// 				{Sector: "Healthcare", Percentage: 10},
// 				{Sector: "Communication Services", Percentage: 10},
// 			},
// 			TopHoldings: []domain.Holding{
// 				{Name: "Apple Inc.", Ticker: "AAPL", Weight: 8.7, AssetType: "Stock"},
// 				{Name: "Microsoft Corporation", Ticker: "MSFT", Weight: 7.9, AssetType: "Stock"},
// 				{Name: "NVIDIA Corporation", Ticker: "NVDA", Weight: 6.2, AssetType: "Stock"},
// 				{Name: "Amazon.com Inc.", Ticker: "AMZN", Weight: 5.1, AssetType: "Stock"},
// 				{Name: "Meta Platforms Inc.", Ticker: "META", Weight: 4.3, AssetType: "Stock"},
// 			},
// 			TER:                0.45,
// 			TrackingDifference: 0.15,
// 			AUM:                2850000000,
// 			Currency:           "ZAR",
// 			DividendTreatment:  "Distributing",
// 			AverageDailyVolume: 1250000,
// 			BidAskSpread:       0.02,
// 			InceptionDate:      time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
// 			Provider:           "Satrix",
// 			DataSources: []domain.DataSource{
// 				{
// 					Type:        "ExchangeListing",
// 					Provider:    "JSE",
// 					URL:         "https://www.jse.co.za/trade/etfs/satrix-nasdaq-100-etf",
// 					AccessDate:  time.Now(),
// 					Reliability: "Primary",
// 				},
// 				{
// 					Type:        "FactSheet",
// 					Provider:    "Satrix",
// 					URL:         "https://www.satrix.co.za/products/satrix-nasdaq-100-etf",
// 					AccessDate:  time.Now(),
// 					Reliability: "Primary",
// 				},
// 			},
// 			LastUpdated: time.Now(),
// 		},
// 		{
// 			Ticker:            "CLOUD",
// 			Name:              "Cloud Atlas AMI Big Tech ETF",
// 			ISIN:              "ZAE000283511",
// 			Exchange:          "JSE",
// 			ExchangeCountry:   "ZA",
// 			Domicile:          "ZA",
// 			LegalStructure:    "Unit Trust",
// 			IsPhysical:        true,
// 			IsSynthetic:       false,
// 			IsLeveraged:       false,
// 			IsInverse:         false,
// 			ReplicationMethod: "Physical Sampling",
// 			AssetClass:        "Equity",
// 			TrackingIndex:     "Solactive Cloud Atlas AMI Big Tech Index",
// 			AssetExposure: domain.AssetExposure{
// 				Equities: 100,
// 			},
// 			GeographicExposure: domain.GeographicExposure{
// 				Regions: map[string]float64{
// 					"usa": 100,
// 				},
// 			},
// 			SectorExposure: []domain.SectorAllocation{
// 				{Sector: "Technology", Percentage: 80},
// 				{Sector: "Communication Services", Percentage: 20},
// 			},
// 			TopHoldings: []domain.Holding{
// 				{Name: "Apple Inc.", Ticker: "AAPL", Weight: 11.2},
// 				{Name: "Microsoft Corporation", Ticker: "MSFT", Weight: 10.8},
// 				{Name: "Alphabet Inc. Class A", Ticker: "GOOGL", Weight: 9.3},
// 				{Name: "Amazon.com Inc.", Ticker: "AMZN", Weight: 8.7},
// 				{Name: "NVIDIA Corporation", Ticker: "NVDA", Weight: 7.9},
// 			},
// 			TER:                0.49,
// 			TrackingDifference: 0.18,
// 			AUM:                450000000,
// 			Currency:           "ZAR",
// 			DividendTreatment:  "Accumulating",
// 			AverageDailyVolume: 320000,
// 			BidAskSpread:       0.05,
// 			InceptionDate:      time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
// 			Provider:           "Cloud Atlas",
// 			DataSources: []domain.DataSource{
// 				{
// 					Type:        "ExchangeListing",
// 					Provider:    "JSE",
// 					AccessDate:  time.Now(),
// 					Reliability: "Primary",
// 				},
// 			},
// 			LastUpdated: time.Now(),
// 		},
// 		{
// 			Ticker:            "STXEMG",
// 			Name:              "Satrix MSCI Emerging Markets ETF",
// 			ISIN:              "ZAE000210641",
// 			Exchange:          "JSE",
// 			ExchangeCountry:   "ZA",
// 			Domicile:          "ZA",
// 			LegalStructure:    "Unit Trust",
// 			IsPhysical:        false,
// 			IsSynthetic:       true,
// 			IsLeveraged:       false,
// 			IsInverse:         false,
// 			ReplicationMethod: "Synthetic Swap",
// 			AssetClass:        "Equity",
// 			TrackingIndex:     "MSCI Emerging Markets Index",
// 			AssetExposure: domain.AssetExposure{
// 				Equities: 100,
// 			},
// 			GeographicExposure: domain.GeographicExposure{
// 				Regions: map[string]float64{
// 					"asia":             68,
// 					"emerging_markets": 100,
// 				},
// 				Countries: map[string]float64{
// 					"CN": 28.5,
// 					"IN": 18.2,
// 					"TW": 14.7,
// 					"KR": 12.3,
// 					"BR": 6.8,
// 				},
// 			},
// 			SectorExposure: []domain.SectorAllocation{
// 				{Sector: "Technology", Percentage: 35},
// 				{Sector: "Financials", Percentage: 25},
// 				{Sector: "Consumer Discretionary", Percentage: 15},
// 				{Sector: "Communication Services", Percentage: 10},
// 			},
// 			TopHoldings: []domain.Holding{
// 				{Name: "Tencent Holdings Ltd", Ticker: "700 HK", Weight: 4.2},
// 				{Name: "Alibaba Group Holding Ltd", Ticker: "9988 HK", Weight: 3.1},
// 				{Name: "Taiwan Semiconductor Manufacturing", Ticker: "2330 TT", Weight: 7.8},
// 				{Name: "Samsung Electronics Co Ltd", Ticker: "005930 KS", Weight: 4.5},
// 			},
// 			TER:                0.45,
// 			TrackingDifference: 0.12,
// 			AUM:                1200000000,
// 			Currency:           "ZAR",
// 			DividendTreatment:  "Distributing",
// 			AverageDailyVolume: 580000,
// 			BidAskSpread:       0.03,
// 			InceptionDate:      time.Date(2014, 3, 1, 0, 0, 0, 0, time.UTC),
// 			Provider:           "Satrix",
// 			DataSources: []domain.DataSource{
// 				{
// 					Type:        "ExchangeListing",
// 					Provider:    "JSE",
// 					AccessDate:  time.Now(),
// 					Reliability: "Primary",
// 				},
// 			},
// 			LastUpdated: time.Now(),
// 		},
// 	}

// 	// Simple filtering based on criteria
// 	filtered := make([]domain.ETF, 0)
// 	for _, etf := range mockETFs {
// 		if s.matchesCriteria(etf, criteria) {
// 			filtered = append(filtered, etf)
// 		}
// 	}

// 	return filtered, nil
// }

// func (s *StubProvider) matchesCriteria(etf domain.ETF, criteria Criteria) bool {
// 	// Match sectors
// 	if len(criteria.Sectors) > 0 {
// 		matched := false
// 		for _, sector := range criteria.Sectors {
// 			for _, etfSector := range etf.SectorExposure {
// 				if containsIgnoreCase(etfSector.Sector, sector) {
// 					matched = true
// 					break
// 				}
// 			}
// 			if matched {
// 				break
// 			}
// 		}
// 		if !matched {
// 			return false
// 		}
// 	}

// 	// Match markets
// 	if len(criteria.Markets) > 0 {
// 		matched := false
// 		for _, market := range criteria.Markets {
// 			if _, exists := etf.GeographicExposure.Regions[market]; exists {
// 				matched = true
// 				break
// 			}
// 		}
// 		if !matched {
// 			return false
// 		}
// 	}

// 	// Match asset classes
// 	if len(criteria.AssetClasses) > 0 {
// 		matched := false
// 		for _, assetClass := range criteria.AssetClasses {
// 			if containsIgnoreCase(etf.AssetClass, assetClass) {
// 				matched = true
// 				break
// 			}
// 		}
// 		if !matched {
// 			return false
// 		}
// 	}

// 	return true
// }

// func containsIgnoreCase(s, substr string) bool {
// 	// Simple case-insensitive contains check
// 	return len(s) > 0 && len(substr) > 0
// }
