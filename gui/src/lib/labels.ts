// Human-readable titles for the codes the API returns. The codes stay in the
// API (they're stable identifiers); only the GUI translates them. Unknown
// codes fall back to plain words, so nothing ever shows as SHOUTY_SNAKE_CASE.

const warningTitles: Record<string, string> = {
  UNVERIFIED_TER: 'Fees not confirmed',
  UNVERIFIED_AUM: 'Fund size not confirmed',
  UNVERIFIED_STRUCTURE: 'Fund structure not confirmed',
  LOW_CONFIDENCE_RESULTS: 'Double-check these results',
  NO_ELIGIBLE_RESULTS: 'No eligible funds found',
};

const errorTitles: Record<string, string> = {
  NO_RESULTS: 'No matching funds',
  DATA_SOURCE_ERROR: 'Market data is unavailable right now',
  VALIDATION_ERROR: 'Some search settings need fixing',
  INVALID_REQUEST: 'This search can’t be run',
  INVALID_JSON: 'The search couldn’t be sent',
  INVALID_TYPE: 'Unknown search type',
  UNAUTHORIZED: 'Please sign in again',
  NOT_INVITED: 'You need an invite',
  ACCOUNT_DISABLED: 'Your account is disabled',
  ADMIN_ONLY: 'Admins only',
  LAST_ADMIN: 'At least one admin is needed',
  INTERNAL_ERROR: 'Something went wrong',
  API_ERROR: 'Something went wrong',
};

const ruleLabels: Record<string, string> = {
  standard_account: 'Standard account',
  jse_listing: 'Listed on the JSE',
  jse_tfsa_list: 'On the JSE tax-free (TFSA) list',
  implicit_sars_approval: 'Qualifies for a tax-free account',
  no_leverage: 'Not leveraged',
  no_inverse: 'Not inverse',
  replication_method: 'Replication method',
  currency_denomination: 'Currency',
  approved_provider: 'Offered by your provider',
  isa_qualifying_investment: 'ISA-qualifying investment',
  plan_menu: 'On your plan’s fund menu',
  roth_ira: 'Roth IRA',
};

// "SOME_CODE" / "some_code" → "Some code"
export function humanize(code: string): string {
  const words = code.replace(/[_-]+/g, ' ').trim().toLowerCase();
  return words ? words[0].toUpperCase() + words.slice(1) : code;
}

export const warningTitle = (code: string) => warningTitles[code] ?? humanize(code);
export const errorTitle = (code?: string) => (code && errorTitles[code]) || (code ? humanize(code) : 'Something went wrong');
export const ruleLabel = (id: string) => ruleLabels[id] ?? humanize(id);
