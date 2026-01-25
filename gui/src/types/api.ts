// UPSTONK API Types

export type AccountType = 'TFSA' | 'ISA' | 'IRA' | 'standard' | 'retirement_annuity';
export type RiskTolerance = 'conservative' | 'moderate' | 'aggressive';
export type EligibilityStatus = 'eligible' | 'ineligible' | 'unknown' | 'conditional';
export type ConfidenceLevel = 'high' | 'medium' | 'low' | 'unknown';

// API Request Types (matching backend DTO structure)
export interface InvestorProfileRequest {
  country: string;
  accountType: AccountType;
  currency: string;
}

export interface ExposureRequest {
  assets: {
    assetClasses: string[];
  };
  geography: {
    markets: string[];
  };
}

export interface ConstraintsRequest {
  // Empty object for now, can be extended
}

export interface OutputOptionsRequest {
  maxResults: number;
  includeSourceLinks: boolean;
}

export interface DiscoveryRequest {
  investorProfile: InvestorProfileRequest;
  exposure: ExposureRequest;
  investmentVehicles: string[];
  constraints: ConstraintsRequest;
  outputOptions: OutputOptionsRequest;
}

// UI Form State Types (richer than API request)
export interface InvestorProfile {
  country: string;
  accountType: AccountType;
  currency: string;
  riskTolerance: RiskTolerance;
  timeHorizon: number;
}

export interface ExposureConfig {
  assetClasses: string[];
  sectors: string[];
  companies: string[];
  indices: string[];
  geography: {
    markets: string[];
    includeEmerging: boolean;
    includeDeveloped: boolean;
    excludedCountries: string[];
  };
}

export interface Constraints {
  tfsaEligibleOnly: boolean;
  maxTER: number;
  minAUM: number;
  liquidityThreshold: number;
  excludeSynthetic: boolean;
  excludeLeveraged: boolean;
  excludeInverse: boolean;
  physicalReplicationOnly: boolean;
}

export interface RankingPreference {
  id: string;
  name: string;
  weight: number;
  priority: number;
}

export interface OutputOptions {
  maxResults: number;
  includeAlternatives: boolean;
  includeSourceLinks: boolean;
  explainEligibility: boolean;
  includeWarnings: boolean;
}

export interface EligibilityInfo {
  status: EligibilityStatus;
  isEligible: boolean;
  confidence: ConfidenceLevel;
  justification: string;
  ruleVersion?: string;
  rulesPassed?: string[];
  rulesFailed?: string[];
}

export interface DataSource {
  type: string;
  provider: string;
  url: string;
  date: string;
}

export interface Holding {
  name: string;
  ticker: string;
  weight: number;
}

export interface AssetBreakdown {
  equities: number;
  bonds: number;
  cash: number;
  commodities: number;
  other: number;
}

export interface GeographicBreakdown {
  regions: Record<string, number> | null;
}

export interface ETFResult {
  rank: number;
  ticker: string;
  name: string;
  isin: string;
  exchange: string;
  provider: string;
  assetClass: string;
  trackingIndex: string;
  geographicFocus: string;
  ter: number;
  aum: number;
  currency: string;
  averageDailyVolume: number;
  eligibility: EligibilityInfo;
  matchScore: number;
  rankingScore: number;
  assetBreakdown: AssetBreakdown;
  geographicBreakdown: GeographicBreakdown;
  topHoldings: Holding[];
  dataSources: DataSource[];
}

export interface APIWarning {
  code: string;
  message: string;
  severity: 'info' | 'warning' | 'error';
}

export interface DiscoverySummary {
  totalSearched: number;
  totalEligible: number;
  totalIneligible: number;
  totalUnknown: number;
  searchDurationMs: number;
  dataSourcesQueried: string[];
}

export interface DiscoveryResponse {
  requestId: string;
  results: ETFResult[];
  summary: DiscoverySummary;
  warnings: APIWarning[];
  generatedAt: string;
  cacheHit: boolean;
}

export interface APIError {
  code: string;
  message: string;
  requestId?: string;
  details?: Record<string, unknown>;
}
