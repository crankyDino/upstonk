package discovery

import (
	"context"
	"errors"
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
	candidates, cacheHit, err := s.searchCandidates(ctx, req)
	if err != nil {
		var outage *search.SourcesFailedError
		if errors.As(err, &outage) {
			return nil, &DataSourceError{Source: "all", Err: outage}
		}
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
	filtered, unverifiedStructure := s.applyConstraints(evaluatedETFs, req.Constraints)

	// Step 4: Calculate match scores (how well each ETF matches requested exposure)
	scored := s.calculateMatchScores(filtered, req.Exposure)

	// Step 5: Rank using weighted scoring
	ranked := s.rankETFs(scored, req.RankingPreferences)

	// Step 6: Build output
	results, alternatives := s.buildOutput(ranked, req.OutputOptions)

	// Step 7: Generate warnings
	warnings := s.generateWarnings(results, req, unverifiedStructure)

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
		CacheHit: cacheHit,
	}, nil
}

// searchCandidates runs the search and reports whether it was served from
// cache (only providers implementing search.CacheReporter can say so).
func (s *Service) searchCandidates(ctx context.Context, req dto.DiscoveryRequest) ([]domain.ETF, bool, error) {
	searchCriteria := search.Criteria{
		Markets:      req.Exposure.Geography.Markets,
		Sectors:      req.Exposure.Assets.Sectors,
		AssetClasses: req.Exposure.Assets.AssetClasses,
		Companies:    req.Exposure.Assets.Companies,
		Country:      req.InvestorProfile.Country,
		Vehicles:     req.InvestmentVehicles,
	}

	if cr, ok := s.searchService.(search.CacheReporter); ok {
		return cr.SearchCached(ctx, searchCriteria)
	}
	etfs, err := s.searchService.Search(ctx, searchCriteria)
	return etfs, false, err
}

func (s *Service) evaluateEligibility(
	ctx context.Context,
	etfs []domain.ETF,
	profile dto.InvestorProfile,
) ([]domain.DiscoveredETF, EligibilitySummary) {

	summary := EligibilitySummary{
		TotalSearched:      len(etfs),
		DataSourcesQueried: contributingSources(etfs),
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

// applyConstraints drops ETFs that fail a constraint on known data. It also
// returns the tickers kept despite a structure constraint because the flag it
// checks is unknown, so they can be flagged rather than silently passed.
func (s *Service) applyConstraints(etfs []domain.DiscoveredETF, constraints dto.Constraints) ([]domain.DiscoveredETF, map[string]bool) {
	filtered := make([]domain.DiscoveredETF, 0)
	unverifiedStructure := make(map[string]bool)

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

		// AUM constraint. AUM 0 means the source didn't report it: keep the
		// ETF (never guess) and let generateWarnings flag it as unverified.
		if constraints.MinAUM > 0 && etf.AUM > 0 && etf.AUM < constraints.MinAUM {
			continue
		}

		// Synthetic exclusion
		if constraints.ExcludeSyntheticETFs && domain.IsTrue(etf.IsSynthetic) {
			continue
		}

		// Leveraged exclusion
		if constraints.ExcludeLeveragedETFs && domain.IsTrue(etf.IsLeveraged) {
			continue
		}

		// Inverse exclusion
		if constraints.ExcludeInverseETFs && domain.IsTrue(etf.IsInverse) {
			continue
		}

		// Physical only
		if constraints.PhysicalOnly && domain.IsFalse(etf.IsPhysical) {
			continue
		}

		// Structure constraints only drop on known flags; unknown ones are flagged.
		if (constraints.ExcludeSyntheticETFs && etf.IsSynthetic == nil) ||
			(constraints.ExcludeLeveragedETFs && etf.IsLeveraged == nil) ||
			(constraints.ExcludeInverseETFs && etf.IsInverse == nil) ||
			(constraints.PhysicalOnly && etf.IsPhysical == nil) {
			unverifiedStructure[etf.Ticker] = true
		}

		// Liquidity constraint
		if constraints.MinLiquidity > 0 && etf.AverageDailyVolume < constraints.MinLiquidity {
			continue
		}

		filtered = append(filtered, discovered)
	}

	return filtered, unverifiedStructure
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
	score := 0.0
	for _, market := range geo.Markets {
		for region, exposure := range etf.GeographicExposure.Regions {
			if strings.EqualFold(region, market) {
				score += exposure / 100.0
			}
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

func (s *Service) generateWarnings(results []dto.ETFResult, req dto.DiscoveryRequest, unverifiedStructure map[string]bool) []dto.Warning {
	warnings := []dto.Warning{}

	if len(results) == 0 {
		warnings = append(warnings, dto.Warning{
			Code:     "NO_ELIGIBLE_RESULTS",
			Message:  "No eligible ETFs found. Consider relaxing constraints or broadening exposure criteria.",
			Severity: "warning",
		})
	}

	// Results kept despite a constraint because the value it checks is
	// unknown (0 = the source didn't report it).
	if req.Constraints.MinAUM > 0 {
		warnings = appendUnverified(warnings, results, "UNVERIFIED_AUM", "Fund size (AUM)", "minimum AUM",
			func(r dto.ETFResult) bool { return r.AUM == 0 })
	}
	if req.Constraints.MaxTER > 0 {
		warnings = appendUnverified(warnings, results, "UNVERIFIED_TER", "Total expense ratio (TER)", "maximum TER",
			func(r dto.ETFResult) bool { return r.TER == 0 })
	}
	warnings = appendUnverified(warnings, results, "UNVERIFIED_STRUCTURE", "Fund structure (synthetic, leveraged, inverse or physical)", "structure",
		func(r dto.ETFResult) bool { return unverifiedStructure[r.Ticker] })

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

// appendUnverified adds a warning naming every result whose value for a
// constraint is unknown, so the constraint could not be checked for it.
func appendUnverified(warnings []dto.Warning, results []dto.ETFResult, code, field, constraint string, unknown func(dto.ETFResult) bool) []dto.Warning {
	tickers := []string{}
	for _, r := range results {
		if unknown(r) {
			tickers = append(tickers, r.Ticker)
		}
	}
	if len(tickers) == 0 {
		return warnings
	}
	return append(warnings, dto.Warning{
		Code: code,
		Message: fmt.Sprintf("%s is unavailable for %s, so the %s constraint could not be checked for them.",
			field, strings.Join(tickers, ", "), constraint),
		Severity: "warning",
	})
}

// contributingSources lists, sorted and de-duplicated, the data source
// providers recorded in the candidates' provenance.
func contributingSources(etfs []domain.ETF) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, etf := range etfs {
		for _, ds := range etf.DataSources {
			if ds.Provider != "" && !seen[ds.Provider] {
				seen[ds.Provider] = true
				out = append(out, ds.Provider)
			}
		}
	}
	sort.Strings(out)
	return out
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
	return containsFold(actual, requested)
}

// containsFold reports whether substr appears in s, ignoring case. Empty
// strings never match.
func containsFold(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func matchesAssetClass(requested, actual string) bool {
	return containsFold(actual, requested)
}

func matchesCompany(requested, actual string) bool {
	return containsFold(actual, requested)
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
