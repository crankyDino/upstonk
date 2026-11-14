// Maps UI form state onto the server's DiscoveryRequest. Type-only imports so
// it runs under plain Node (see scripts/write-contract-fixture.ts).
import type {
  Constraints,
  DiscoveryRequest,
  ExposureConfig,
  InvestorProfile,
  OutputOptions,
  RankingComponent,
  RankingPreference,
} from '@/types/api';

export interface DiscoveryFormState {
  investorProfile: InvestorProfile;
  exposure: ExposureConfig;
  constraints: Constraints;
  rankingPreferences: RankingPreference[];
  outputOptions: OutputOptions;
}

// Server rejects weightings that don't sum to 1.0, so normalise. With all
// weights at 0, fall back to an even split.
function toWeighting(prefs: RankingPreference[]): Record<RankingComponent, number> {
  const components: RankingComponent[] = ['fees', 'liquidity', 'tracking', 'stability'];
  const raw = Object.fromEntries(
    components.map((c) => [c, Math.max(0, prefs.find((p) => p.id === c)?.weight ?? 0)]),
  ) as Record<RankingComponent, number>;
  const total = components.reduce((sum, c) => sum + raw[c], 0);
  const round = (n: number) => Math.round(n * 10_000) / 10_000;
  return Object.fromEntries(
    components.map((c) => [c, round(total > 0 ? raw[c] / total : 1 / components.length)]),
  ) as Record<RankingComponent, number>;
}

export function buildDiscoveryRequest(state: DiscoveryFormState): DiscoveryRequest {
  const { investorProfile: p, exposure: e, constraints: c, outputOptions: o } = state;
  return {
    investorProfile: {
      country: p.country,
      accountType: p.accountType,
      currency: p.currency,
      riskTolerance: p.riskTolerance,
      timeHorizonYears: p.timeHorizon,
    },
    exposure: {
      assets: {
        companies: e.companies,
        sectors: e.sectors,
        assetClasses: e.assetClasses,
      },
      geography: {
        markets: e.geography.markets,
        emergingMarkets: e.geography.includeEmerging,
        developedMarkets: e.geography.includeDeveloped,
      },
    },
    investmentVehicles: ['etf'],
    constraints: {
      tfsaEligibleOnly: c.tfsaEligibleOnly,
      maxTER: c.maxTER,
      minAUM: c.minAUM,
      minLiquidity: c.minDailyVolume,
      excludeSyntheticETFs: c.excludeSynthetic,
      excludeLeveragedETFs: c.excludeLeveraged,
      excludeInverseETFs: c.excludeInverse,
      physicalOnly: c.physicalReplicationOnly,
    },
    rankingPreferences: { weighting: toWeighting(state.rankingPreferences) },
    outputOptions: {
      maxResults: o.maxResults,
      includeAlternatives: o.includeAlternatives,
      includeSourceLinks: o.includeSourceLinks,
      explainEligibility: o.explainEligibility,
      includeWarnings: o.includeWarnings,
    },
  };
}
