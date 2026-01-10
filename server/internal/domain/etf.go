package domain

import "time"

// ETF represents a discovered Exchange-Traded Fund
type ETF struct {
	// Identity
	Ticker          string `json:"ticker"`
	Name            string `json:"name"`
	ISIN            string `json:"isin"`
	Exchange        string `json:"exchange"`
	ExchangeCountry string `json:"exchangeCountry"`

	// Structure
	Domicile          string `json:"domicile"`
	LegalStructure    string `json:"legalStructure"` // e.g., "UCITS", "Section 12J", "Unit Trust"
	IsPhysical        bool   `json:"isPhysical"`
	IsSynthetic       bool   `json:"isSynthetic"`
	IsLeveraged       bool   `json:"isLeveraged"`
	IsInverse         bool   `json:"isInverse"`
	ReplicationMethod string `json:"replicationMethod"` // "Physical Full", "Physical Sampling", "Synthetic Swap"

	// Exposure
	AssetClass         string             `json:"assetClass"`
	TrackingIndex      string             `json:"trackingIndex"`
	AssetExposure      AssetExposure      `json:"assetExposure"`
	GeographicExposure GeographicExposure `json:"geographicExposure"`
	SectorExposure     []SectorAllocation `json:"sectorExposure"`
	TopHoldings        []Holding          `json:"topHoldings"`

	// Costs & Performance
	TER                float64 `json:"ter"`                          // Total Expense Ratio (%)
	TrackingDifference float64 `json:"trackingDifference,omitempty"` // Annualized (%)
	AUM                float64 `json:"aum"`                          // Assets Under Management (base currency)
	Currency           string  `json:"currency"`
	DividendTreatment  string  `json:"dividendTreatment"` // "Distributing", "Accumulating"

	// Liquidity
	AverageDailyVolume float64 `json:"averageDailyVolume"`
	BidAskSpread       float64 `json:"bidAskSpread,omitempty"` // Percentage

	// Metadata
	InceptionDate time.Time `json:"inceptionDate"`
	Provider      string    `json:"provider"` // e.g., "Satrix", "CoreShares", "Vanguard"

	// Source attribution
	DataSources []DataSource `json:"dataSources"`
	LastUpdated time.Time    `json:"lastUpdated"`
}

// AssetExposure describes what the ETF invests in
type AssetExposure struct {
	Equities    float64 `json:"equities"` // Percentage
	Bonds       float64 `json:"bonds"`
	Cash        float64 `json:"cash"`
	Commodities float64 `json:"commodities"`
	RealEstate  float64 `json:"realEstate"`
	Other       float64 `json:"other"`
}

// GeographicExposure describes regional allocation
type GeographicExposure struct {
	Regions   map[string]float64 `json:"regions"`             // Region name -> percentage
	Countries map[string]float64 `json:"countries,omitempty"` // Country code -> percentage
}

// SectorAllocation represents sector weights
type SectorAllocation struct {
	Sector     string  `json:"sector"`
	Percentage float64 `json:"percentage"`
}

// Holding represents a top position
type Holding struct {
	Name      string  `json:"name"`
	Ticker    string  `json:"ticker,omitempty"`
	Weight    float64 `json:"weight"` // Percentage
	AssetType string  `json:"assetType,omitempty"`
}

// DataSource tracks where information came from
type DataSource struct {
	Type        string    `json:"type"` // "FactSheet", "ExchangeListing", "API", "Manual"
	Provider    string    `json:"provider"`
	URL         string    `json:"url,omitempty"`
	AccessDate  time.Time `json:"accessDate"`
	Reliability string    `json:"reliability"` // "Primary", "Secondary", "Tertiary"
}

// EligibilityResult captures eligibility determination
type EligibilityResult struct {
	IsEligible   bool                  `json:"isEligible"`
	Status       EligibilityStatus     `json:"status"`
	Reasons      []string              `json:"reasons"`
	RulesPassed  []string              `json:"rulesPassed"`
	RulesFailed  []string              `json:"rulesFailed"`
	RulesSkipped []string              `json:"rulesSkipped,omitempty"`
	Confidence   ConfidenceLevel       `json:"confidence"`
	Evidence     []EligibilityEvidence `json:"evidence"`
	RuleVersion  string                `json:"ruleVersion"` // e.g., "tfsa_za_v1.2"
	EvaluatedAt  time.Time             `json:"evaluatedAt"`
}

type EligibilityStatus string

const (
	StatusEligible    EligibilityStatus = "eligible"
	StatusIneligible  EligibilityStatus = "ineligible"
	StatusUnknown     EligibilityStatus = "unknown"
	StatusConditional EligibilityStatus = "conditional" // Eligible with warnings
)

type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"   // All data from primary sources
	ConfidenceMedium ConfidenceLevel = "medium" // Mix of primary/secondary
	ConfidenceLow    ConfidenceLevel = "low"    // Missing key data points
	ConfidenceNone   ConfidenceLevel = "none"   // Insufficient data
)

// EligibilityEvidence provides audit trail
type EligibilityEvidence struct {
	Criterion  string     `json:"criterion"`
	Expected   string     `json:"expected"`
	Actual     string     `json:"actual"`
	Result     string     `json:"result"` // "pass", "fail", "unknown"
	DataSource DataSource `json:"dataSource,omitempty"`
}

// RankingScore represents weighted scoring
type RankingScore struct {
	TotalScore      float64            `json:"totalScore"` // 0-100
	ComponentScores map[string]float64 `json:"componentScores"`
	Rank            int                `json:"rank"`
	Explanation     string             `json:"explanation"`
}

// DiscoveredETF combines ETF data with eligibility and ranking
type DiscoveredETF struct {
	ETF         ETF               `json:"etf"`
	Eligibility EligibilityResult `json:"eligibility"`
	Ranking     RankingScore      `json:"ranking"`
	MatchScore  float64           `json:"matchScore"` // How well it matches requested exposure (0-100)
}
