// Writes the request the GUI sends with its default form state, and every
// country/account pair the GUI offers, to the server's router testdata so Go
// tests can prove the server accepts them.
// Run after changing the request builder or defaults:
//   npm run contract:fixture
import { writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { buildDiscoveryRequest } from '../src/lib/api/buildRequest.ts';
import {
  defaultConstraints,
  defaultExposure,
  defaultInvestorProfile,
  defaultOutputOptions,
  defaultRankingPreferences,
  SUPPORTED_ACCOUNTS,
} from '../src/lib/discoveryOptions.ts';

const request = buildDiscoveryRequest({
  investorProfile: defaultInvestorProfile,
  exposure: defaultExposure,
  constraints: defaultConstraints,
  rankingPreferences: defaultRankingPreferences,
  outputOptions: defaultOutputOptions,
});

const testdata = new URL('../../server/internal/api/router/testdata/', import.meta.url);

function write(name: string, data: unknown) {
  const out = fileURLToPath(new URL(name, testdata));
  writeFileSync(out, JSON.stringify(data, null, 2) + '\n');
  console.log(`wrote ${out}`);
}

write('gui_default_request.json', request);

// Every country/account pair the GUI lets a user pick.
write(
  'gui_account_pairs.json',
  Object.entries(SUPPORTED_ACCOUNTS).flatMap(([country, accounts]) =>
    accounts.map((a) => ({ country, accountType: a.value })),
  ),
);
