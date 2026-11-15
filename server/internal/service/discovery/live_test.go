//go:build live

// Live tests hit the real public data sources. They are excluded from the
// default `go test ./...`; run them with `go test -tags live ./...`.
// They assert invariants only (shape, provenance, eligibility vocabulary),
// never market values, because those change daily.
package discovery_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"upstonk/internal/api/dto"
	"upstonk/internal/app"
	"upstonk/internal/service/discovery"
)

func requireNetwork(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "query1.finance.yahoo.com:443", 5*time.Second)
	if err != nil {
		t.Skipf("network unavailable, skipping live test: %v", err)
	}
	conn.Close()
}

func newLiveService() *discovery.Service {
	return discovery.NewService(
		app.NewSearchProvider(os.Getenv("ALPHA_VANTAGE_API_KEY"), nil),
		app.NewEligibilityEngine(),
		app.NewRankingEngine(),
	)
}

var allowedStatuses = map[string]bool{"eligible": true, "conditional": true, "ineligible": true, "unknown": true}

func TestLiveZATFSAReturnsJSEListedResultsWithProvenance(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := dto.DiscoveryRequest{
		InvestorProfile: dto.InvestorProfile{Country: "ZA", AccountType: "tfsa", Currency: "ZAR"},
		Exposure: dto.ExposureRequest{
			Assets:    dto.AssetExposureRequest{AssetClasses: []string{"equity"}},
			Geography: dto.GeographyExposureRequest{Markets: []string{"south africa"}},
		},
		InvestmentVehicles: []string{"etf"},
		OutputOptions: dto.OutputOptions{
			MaxResults: 10, IncludeAlternatives: true, IncludeSourceLinks: true,
		},
	}

	res, err := newLiveService().DiscoverETFs(ctx, req)
	if err != nil {
		t.Fatalf("DiscoverETFs: %v", err)
	}

	all := append(res.Results, res.Alternatives...)
	if len(all) == 0 {
		t.Fatalf("no results or alternatives from live sources (summary: %+v)", res.Summary)
	}
	for _, r := range all {
		if r.Exchange != "JSE" {
			t.Errorf("%s: exchange = %q, want JSE (ZA searches only JSE)", r.Ticker, r.Exchange)
		}
		if r.TER < 0 {
			t.Errorf("%s: negative TER %v", r.Ticker, r.TER)
		}
		if !allowedStatuses[r.Eligibility.Status] {
			t.Errorf("%s: eligibility status %q not in allowed set", r.Ticker, r.Eligibility.Status)
		}
		if len(r.DataSources) == 0 {
			t.Errorf("%s: no data sources (provenance lost)", r.Ticker)
		}
		for _, ds := range r.DataSources {
			if ds.Provider == "" || ds.Date == "" {
				t.Errorf("%s: incomplete data source %+v", r.Ticker, ds)
			}
		}
	}
}

// Each Phase 6 rule, against live data: it must run (ruleVersion set) and,
// under ADR 0001, must never claim plain "eligible".
func TestLiveTaxAdvantagedRulesNeverClaimPlainEligible(t *testing.T) {
	requireNetwork(t)

	cases := []struct {
		country, account, version string
		allowed                   map[string]bool
	}{
		{"GB", "isa", "isa_uk_v1.0_2026", map[string]bool{"conditional": true, "unknown": true}},
		{"US", "ira", "ira_us_v1.0_2026", map[string]bool{"conditional": true}},
		{"US", "roth_ira", "ira_us_v1.0_2026", map[string]bool{"conditional": true}},
		{"US", "401k", "k401_us_v1.0_2026", map[string]bool{"unknown": true}},
	}
	svc := newLiveService()
	for _, tc := range cases {
		t.Run(tc.country+"/"+tc.account, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			req := dto.DiscoveryRequest{
				InvestorProfile:    dto.InvestorProfile{Country: tc.country, AccountType: tc.account, Currency: "USD"},
				Exposure:           dto.ExposureRequest{Geography: dto.GeographyExposureRequest{Markets: []string{"usa"}}},
				InvestmentVehicles: []string{"etf"},
				OutputOptions:      dto.OutputOptions{MaxResults: 10, IncludeAlternatives: true},
			}
			res, err := svc.DiscoverETFs(ctx, req)
			if err != nil {
				t.Fatalf("DiscoverETFs: %v", err)
			}
			all := append(res.Results, res.Alternatives...)
			if len(all) == 0 {
				t.Fatalf("no funds returned (summary %+v)", res.Summary)
			}
			for _, r := range all {
				if r.Eligibility.RuleVersion != tc.version {
					t.Errorf("%s: ruleVersion %q, want %q", r.Ticker, r.Eligibility.RuleVersion, tc.version)
				}
				if !tc.allowed[r.Eligibility.Status] {
					t.Errorf("%s: status %q not allowed with today's data", r.Ticker, r.Eligibility.Status)
				}
			}
		})
	}
}
