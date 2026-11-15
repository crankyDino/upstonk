package discovery_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"upstonk/internal/api/dto"
	"upstonk/internal/app"
	"upstonk/internal/domain"
	"upstonk/internal/service/discovery"
	"upstonk/internal/service/eligibility"
	"upstonk/internal/service/search"
	"upstonk/internal/testutil/fakeprovider"
)

func newService(p search.Provider) *discovery.Service {
	return discovery.NewService(p, app.NewEligibilityEngine(), app.NewRankingEngine())
}

// standardRequest is a minimal valid request for a ZA standard account, with
// the given exposure.
func standardRequest(exposure dto.ExposureRequest) dto.DiscoveryRequest {
	return dto.DiscoveryRequest{
		InvestorProfile:    dto.InvestorProfile{Country: "ZA", AccountType: "standard", Currency: "ZAR"},
		Exposure:           exposure,
		InvestmentVehicles: []string{"etf"},
		OutputOptions:      dto.OutputOptions{MaxResults: 10},
	}
}

func TestNoCandidatesIsANoResultsError(t *testing.T) {
	svc := newService(fakeprovider.New())

	_, err := svc.DiscoverETFs(context.Background(), standardRequest(dto.ExposureRequest{}))

	var noResults *discovery.NoResultsError
	if !errors.As(err, &noResults) {
		t.Fatalf("err = %v, want *NoResultsError", err)
	}
}

// discover runs one request against a service backed by the given ETFs and
// fails the test on error.
func discover(t *testing.T, req dto.DiscoveryRequest, etfs ...domain.ETF) *discovery.DiscoveryResult {
	t.Helper()
	res, err := newService(fakeprovider.New(etfs...)).DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatalf("DiscoverETFs: %v", err)
	}
	return res
}

// assertScore compares scores to a hand-worked literal, allowing for
// floating-point rounding.
func assertScore(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func onlyResult(t *testing.T, res *discovery.DiscoveryResult) dto.ETFResult {
	t.Helper()
	if len(res.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(res.Results))
	}
	return res.Results[0]
}

func TestSectorMatchIgnoresCaseAndAcceptsPartialNames(t *testing.T) {
	etf := domain.ETF{
		Ticker:         "TECH",
		SectorExposure: []domain.SectorAllocation{{Sector: "Technology", Percentage: 100}},
	}

	res := discover(t, standardRequest(dto.ExposureRequest{
		Assets: dto.AssetExposureRequest{Sectors: []string{"tech"}},
	}), etf)

	assertScore(t, "matchScore", onlyResult(t, res).MatchScore, 100)
}

func TestCompanyMatchIgnoresCaseAndUsesHoldingWeight(t *testing.T) {
	etf := domain.ETF{
		Ticker: "BIG",
		TopHoldings: []domain.Holding{
			{Name: "APPLE INC", Weight: 7.0},
			{Name: "MICROSOFT CORP", Weight: 6.0},
		},
	}

	res := discover(t, standardRequest(dto.ExposureRequest{
		Assets: dto.AssetExposureRequest{Companies: []string{"apple"}},
	}), etf)

	// Apple is 7% of the fund, so the company component scores 7 out of 100.
	assertScore(t, "matchScore", onlyResult(t, res).MatchScore, 7)
}

func TestAssetClassMatchIgnoresCase(t *testing.T) {
	etf := domain.ETF{Ticker: "EQ", AssetClass: "equity"}

	res := discover(t, standardRequest(dto.ExposureRequest{
		Assets: dto.AssetExposureRequest{AssetClasses: []string{"Equity"}},
	}), etf)

	assertScore(t, "matchScore", onlyResult(t, res).MatchScore, 100)
}

func TestGeographyMatchIgnoresCaseOfRegionNames(t *testing.T) {
	etf := domain.ETF{
		Ticker:             "US",
		GeographicExposure: domain.GeographicExposure{Regions: map[string]float64{"USA": 80, "Canada": 20}},
	}

	res := discover(t, standardRequest(dto.ExposureRequest{
		Geography: dto.GeographyExposureRequest{Markets: []string{"usa"}},
	}), etf)

	// 80% of the fund is in the requested market.
	assertScore(t, "matchScore", onlyResult(t, res).MatchScore, 80)
}

