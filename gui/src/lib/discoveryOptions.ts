// Form options and defaults for the discovery flow. Kept free of React and
// runtime imports (type-only) so scripts/write-contract-fixture.ts can run it
// under plain Node.
import type {
  AccountType,
  Constraints,
  Country,
  ExposureConfig,
  InvestorProfile,
  OutputOptions,
  RankingPreference,
} from '@/types/api';

export interface AccountOption {
  value: AccountType;
  label: string;
  description: string;
}

// Country/account pairs the server accepts
// (isAccountTypeSupported in server/internal/api/handlers/discovery.go).
export const SUPPORTED_ACCOUNTS: Record<Country, AccountOption[]> = {
  ZA: [
    { value: 'tfsa', label: 'TFSA', description: 'Tax-Free Savings Account (South Africa)' },
    { value: 'standard', label: 'Standard', description: 'Taxable brokerage account' },
  ],
  US: [
    { value: 'ira', label: 'IRA', description: 'Individual Retirement Account' },
    { value: 'roth_ira', label: 'Roth IRA', description: 'Roth Individual Retirement Account' },
    { value: '401k', label: '401(k)', description: 'Employer retirement plan' },
    { value: 'standard', label: 'Standard', description: 'Taxable brokerage account' },
  ],
  GB: [
    { value: 'isa', label: 'ISA', description: 'Individual Savings Account' },
    { value: 'standard', label: 'Standard', description: 'Taxable brokerage account' },
  ],
};

export const COUNTRIES: { code: Country; name: string; currency: string }[] = [
  { code: 'ZA', name: 'South Africa', currency: 'ZAR' },
  { code: 'US', name: 'United States', currency: 'USD' },
  { code: 'GB', name: 'United Kingdom', currency: 'GBP' },
];

// Minimum average daily volume steps; they match the liquidity scorer's
// thresholds (server/internal/service/ranking/scorer.go). 0 = no minimum.
export const LIQUIDITY_STEPS = [0, 10_000, 100_000, 500_000] as const;

export const defaultInvestorProfile: InvestorProfile = {
  country: 'ZA',
  accountType: 'standard',
  currency: 'ZAR',
  riskTolerance: 'moderate',
  timeHorizon: 10,
};

export const defaultExposure: ExposureConfig = {
  assetClasses: ['equity'],
  sectors: [],
  companies: [],
  indices: [],
  geography: {
    markets: ['US', 'CA'],
    includeEmerging: false,
    includeDeveloped: true,
    excludedCountries: [],
  },
};

export const defaultConstraints: Constraints = {
  tfsaEligibleOnly: true,
  maxTER: 0.5,
  minAUM: 100_000_000,
  minDailyVolume: 0,
  excludeSynthetic: true,
  excludeLeveraged: true,
  excludeInverse: true,
  physicalReplicationOnly: false,
};

// Same split as the server's default weights.
export const defaultRankingPreferences: RankingPreference[] = [
  { id: 'fees', name: 'Fees (TER)', weight: 0.4, priority: 1 },
  { id: 'liquidity', name: 'Liquidity', weight: 0.3, priority: 2 },
  { id: 'tracking', name: 'Tracking Accuracy', weight: 0.2, priority: 3 },
  { id: 'stability', name: 'Fund Size (AUM)', weight: 0.1, priority: 4 },
];

export const defaultOutputOptions: OutputOptions = {
  maxResults: 10,
  includeAlternatives: true,
  includeSourceLinks: true,
  explainEligibility: true,
  includeWarnings: true,
};
