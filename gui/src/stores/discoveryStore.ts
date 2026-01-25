import { create } from 'zustand';
import type { 
  DiscoveryRequest, 
  DiscoveryResponse, 
  InvestorProfile, 
  ExposureConfig, 
  Constraints, 
  RankingPreference,
  OutputOptions,
  APIError 
} from '@/types/api';
import { discoverETFs } from '@/lib/api/upstonk';

interface DiscoveryState {
  // Form state
  investorProfile: InvestorProfile;
  exposure: ExposureConfig;
  constraints: Constraints;
  rankingPreferences: RankingPreference[];
  outputOptions: OutputOptions;
  
  // UI state
  isLoading: boolean;
  error: APIError | null;
  response: DiscoveryResponse | null;
  selectedETFTicker: string | null;
  activeStep: number;
  
  // Actions
  setInvestorProfile: (profile: Partial<InvestorProfile>) => void;
  setExposure: (exposure: Partial<ExposureConfig>) => void;
  setConstraints: (constraints: Partial<Constraints>) => void;
  setRankingPreferences: (preferences: RankingPreference[]) => void;
  setOutputOptions: (options: Partial<OutputOptions>) => void;
  setActiveStep: (step: number) => void;
  setSelectedETF: (ticker: string | null) => void;
  submitDiscovery: () => Promise<void>;
  clearResults: () => void;
  resetForm: () => void;
}

const defaultInvestorProfile: InvestorProfile = {
  country: 'ZA',
  accountType: 'standard',
  currency: 'ZAR',
  riskTolerance: 'moderate',
  timeHorizon: 10,
};

const defaultExposure: ExposureConfig = {
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

const defaultConstraints: Constraints = {
  tfsaEligibleOnly: true,
  maxTER: 0.50,
  minAUM: 100000000,
  liquidityThreshold: 80,
  excludeSynthetic: true,
  excludeLeveraged: true,
  excludeInverse: true,
  physicalReplicationOnly: false,
};

const defaultRankingPreferences: RankingPreference[] = [
  { id: 'ter', name: 'Expense Ratio (TER)', weight: 0.25, priority: 1 },
  { id: 'liquidity', name: 'Liquidity', weight: 0.20, priority: 2 },
  { id: 'aum', name: 'Assets Under Management', weight: 0.20, priority: 3 },
  { id: 'tracking', name: 'Tracking Accuracy', weight: 0.20, priority: 4 },
  { id: 'eligibility', name: 'Eligibility Confidence', weight: 0.15, priority: 5 },
];

const defaultOutputOptions: OutputOptions = {
  maxResults: 10,
  includeAlternatives: true,
  includeSourceLinks: true,
  explainEligibility: true,
  includeWarnings: true,
};

export const useDiscoveryStore = create<DiscoveryState>((set, get) => ({
  investorProfile: defaultInvestorProfile,
  exposure: defaultExposure,
  constraints: defaultConstraints,
  rankingPreferences: defaultRankingPreferences,
  outputOptions: defaultOutputOptions,
  isLoading: false,
  error: null,
  response: null,
  selectedETFTicker: null,
  activeStep: 0,
  
  setInvestorProfile: (profile) => 
    set((state) => ({ 
      investorProfile: { ...state.investorProfile, ...profile } 
    })),
    
  setExposure: (exposure) => 
    set((state) => ({ 
      exposure: { ...state.exposure, ...exposure } 
    })),
    
  setConstraints: (constraints) => 
    set((state) => ({ 
      constraints: { ...state.constraints, ...constraints } 
    })),
    
  setRankingPreferences: (preferences) => 
    set({ rankingPreferences: preferences }),
    
  setOutputOptions: (options) => 
    set((state) => ({ 
      outputOptions: { ...state.outputOptions, ...options } 
    })),
    
  setActiveStep: (step) => set({ activeStep: step }),
  
  setSelectedETF: (ticker) => set({ selectedETFTicker: ticker }),
  
  submitDiscovery: async () => {
    const state = get();
    
    // Transform UI state to API request format
    const request: DiscoveryRequest = {
      investorProfile: {
        country: state.investorProfile.country,
        accountType: state.investorProfile.accountType,
        currency: state.investorProfile.currency,
      },
      exposure: {
        assets: {
          assetClasses: state.exposure.assetClasses,
        },
        geography: {
          markets: state.exposure.geography.markets,
        },
      },
      investmentVehicles: ['etf'],
      constraints: {},
      outputOptions: {
        maxResults: state.outputOptions.maxResults,
        includeSourceLinks: state.outputOptions.includeSourceLinks,
      },
    };
    
    set({ isLoading: true, error: null });
    
    try {
      const response = await discoverETFs(request);
      set({ response, isLoading: false });
    } catch (error) {
      set({ error: error as APIError, isLoading: false });
    }
  },
  
  clearResults: () => set({ response: null, error: null }),
  
  resetForm: () => set({
    investorProfile: defaultInvestorProfile,
    exposure: defaultExposure,
    constraints: defaultConstraints,
    rankingPreferences: defaultRankingPreferences,
    outputOptions: defaultOutputOptions,
    response: null,
    error: null,
    activeStep: 0,
  }),
}));