func TestRepeatedRequestIsServedFromCacheAndSaysSo(t *testing.T) {
	leaf := fakeprovider.New(domain.ETF{Ticker: "ONE"})
	svc := newService(search.NewAggregatedProvider(leaf))
	req := standardRequest(dto.ExposureRequest{
		Geography: dto.GeographyExposureRequest{Markets: []string{"usa"}},
	})

	first, err := svc.DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatalf("first DiscoverETFs: %v", err)
	}
	second, err := svc.DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatalf("second DiscoverETFs: %v", err)
	}

	if first.CacheHit {
		t.Errorf("first request: CacheHit = true, want false")
	}
	if !second.CacheHit {
		t.Errorf("second request: CacheHit = false, want true")
	}
	if leaf.Calls() != 1 {
		t.Errorf("data source called %d times, want 1", leaf.Calls())
	}
}

func TestRequestsForDifferentAssetClassesAreNotServedFromEachOthersCache(t *testing.T) {
	leaf := fakeprovider.New(domain.ETF{Ticker: "ONE"})
	svc := newService(search.NewAggregatedProvider(leaf))
	equity := standardRequest(dto.ExposureRequest{Assets: dto.AssetExposureRequest{AssetClasses: []string{"equity"}}})
	bonds := standardRequest(dto.ExposureRequest{Assets: dto.AssetExposureRequest{AssetClasses: []string{"bond"}}})

	if _, err := svc.DiscoverETFs(context.Background(), equity); err != nil {
		t.Fatalf("equity: %v", err)
	}
	res, err := svc.DiscoverETFs(context.Background(), bonds)
	if err != nil {
		t.Fatalf("bonds: %v", err)
	}

	if res.CacheHit {
		t.Errorf("bond request was answered from the equity request's cache entry")
	}
	if leaf.Calls() != 2 {
		t.Errorf("data source called %d times, want 2", leaf.Calls())
	}
}

func tickers(results []dto.ETFResult) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Ticker
	}
	return out
}

func TestEachConstraintDropsTheETFThatViolatesIt(t *testing.T) {
	// ok passes every constraint used below.
	ok := domain.ETF{Ticker: "OK", TER: 0.2, AUM: 1e9, AverageDailyVolume: 200000, IsPhysical: domain.Flag(true)}

	cases := []struct {
		name        string
		constraints dto.Constraints
		violator    func(e domain.ETF) domain.ETF
	}{
		{"maxTER", dto.Constraints{MaxTER: 0.5}, func(e domain.ETF) domain.ETF { e.TER = 0.75; return e }},
		{"minAUM", dto.Constraints{MinAUM: 1e8}, func(e domain.ETF) domain.ETF { e.AUM = 5e7; return e }},
		{"minLiquidity", dto.Constraints{MinLiquidity: 100000}, func(e domain.ETF) domain.ETF { e.AverageDailyVolume = 50000; return e }},
		{"excludeSynthetic", dto.Constraints{ExcludeSyntheticETFs: true}, func(e domain.ETF) domain.ETF { e.IsSynthetic = domain.Flag(true); return e }},
		{"excludeLeveraged", dto.Constraints{ExcludeLeveragedETFs: true}, func(e domain.ETF) domain.ETF { e.IsLeveraged = domain.Flag(true); return e }},
		{"excludeInverse", dto.Constraints{ExcludeInverseETFs: true}, func(e domain.ETF) domain.ETF { e.IsInverse = domain.Flag(true); return e }},
		{"physicalOnly", dto.Constraints{PhysicalOnly: true}, func(e domain.ETF) domain.ETF { e.IsPhysical = domain.Flag(false); return e }},
		{"allowedExchanges", dto.Constraints{AllowedExchanges: []string{"jse"}}, func(e domain.ETF) domain.ETF { e.Exchange = "NYSE"; return e }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			okHere := ok
			okHere.Exchange = "JSE"
			bad := tc.violator(okHere)
			bad.Ticker = "BAD"

			req := standardRequest(dto.ExposureRequest{})
			req.Constraints = tc.constraints
			res := discover(t, req, okHere, bad)

			got := tickers(res.Results)
			if len(got) != 1 || got[0] != "OK" {
				t.Fatalf("results = %v, want [OK]", got)
			}
		})
	}
}

