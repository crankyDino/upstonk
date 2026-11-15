package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/router"
	"upstonk/internal/app"
	"upstonk/internal/domain"
	"upstonk/internal/service/discovery"
	"upstonk/internal/service/search"
	"upstonk/internal/store/storedb"
	"upstonk/internal/testutil/dbtest"
	"upstonk/internal/testutil/fakeprovider"
)

// ZA searches are answered from the store; other countries from the live
// sources (a fake here).
func TestZASearchesReadTheStoreOthersUseLiveSources(t *testing.T) {
	pool := dbtest.New(t)
	asOf := pgtype.Date{Time: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), Valid: true}
	name := "Satrix 40 ETF"
	if err := storedb.New(pool).UpsertETF(context.Background(), storedb.UpsertETFParams{
		Code: "STX40", Name: name, Exchange: "JSE", ListingSource: "JSE", ListedAsOf: asOf,
	}); err != nil {
		t.Fatal(err)
	}
	live := fakeprovider.New(domain.ETF{Ticker: "SPY", Exchange: "PCX"})
	provider := search.ByCountry(map[string]search.Provider{"ZA": search.NewStoredProvider(pool)}, live)
	svc := discovery.NewService(provider, app.NewEligibilityEngine(), app.NewRankingEngine())
	api := router.New(handlers.NewDiscoveryHandler(svc))

	tickersFor := func(country, currency string) []string {
		body := []byte(`{"investorProfile":{"country":"` + country + `","accountType":"standard","currency":"` + currency + `"},
			"exposure":{"assets":{"assetClasses":["equity"]}},"investmentVehicles":["etf"]}`)
		rec := postDiscover(t, api, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", country, rec.Code, rec.Body.String())
		}
		var resp struct{ Results []struct{ Ticker string } }
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		out := []string{}
		for _, r := range resp.Results {
			out = append(out, r.Ticker)
		}
		return out
	}

	if got := tickersFor("ZA", "ZAR"); len(got) != 1 || got[0] != "STX40" {
		t.Fatalf("ZA results = %v, want [STX40] from the store", got)
	}
	if got := tickersFor("US", "USD"); len(got) != 1 || got[0] != "SPY" {
		t.Fatalf("US results = %v, want [SPY] from live sources", got)
	}
}
