package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"upstonk/internal/api/dto"
	"upstonk/internal/domain"
	"upstonk/internal/service/eligibility"
	"upstonk/internal/service/ranking"
	"upstonk/internal/service/search"
)

// Service orchestrates ETF discovery workflow
type Service struct {
	searchService     search.Provider
	eligibilityEngine eligibility.Engine
	rankingEngine     ranking.Engine
	cacheEnabled      bool
}

func NewService(
	searchProvider search.Provider,
	eligibilityEngine eligibility.Engine,
	rankingEngine ranking.Engine,
) *Service {
	return &Service{
		searchService:     searchProvider,
		eligibilityEngine: eligibilityEngine,
		rankingEngine:     rankingEngine,
		cacheEnabled:      true,
	}
}

type DiscoveryResult struct {
	Results      []dto.ETFResult
	Alternatives []dto.ETFResult
	Summary      dto.SearchSummary
	Warnings     []dto.Warning
	CacheHit     bool
}

// DiscoverETFs is the main workflow orchestrator
func (s *Service) DiscoverETFs(ctx context.Context, req dto.DiscoveryRequest) (*DiscoveryResult, error) {
	startTime := time.Now()

	// Step 1: Search for candidate ETFs
	candidates, err := s.searchCandidates(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(candidates) == 0 {
		return nil, &NoResultsError{
			Message: "No ETFs found matching the requested exposure criteria",
		}
	}

	// Step 2: Evaluate eligibility for each candidate
	evaluatedETFs, summary := s.evaluateEligibility(ctx, candidates, req.InvestorProfile)

	// Step 3: Filter based on constraints
	filtered := s.applyConstraints(evaluatedETFs, req.Constraints)

	// Step 4: Calculate match scores (how well each ETF matches requested exposure)
	scored := s.calculateMatchScores(filtered, req.Exposure)

	// Step 5: Rank using weighted scoring
	ranked := s.rankETFs(scored, req.RankingPreferences)

	// Step 6: Build output
	results, alternatives := s.buildOutput(ranked, req.OutputOptions)

	// Step 7: Generate warnings
	warnings := s.generateWarnings(results, req)

	searchDuration := time.Since(startTime).Milliseconds()

	return &DiscoveryResult{
		Results:      results,
		Alternatives: alternatives,
		Summary: dto.SearchSummary{
			TotalSearched:      summary.TotalSearched,
			TotalEligible:      summary.TotalEligible,
			TotalIneligible:    summary.TotalIneligible,
			TotalUnknown:       summary.TotalUnknown,
			SearchDurationMs:   searchDuration,
			DataSourcesQueried: summary.DataSourcesQueried,
		},
		Warnings: warnings,
		CacheHit: false,
	}, nil
}

func (s *Service) searchCandidates(ctx context.Context, req dto.DiscoveryRequest) ([]domain.ETF, error) {
	searchCriteria := search.Criteria{
		Markets:      req.Exposure.Geography.Markets,
		Sectors:      req.Exposure.Assets.Sectors,
		AssetClasses: req.Exposure.Assets.AssetClasses,
		Companies:    req.Exposure.Assets.Companies,
		Country:      req.InvestorProfile.Country,
		Vehicles:     req.InvestmentVehicles,
	}

	return s.searchService.Search(ctx, searchCriteria)
}

func (s *Service) evaluateEligibility(
	ctx context.Context,
	etfs []domain.ETF,
	profile dto.InvestorProfile,
) ([]domain.DiscoveredETF, EligibilitySummary) {

	summary := EligibilitySummary{
		TotalSearched:      len(etfs),
		DataSourcesQueried: []string{"JSE", "FactSheets", "Provider APIs"},
	}

	discovered := make([]domain.DiscoveredETF, 0, len(etfs))

	for _, etf := range etfs {
		eligibility := s.eligibilityEngine.Evaluate(ctx, etf, profile.Country, profile.AccountType)

		discovered = append(discovered, domain.DiscoveredETF{
			ETF:         etf,
			Eligibility: eligibility,
		})

		// Update summary counts
		switch eligibility.Status {
		case domain.StatusEligible, domain.StatusConditional:
			summary.TotalEligible++
		case domain.StatusIneligible:
			summary.TotalIneligible++
		case domain.StatusUnknown:
			summary.TotalUnknown++
		}
	}

	return discovered, summary
}

func (s *Service) applyConstraints(etfs []domain.DiscoveredETF, constraints dto.Constraints) []domain.DiscoveredETF {
	filtered := make([]domain.DiscoveredETF, 0)

	// Build exchange allowlist map for fast lookup
	allowedExchanges := make(map[string]bool)
	for _, exchange := range constraints.AllowedExchanges {
		allowedExchanges[strings.ToUpper(exchange)] = true
	}
	hasExchangeConstraint := len(constraints.AllowedExchanges) > 0

	for _, discovered := range etfs {
		etf := discovered.ETF

		// Skip ineligible if TFSA-only requested
		// If tfsaEligibleOnly is false, allow eligible, conditional, and unknown ETFs
		// Only filter out ineligible ETFs when tfsaEligibleOnly is true
		if constraints.TFSAEligibleOnly {
			if !discovered.Eligibility.IsEligible {
				continue
			}
		} else {
			// When tfsaEligibleOnly is false, only exclude explicitly ineligible ETFs
			// Allow eligible, conditional, and unknown
			if discovered.Eligibility.Status == domain.StatusIneligible {
				continue
			}
		}

		// Exchange constraint: if specified, only allow listed exchanges
		if hasExchangeConstraint {
			etfExchange := strings.ToUpper(etf.Exchange)
			if !allowedExchanges[etfExchange] {
				continue
			}
		}

		// TER constraint
		if constraints.MaxTER > 0 && etf.TER > constraints.MaxTER {
			continue
		}

		// AUM constraint
		if constraints.MinAUM > 0 && etf.AUM < constraints.MinAUM {
			continue
		}

		// Synthetic exclusion
		if constraints.ExcludeSyntheticETFs && etf.IsSynthetic {
			continue
		}

		// Leveraged exclusion
		if constraints.ExcludeLeveragedETFs && etf.IsLeveraged {
			continue
		}

		// Inverse exclusion
		if constraints.ExcludeInverseETFs && etf.IsInverse {
			continue
		}

		// Physical only
		if constraints.PhysicalOnly && !etf.IsPhysical {
			continue
		}

		// Liquidity constraint
		if constraints.MinLiquidity > 0 && etf.AverageDailyVolume < constraints.MinLiquidity {
			continue
		}

		filtered = append(filtered, discovered)
	}

	return filtered
}

func (s *Service) calculateMatchScores(etfs []domain.DiscoveredETF, exposure dto.ExposureRequest) []domain.DiscoveredETF {
	for i := range etfs {
		score := 0.0
		maxScore := 0.0

		// Score geographic match
		if len(exposure.Geography.Markets) > 0 {
			maxScore += 30.0
			geoScore := s.calculateGeographicMatch(etfs[i].ETF, exposure.Geography)
			score += geoScore * 30.0
		}

		// Score sector match
		if len(exposure.Assets.Sectors) > 0 {
			maxScore += 25.0
			sectorScore := s.calculateSectorMatch(etfs[i].ETF, exposure.Assets.Sectors)
			score += sectorScore * 25.0
		}

		// Score asset class match
		if len(exposure.Assets.AssetClasses) > 0 {
			maxScore += 25.0
			assetScore := s.calculateAssetClassMatch(etfs[i].ETF, exposure.Assets.AssetClasses)
			score += assetScore * 25.0
		}

		// Score company holdings match
		if len(exposure.Assets.Companies) > 0 {
			maxScore += 20.0
			companyScore := s.calculateCompanyMatch(etfs[i].ETF, exposure.Assets.Companies)
			score += companyScore * 20.0
		}

		// Normalize to 0-100
		if maxScore > 0 {
			etfs[i].MatchScore = (score / maxScore) * 100.0
		} else {
			etfs[i].MatchScore = 50.0 // Default if no criteria
		}
	}

	return etfs
}

func (s *Service) calculateGeographicMatch(etf domain.ETF, geo dto.GeographyExposureRequest) float64 {
	// Simplified matching logic
	score := 0.0
	for _, market := range geo.Markets {
		if exposure, exists := etf.GeographicExposure.Regions[market]; exists {
			score += exposure / 100.0
		}
	}
	return min(score/float64(len(geo.Markets)), 1.0)
}

func (s *Service) calculateSectorMatch(etf domain.ETF, sectors []string) float64 {
	score := 0.0
	for _, requestedSector := range sectors {
		for _, etfSector := range etf.SectorExposure {
			if matchesSector(requestedSector, etfSector.Sector) {
				score += etfSector.Percentage / 100.0
			}
		}
	}
	return min(score/float64(len(sectors)), 1.0)
}

func (s *Service) calculateAssetClassMatch(etf domain.ETF, assetClasses []string) float64 {
	// Simple asset class matching
	hasMatch := false
	for _, requested := range assetClasses {
		if matchesAssetClass(requested, etf.AssetClass) {
			hasMatch = true
			break
		}
	}
	if hasMatch {
		return 1.0
	}
	return 0.0
}

func (s *Service) calculateCompanyMatch(etf domain.ETF, companies []string) float64 {
	matchedWeight := 0.0
	for _, company := range companies {
		for _, holding := range etf.TopHoldings {
			if matchesCompany(company, holding.Name) {
				matchedWeight += holding.Weight
			}
		}
	}
	return min(matchedWeight/100.0, 1.0)
}

func (s *Service) rankETFs(etfs []domain.DiscoveredETF, preferences dto.RankingPreferences) []domain.DiscoveredETF {
	// Use ranking engine for weighted scoring
	for i := range etfs {
		rankingScore := s.rankingEngine.Score(etfs[i].ETF, preferences)
		etfs[i].Ranking = rankingScore
	}

	// Sort by combined score (match + ranking)
	sort.Slice(etfs, func(i, j int) bool {
		scoreI := etfs[i].MatchScore*0.4 + etfs[i].Ranking.TotalScore*0.6
		scoreJ := etfs[j].MatchScore*0.4 + etfs[j].Ranking.TotalScore*0.6
		return scoreI > scoreJ
	})

	// Assign ranks
	for i := range etfs {
		etfs[i].Ranking.Rank = i + 1
	}

	return etfs
}

func (s *Service) buildOutput(ranked []domain.DiscoveredETF, options dto.OutputOptions) ([]dto.ETFResult, []dto.ETFResult) {
	results := make([]dto.ETFResult, 0)
	alternatives := make([]dto.ETFResult, 0)

	for i, discovered := range ranked {
		result := s.toETFResult(discovered, options)

		if i < options.MaxResults && discovered.Eligibility.IsEligible {
			results = append(results, result)
		} else if options.IncludeAlternatives && len(alternatives) < 5 {
			alternatives = append(alternatives, result)
		}
	}

	return results, alternatives
}

func (s *Service) toETFResult(discovered domain.DiscoveredETF, options dto.OutputOptions) dto.ETFResult {
	etf := discovered.ETF

	result := dto.ETFResult{
		Ticker:             etf.Ticker,
		Name:               etf.Name,
		ISIN:               etf.ISIN,
		Exchange:           etf.Exchange,
		Provider:           etf.Provider,
		AssetClass:         etf.AssetClass,
		TrackingIndex:      etf.TrackingIndex,
		TER:                etf.TER,
		AUM:                etf.AUM,
		Currency:           etf.Currency,
		AverageDailyVolume: etf.AverageDailyVolume,
		MatchScore:         discovered.MatchScore,
		RankingScore:       discovered.Ranking.TotalScore,
		Rank:               discovered.Ranking.Rank,
		Eligibility: dto.EligibilityDetail{
			Status:        string(discovered.Eligibility.Status),
			IsEligible:    discovered.Eligibility.IsEligible,
			Confidence:    string(discovered.Eligibility.Confidence),
			Justification: formatJustification(discovered.Eligibility),
			RuleVersion:   discovered.Eligibility.RuleVersion,
		},
	}

	// Add detailed breakdowns if requested
	if options.ExplainEligibility {
		result.Eligibility.RulesPassed = discovered.Eligibility.RulesPassed
		result.Eligibility.RulesFailed = discovered.Eligibility.RulesFailed
		result.Eligibility.Warnings = extractWarnings(discovered.Eligibility)
	}

	// Add holdings and breakdowns
	result.AssetBreakdown = &dto.AssetBreakdown{
		Equities:    etf.AssetExposure.Equities,
		Bonds:       etf.AssetExposure.Bonds,
		Cash:        etf.AssetExposure.Cash,
		Commodities: etf.AssetExposure.Commodities,
		Other:       etf.AssetExposure.Other,
	}

	result.GeographicBreakdown = &dto.GeographicBreakdown{
		Regions:   etf.GeographicExposure.Regions,
		Countries: etf.GeographicExposure.Countries,
	}

	// Top holdings
	for _, holding := range etf.TopHoldings {
		result.TopHoldings = append(result.TopHoldings, dto.HoldingInfo{
			Name:   holding.Name,
			Ticker: holding.Ticker,
			Weight: holding.Weight,
		})
	}

	// Data sources
	if options.IncludeSourceLinks {
		for _, ds := range etf.DataSources {
			result.DataSources = append(result.DataSources, dto.SourceReference{
				Type:     ds.Type,
				Provider: ds.Provider,
				URL:      ds.URL,
				Date:     ds.AccessDate.Format("2006-01-02"),
			})
		}
	}

	return result
}

func (s *Service) generateWarnings(results []dto.ETFResult, req dto.DiscoveryRequest) []dto.Warning {
	warnings := []dto.Warning{}

	if len(results) == 0 {
		warnings = append(warnings, dto.Warning{
			Code:     "NO_ELIGIBLE_RESULTS",
			Message:  "No eligible ETFs found. Consider relaxing constraints or broadening exposure criteria.",
			Severity: "warning",
		})
	}

	// Check for low confidence results
	lowConfidenceCount := 0
	for _, result := range results {
		if result.Eligibility.Confidence == "low" || result.Eligibility.Confidence == "medium" {
			lowConfidenceCount++
		}
	}

	if lowConfidenceCount > 0 {
		warnings = append(warnings, dto.Warning{
			Code: "LOW_CONFIDENCE_RESULTS",
			Message: fmt.Sprintf("%d results have medium or low confidence. Verify eligibility with your platform before investing.",
				lowConfidenceCount),
			Severity: "info",
		})
	}

	return warnings
}

// Helper functions
func formatJustification(eligibility domain.EligibilityResult) string {
	if len(eligibility.Reasons) == 0 {
		return "Eligibility could not be determined"
	}
	justification := ""
	for _, reason := range eligibility.Reasons {
		justification += reason + "; "
	}
	return justification
}

func extractWarnings(eligibility domain.EligibilityResult) []string {
	warnings := []string{}

	for _, reason := range eligibility.Reasons {
		if strings.HasPrefix(reason, "⚠") {
			warnings = append(warnings, reason)
		}
	}

	return warnings
}

func matchesSector(requested, actual string) bool {
	// Simplified matching - in production, use taxonomy mapping
	return contains(actual, requested)
}

func matchesAssetClass(requested, actual string) bool {
	return contains(actual, requested)
}

func matchesCompany(requested, actual string) bool {
	return contains(actual, requested)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
			fmt.Sprintf("%s", s) == fmt.Sprintf("%s", substr))
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Error types
type NoResultsError struct {
	Message string
}

func (e *NoResultsError) Error() string {
	return e.Message
}

type UnsupportedCountryError struct {
	Country string
}

func (e *UnsupportedCountryError) Error() string {
	return fmt.Sprintf("country '%s' is not supported", e.Country)
}

type DataSourceError struct {
	Source string
	Err    error
}

func (e *DataSourceError) Error() string {
	return fmt.Sprintf("data source '%s' error: %v", e.Source, e.Err)
}

type EligibilitySummary struct {
	TotalSearched      int
	TotalEligible      int
	TotalIneligible    int
	TotalUnknown       int
	DataSourcesQueried []string
}