func TestCustomWeightingDecidesTheOrder(t *testing.T) {
	cheap := domain.ETF{Ticker: "CHEAP", TER: 0.05, AverageDailyVolume: 5000}
	liquid := domain.ETF{Ticker: "LIQUID", TER: 0.9, AverageDailyVolume: 1000000}

	cases := []struct {
		weighting map[string]float64
		wantOrder []string
		wantTop   float64 // ranking score of the top result, worked by hand
	}{
		{map[string]float64{"fees": 1}, []string{"CHEAP", "LIQUID"}, 95},
		{map[string]float64{"liquidity": 1}, []string{"LIQUID", "CHEAP"}, 100},
	}

	for _, tc := range cases {
		req := standardRequest(dto.ExposureRequest{})
		req.RankingPreferences = dto.RankingPreferences{Weighting: tc.weighting}

		res := discover(t, req, cheap, liquid)

		got := tickers(res.Results)
		if len(got) != 2 || got[0] != tc.wantOrder[0] || got[1] != tc.wantOrder[1] {
			t.Fatalf("weighting %v: order = %v, want %v", tc.weighting, got, tc.wantOrder)
		}
		assertScore(t, "top rankingScore", res.Results[0].RankingScore, tc.wantTop)
	}
}

func TestUnknownAUMPassesMinAUMButIsFlagged(t *testing.T) {
	known := domain.ETF{Ticker: "KNOWN", AUM: 2e9}
	unknown := domain.ETF{Ticker: "NOAUM"} // AUM 0 = source didn't report it

	req := standardRequest(dto.ExposureRequest{})
	req.Constraints = dto.Constraints{MinAUM: 1e8}
	res := discover(t, req, known, unknown)

	if len(res.Results) != 2 {
		t.Fatalf("results = %v, want both KNOWN and NOAUM", tickers(res.Results))
	}
	var flagged *dto.Warning
	for i := range res.Warnings {
		if res.Warnings[i].Code == "UNVERIFIED_AUM" {
			flagged = &res.Warnings[i]
		}
	}
	if flagged == nil {
		t.Fatalf("no UNVERIFIED_AUM warning in %+v", res.Warnings)
	}
	if !strings.Contains(flagged.Message, "NOAUM") || strings.Contains(flagged.Message, "KNOWN") {
		t.Fatalf("warning should name NOAUM only, got %q", flagged.Message)
	}
}

func TestUnknownTERPassesMaxTERButIsFlagged(t *testing.T) {
	known := domain.ETF{Ticker: "KNOWN", TER: 0.2}
	unknown := domain.ETF{Ticker: "NOTER"} // TER 0 = source didn't report it

	req := standardRequest(dto.ExposureRequest{})
	req.Constraints = dto.Constraints{MaxTER: 0.5}
	res := discover(t, req, known, unknown)

	if len(res.Results) != 2 {
		t.Fatalf("results = %v, want both KNOWN and NOTER", tickers(res.Results))
	}
	var flagged *dto.Warning
	for i := range res.Warnings {
		if res.Warnings[i].Code == "UNVERIFIED_TER" {
			flagged = &res.Warnings[i]
		}
	}
	if flagged == nil {
		t.Fatalf("no UNVERIFIED_TER warning in %+v", res.Warnings)
	}
	if !strings.Contains(flagged.Message, "NOTER") || strings.Contains(flagged.Message, "KNOWN") {
		t.Fatalf("warning should name NOTER only, got %q", flagged.Message)
	}
}

// namedRule is a minimal eligibility rule that applies to every request and
// stamps its name into the result, so tests can see which rule ran.
type namedRule struct{ name string }

func (r namedRule) Name() string                           { return r.name }
func (r namedRule) Version() string                        { return "v1" }
func (r namedRule) AppliesTo(country, account string) bool { return true }
func (r namedRule) Evaluate(_ context.Context, _ domain.ETF) domain.EligibilityResult {
	return domain.EligibilityResult{Status: domain.StatusEligible, IsEligible: true, RuleVersion: r.name}
}

