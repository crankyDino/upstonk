package dto

// DiscoveryRequest represents the API payload
type DiscoveryRequest struct {
	InvestorProfile    InvestorProfile    `json:"investorProfile" validate:"required"`
	Exposure           ExposureRequest    `json:"exposure" validate:"required"`
	InvestmentVehicles []string           `json:"investmentVehicles" validate:"required,min=1"`
	Constraints        Constraints        `json:"constraints"`
	RankingPreferences RankingPreferences `json:"rankingPreferences"`
	OutputOptions      OutputOptions      `json:"outputOptions"`
}

type InvestorProfile struct {
	Country          string `json:"country" validate:"required,iso3166_1_alpha2"`
	AccountType      string `json:"accountType" validate:"required"`
	Currency         string `json:"currency" validate:"required,iso4217"`
	RiskTolerance    string `json:"riskTolerance" validate:"omitempty,oneof=conservative moderate aggressive"`
	TimeHorizonYears int    `json:"timeHorizonYears" validate:"omitempty,min=1,max=50"`
}

type ExposureRequest struct {
	Assets    AssetExposureRequest     `json:"assets"`
	Geography GeographyExposureRequest `json:"geography"`
}

type AssetExposureRequest struct {
	Companies    []string `json:"companies"`
	Sectors      []string `json:"sectors"`
	AssetClasses []string `json:"assetClasses"`
	Indices      []string `json:"indices,omitempty"`
}

type GeographyExposureRequest struct {
	Markets          []string `json:"markets"`
	EmergingMarkets  bool     `json:"emergingMarkets"`
	DevelopedMarkets bool     `json:"developedMarkets"`
	ExcludeCountries []string `json:"excludeCountries,omitempty"`
}

type Constraints struct {
	TFSAEligibleOnly     bool     `json:"tfsaEligibleOnly"`
	AllowedExchanges     []string `json:"allowedExchanges,omitempty"` // If empty, allows all exchanges
	MaxTER               float64  `json:"maxTER" validate:"omitempty,min=0,max=5"`
	MinAUM               float64  `json:"minAUM,omitempty"` // Minimum assets under management
	ExcludeSyntheticETFs bool     `json:"excludeSyntheticETFs"`
	ExcludeLeveragedETFs bool     `json:"excludeLeveragedETFs"`
	ExcludeInverseETFs   bool     `json:"excludeInverseETFs"`
	PhysicalOnly         bool     `json:"physicalOnly"`
	MinLiquidity         float64  `json:"minLiquidity,omitempty"` // Minimum avg daily volume
}

type RankingPreferences struct {
	Priority  []string           `json:"priority" validate:"omitempty,dive,oneof=lowest_fees tracking_accuracy liquidity diversification tax_efficiency"`
	Weighting map[string]float64 `json:"weighting"`
}

type OutputOptions struct {
	MaxResults          int  `json:"maxResults" validate:"min=1,max=100"`
	IncludeAlternatives bool `json:"includeAlternatives"`
	IncludeSourceLinks  bool `json:"includeSourceLinks"`
	ExplainEligibility  bool `json:"explainEligibility"`
	IncludeWarnings     bool `json:"includeWarnings"`
}

// DiscoveryResponse is the API output
type DiscoveryResponse struct {
	RequestID    string        `json:"requestId"`
	Results      []ETFResult   `json:"results"`
	Alternatives []ETFResult   `json:"alternatives,omitempty"`
	Summary      SearchSummary `json:"summary"`
	Warnings     []Warning     `json:"warnings,omitempty"`
	GeneratedAt  string        `json:"generatedAt"`
	CacheHit     bool          `json:"cacheHit"`
}

type ETFResult struct {
	// Core ETF data
	Ticker   string `json:"ticker"`
	Name     string `json:"name"`
	ISIN     string `json:"isin"`
	Exchange string `json:"exchange"`
	Provider string `json:"provider"`

	// Exposure summary
	AssetClass      string `json:"assetClass"`
	TrackingIndex   string `json:"trackingIndex"`
	GeographicFocus string `json:"geographicFocus"`
	SectorFocus     string `json:"sectorFocus,omitempty"`

	// Key metrics
	TER                float64 `json:"ter"`
	AUM                float64 `json:"aum"`
	Currency           string  `json:"currency"`
	AverageDailyVolume float64 `json:"averageDailyVolume"`

	// Eligibility
	Eligibility EligibilityDetail `json:"eligibility"`

	// Scoring
	MatchScore   float64 `json:"matchScore"`
	RankingScore float64 `json:"rankingScore"`
	Rank         int     `json:"rank"`

	// Breakdown (optional based on OutputOptions)
	AssetBreakdown      *AssetBreakdown      `json:"assetBreakdown,omitempty"`
	GeographicBreakdown *GeographicBreakdown `json:"geographicBreakdown,omitempty"`
	TopHoldings         []HoldingInfo        `json:"topHoldings,omitempty"`

	// Sources
	DataSources []SourceReference `json:"dataSources,omitempty"`
}

type EligibilityDetail struct {
	Status        string   `json:"status"` // "eligible", "ineligible", "unknown", "conditional"
	IsEligible    bool     `json:"isEligible"`
	Confidence    string   `json:"confidence"`
	Justification string   `json:"justification"`
	RulesPassed   []string `json:"rulesPassed,omitempty"`
	RulesFailed   []string `json:"rulesFailed,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	RuleVersion   string   `json:"ruleVersion,omitempty"`
}

type AssetBreakdown struct {
	Equities    float64 `json:"equities"`
	Bonds       float64 `json:"bonds"`
	Cash        float64 `json:"cash"`
	Commodities float64 `json:"commodities"`
	Other       float64 `json:"other"`
}

type GeographicBreakdown struct {
	Regions   map[string]float64 `json:"regions"`
	Countries map[string]float64 `json:"countries,omitempty"`
}

type HoldingInfo struct {
	Name   string  `json:"name"`
	Ticker string  `json:"ticker,omitempty"`
	Weight float64 `json:"weight"`
}

type SourceReference struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	URL      string `json:"url,omitempty"`
	Date     string `json:"date"`
}

type SearchSummary struct {
	TotalSearched      int      `json:"totalSearched"`
	TotalEligible      int      `json:"totalEligible"`
	TotalIneligible    int      `json:"totalIneligible"`
	TotalUnknown       int      `json:"totalUnknown"`
	SearchDurationMs   int64    `json:"searchDurationMs"`
	DataSourcesQueried []string `json:"dataSourcesQueried"`
}

type Warning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "info", "warning", "critical"
}

// ErrorResponse for error cases
type ErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	Code      string            `json:"code"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"requestId"`
}
