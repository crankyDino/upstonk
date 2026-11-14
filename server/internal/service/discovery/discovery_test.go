package discovery_test

import (
	"context"
	"errors"
	"testing"

	"upstonk/internal/api/dto"
	"upstonk/internal/app"
	"upstonk/internal/service/discovery"
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