func TestOverlappingRulesAlwaysResolveToTheFirstRegistered(t *testing.T) {
	engine := eligibility.NewEngine()
	for _, n := range []string{"FIRST", "SECOND", "THIRD", "FOURTH", "FIFTH"} {
		engine.RegisterRule(namedRule{n})
	}
	svc := discovery.NewService(fakeprovider.New(domain.ETF{Ticker: "X"}), engine, app.NewRankingEngine())

	for i := 0; i < 50; i++ {
		res, err := svc.DiscoverETFs(context.Background(), standardRequest(dto.ExposureRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if got := onlyResult(t, res).Eligibility.RuleVersion; got != "FIRST" {
			t.Fatalf("run %d: rule %q applied, want FIRST", i, got)
		}
	}
}

// satrix40 is a JSE ETF with every TFSA-relevant attribute known.
func satrix40() domain.ETF {
	return domain.ETF{
		Ticker: "STX40", Exchange: "JSE", ExchangeCountry: "ZA", Currency: "ZAR", Provider: "Satrix",
		IsPhysical: domain.Flag(true), IsSynthetic: domain.Flag(false),
		IsLeveraged: domain.Flag(false), IsInverse: domain.Flag(false),
		TFSAFriendly: domain.Flag(true),
	}
}

func tfsaRequest() dto.DiscoveryRequest {
	req := standardRequest(dto.ExposureRequest{})
	req.InvestorProfile.AccountType = "tfsa"
	req.OutputOptions.ExplainEligibility = true
	return req
}

func TestTFSAWithAllDataKnownIsEligible(t *testing.T) {
	res := discover(t, tfsaRequest(), satrix40())

	if got := onlyResult(t, res).Eligibility.Status; got != "eligible" {
		t.Fatalf("status = %q, want eligible", got)
	}
}

func TestTFSAWithUnknownLeverageAndInverseIsConditionalNotEligible(t *testing.T) {
	etf := satrix40()
	etf.IsLeveraged, etf.IsInverse = nil, nil

	res := discover(t, tfsaRequest(), etf)

	el := onlyResult(t, res).Eligibility
	if el.Status != "conditional" {
		t.Fatalf("status = %q, want conditional (two criteria unchecked)", el.Status)
	}
	for _, passed := range el.RulesPassed {
		if passed == "no_leverage" || passed == "no_inverse" {
			t.Fatalf("criterion %q passed without data; rulesPassed = %v", passed, el.RulesPassed)
		}
	}
}

func TestUnknownStructureFlagsPassTheirConstraintButAreFlagged(t *testing.T) {
	cases := []struct {
		name        string
		constraints dto.Constraints
	}{
		{"excludeSynthetic", dto.Constraints{ExcludeSyntheticETFs: true}},
		{"excludeLeveraged", dto.Constraints{ExcludeLeveragedETFs: true}},
		{"excludeInverse", dto.Constraints{ExcludeInverseETFs: true}},
		{"physicalOnly", dto.Constraints{PhysicalOnly: true}},
	}
	known := domain.ETF{Ticker: "KNOWN", IsPhysical: domain.Flag(true), IsSynthetic: domain.Flag(false),
		IsLeveraged: domain.Flag(false), IsInverse: domain.Flag(false)}
	unknown := domain.ETF{Ticker: "NOFLAGS"} // no source reported structure

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := standardRequest(dto.ExposureRequest{})
			req.Constraints = tc.constraints
			res := discover(t, req, known, unknown)

			if len(res.Results) != 2 {
				t.Fatalf("results = %v, want KNOWN and NOFLAGS", tickers(res.Results))
			}
			var flagged *dto.Warning
			for i := range res.Warnings {
				if res.Warnings[i].Code == "UNVERIFIED_STRUCTURE" {
					flagged = &res.Warnings[i]
				}
			}
			if flagged == nil {
				t.Fatalf("no UNVERIFIED_STRUCTURE warning in %+v", res.Warnings)
			}
			if !strings.Contains(flagged.Message, "NOFLAGS") || strings.Contains(flagged.Message, "KNOWN") {
				t.Fatalf("warning should name NOFLAGS only, got %q", flagged.Message)
			}
		})
	}
}

