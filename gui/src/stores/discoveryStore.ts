import { create } from 'zustand';
import type {
  DiscoveryResponse,
  InvestorProfile,
  ExposureConfig,
  Constraints,
  RankingPreference,
  OutputOptions,
  APIError
} from '@/types/api';
import { discoverETFs } from '@/lib/api/upstonk';
import { buildDiscoveryRequest } from '@/lib/api/buildRequest';
import {
  SUPPORTED_ACCOUNTS,
  COUNTRIES,
  defaultInvestorProfile,
  defaultExposure,
  defaultConstraints,
  defaultRankingPreferences,
  defaultOutputOptions,
} from '@/lib/discoveryOptions';

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
  
  // Changing country resets the account type if the new country doesn't
  // offer it, and switches to that country's currency.
  setInvestorProfile: (profile) =>
    set((state) => {
      const next = { ...state.investorProfile, ...profile };
      if (profile.country && profile.country !== state.investorProfile.country) {
        const offered = SUPPORTED_ACCOUNTS[next.country].map((a) => a.value);
        if (!offered.includes(next.accountType)) next.accountType = 'standard';
        if (!profile.currency) {
          next.currency = COUNTRIES.find((c) => c.code === next.country)?.currency ?? next.currency;
        }
      }
      return { investorProfile: next };
    }),

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
    
    const request = buildDiscoveryRequest(state);

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