func TestStructureFlagKnownToAnySourceSurvivesTheMerge(t *testing.T) {
	silent := fakeprovider.New(domain.ETF{Ticker: "LEV"}) // doesn't report leverage
	informed := fakeprovider.New(domain.ETF{Ticker: "LEV", IsLeveraged: domain.Flag(true)})
	other := fakeprovider.New(domain.ETF{Ticker: "PLAIN", IsLeveraged: domain.Flag(false)})
	svc := newService(search.NewAggregatedProvider(silent, informed, other))

	req := standardRequest(dto.ExposureRequest{})
	req.Constraints = dto.Constraints{ExcludeLeveragedETFs: true}
	res, err := svc.DiscoverETFs(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if got := tickers(res.Results); len(got) != 1 || got[0] != "PLAIN" {
		t.Fatalf("results = %v, want [PLAIN] (LEV is leveraged per one source)", got)
	}
}

func requestFor(country, account string) dto.DiscoveryRequest {
	req := standardRequest(dto.ExposureRequest{})
	req.InvestorProfile.Country = country
	req.InvestorProfile.AccountType = account
	req.OutputOptions.ExplainEligibility = true
	req.OutputOptions.IncludeAlternatives = true
	return req
}

// onlyFund returns the single ETF from results or alternatives (unknown and
// ineligible funds land in alternatives).
func onlyFund(t *testing.T, res *discovery.DiscoveryResult) dto.ETFResult {
	t.Helper()
	all := append(append([]dto.ETFResult{}, res.Results...), res.Alternatives...)
	if len(all) != 1 {
		t.Fatalf("got %d funds, want 1", len(all))
	}
	return all[0]
}

func hasWarningContaining(e dto.EligibilityDetail, substr string) bool {
	for _, w := range e.Warnings {
		if strings.Contains(strings.ToLower(w), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// spy is a US-listed ETF as live data describes it today: exchange known,
// domicile and legal structure not reported.
func spy() domain.ETF {
	return domain.ETF{Ticker: "SPY", Exchange: "PCX", ExchangeCountry: "US", Currency: "USD",
		IsLeveraged: domain.Flag(false), IsInverse: domain.Flag(false)}
}

func TestISAWithUSListedETFIsConditionalWithRetailDisclosureWarning(t *testing.T) {
	el := onlyFund(t, discover(t, requestFor("GB", "isa"), spy())).Eligibility

	if el.Status != "conditional" {
		t.Fatalf("status = %q, want conditional", el.Status)
	}
	if el.RuleVersion != "isa_uk_v1.0_2026" {
		t.Fatalf("ruleVersion = %q, want isa_uk_v1.0_2026", el.RuleVersion)
	}
	if !hasWarningContaining(el, "product summary") {
		t.Fatalf("no UK retail product-summary/KID warning in %v", el.Warnings)
	}
}

func TestISAWithNoListingOrStructureDataIsUnknown(t *testing.T) {
	el := onlyFund(t, discover(t, requestFor("GB", "isa"), domain.ETF{Ticker: "BLANK"})).Eligibility

	if el.Status != "unknown" || el.IsEligible {
		t.Fatalf("status = %q (isEligible %v), want unknown/false", el.Status, el.IsEligible)
	}
}

func TestISAWithNonUCITSFundQuotedOnlyOTCIsIneligible(t *testing.T) {
	otc := domain.ETF{Ticker: "OTCX", Exchange: "PNK", LegalStructure: "Grantor Trust"}

	res := discover(t, requestFor("GB", "isa"), otc)

	// Ineligible funds are always filtered out; the summary counts them.
	if res.Summary.TotalIneligible != 1 || len(res.Results)+len(res.Alternatives) != 0 {
		t.Fatalf("summary = %+v, results = %d, alternatives = %d; want 1 ineligible and nothing shown",
			res.Summary, len(res.Results), len(res.Alternatives))
	}
}

func TestISALeveragedETFIsConditionalNotIneligible(t *testing.T) {
	lev := spy()
	lev.Ticker, lev.IsLeveraged = "SSO", domain.Flag(true)

	el := onlyFund(t, discover(t, requestFor("GB", "isa"), lev)).Eligibility

	if el.Status != "conditional" {
		t.Fatalf("status = %q, want conditional (leverage is legal in an ISA)", el.Status)
	}
	if !hasWarningContaining(el, "leveraged") {
		t.Fatalf("no provider-restriction warning for leverage in %v", el.Warnings)
	}
}

func TestISAUKUCITSIsConditionalPendingFCARegisterCheck(t *testing.T) {
	ucits := domain.ETF{Ticker: "VUSA", Exchange: "LSE", Domicile: "GB", LegalStructure: "UCITS",
		IsLeveraged: domain.Flag(false), IsInverse: domain.Flag(false)}

	el := onlyFund(t, discover(t, requestFor("GB", "isa"), ucits)).Eligibility

	if el.Status != "conditional" || !hasWarningContaining(el, "FCA register") {
		t.Fatalf("status = %q, warnings = %v; want conditional with an FCA-register note", el.Status, el.Warnings)
	}
}

func TestIRAPlainUSETFIsConditionalOnCustodian(t *testing.T) {
	for _, account := range []string{"ira", "roth_ira"} {
		el := onlyFund(t, discover(t, requestFor("US", account), spy())).Eligibility

		if el.Status != "conditional" {
			t.Fatalf("%s: status = %q, want conditional", account, el.Status)
		}
		if el.RuleVersion != "ira_us_v1.0_2026" {
			t.Fatalf("%s: ruleVersion = %q, want ira_us_v1.0_2026", account, el.RuleVersion)
		}
		if !hasWarningContaining(el, "custodian") {
			t.Fatalf("%s: no custodian warning in %v", account, el.Warnings)
		}
	}
}

func TestIRASpecialCasesStayConditionalWithTheirWarning(t *testing.T) {
	gold := spy()
	gold.Ticker, gold.AssetClass, gold.IsPhysical = "GLD", "Commodity", domain.Flag(true)
	foreign := spy()
	foreign.Ticker, foreign.Domicile, foreign.Exchange = "VUAA", "IE", "LSE"
	inverse := spy()
	inverse.Ticker, inverse.IsInverse = "SH", domain.Flag(true)

	cases := []struct {
		etf  domain.ETF
		want string // substring of the expected warning
	}{
		{gold, "collectible"},
		{foreign, "non-US"},
		{inverse, "inverse"},
	}
	for _, tc := range cases {
		t.Run(tc.etf.Ticker, func(t *testing.T) {
			el := onlyFund(t, discover(t, requestFor("US", "ira"), tc.etf)).Eligibility

			if el.Status != "conditional" {
				t.Fatalf("status = %q, want conditional (IRA law doesn't exclude it)", el.Status)
			}
			if !hasWarningContaining(el, tc.want) {
				t.Fatalf("no %q warning in %v", tc.want, el.Warnings)
			}
		})
	}
}

func TestFourOhOneKIsAlwaysUnknownBecauseThePlanMenuDecides(t *testing.T) {
	full := spy()
	full.Domicile, full.LegalStructure, full.IsPhysical = "US", "Open-end fund", domain.Flag(true)

	el := onlyFund(t, discover(t, requestFor("US", "401k"), full)).Eligibility

	if el.Status != "unknown" || el.IsEligible {
		t.Fatalf("status = %q (isEligible %v), want unknown/false", el.Status, el.IsEligible)
	}
	if el.RuleVersion != "k401_us_v1.0_2026" {
		t.Fatalf("ruleVersion = %q, want k401_us_v1.0_2026", el.RuleVersion)
	}
	if !hasWarningContaining(el, "plan's investment menu") {
		t.Fatalf("no plan-menu warning in %v", el.Warnings)
	}
}

func TestUnknownTERAndAUMScoreNeutrallyInRanking(t *testing.T) {
	cases := []struct {
		component string
	}{
		{"fees"},      // TER 0 = unknown
		{"stability"}, // AUM 0 = unknown
	}
	for _, tc := range cases {
		t.Run(tc.component, func(t *testing.T) {
			req := standardRequest(dto.ExposureRequest{})
			req.RankingPreferences = dto.RankingPreferences{Weighting: map[string]float64{tc.component: 1}}

			res := discover(t, req, domain.ETF{Ticker: "UNKNOWN"})

			// Neutral 0.5 on the only weighted component → ranking score 50.
			assertScore(t, "rankingScore", onlyResult(t, res).RankingScore, 50)
		})
	}
}

func TestSummaryListsTheDataSourcesThatActuallySuppliedData(t *testing.T) {
	a := domain.ETF{Ticker: "A", DataSources: []domain.DataSource{{Provider: "Yahoo Finance"}, {Provider: "JSE"}}}
	b := domain.ETF{Ticker: "B", DataSources: []domain.DataSource{{Provider: "Yahoo Finance"}}}

	res := discover(t, standardRequest(dto.ExposureRequest{}), a, b)

	got := res.Summary.DataSourcesQueried
	if len(got) != 2 || got[0] != "JSE" || got[1] != "Yahoo Finance" {
		t.Fatalf("dataSourcesQueried = %v, want [JSE Yahoo Finance]", got)
	}
}
