package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/router"
	"upstonk/internal/app"
	"upstonk/internal/domain"
	"upstonk/internal/service/discovery"
	"upstonk/internal/testutil/fakeprovider"
)

// newAPI wires the real router, handler, service and engines around a fake
// data provider.
func newAPI(etfs ...domain.ETF) http.Handler {
	svc := discovery.NewService(fakeprovider.New(etfs...), app.NewEligibilityEngine(), app.NewRankingEngine())
	return router.New(handlers.NewDiscoveryHandler(svc))
}

func TestHealthReportsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	newAPI().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["status"] == nil {
		t.Fatalf("body has no status field: %s", rec.Body.String())
	}
}

func TestCrossOriginRequestIsAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	newAPI().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestPreflightForDiscoverSucceeds(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/discover", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()

	newAPI().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// The testdata fixtures are generated from the GUI's own code by
// `npm run contract:fixture` in gui/, so these tests break when the GUI
// starts sending something the server rejects.

func postDiscover(t *testing.T, api http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discover", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture (regenerate with `npm run contract:fixture` in gui/): %v", err)
	}
	return b
}

var guiFixtureETF = domain.ETF{Ticker: "STX500", Exchange: "JSE", TER: 0.1, AUM: 5e9, AverageDailyVolume: 200000}

func TestServerAcceptsTheGUIDefaultRequest(t *testing.T) {
	rec := postDiscover(t, newAPI(guiFixtureETF), readFixture(t, "gui_default_request.json"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
}

func TestServerAcceptsEveryAccountPairTheGUIOffers(t *testing.T) {
	var base map[string]any
	if err := json.Unmarshal(readFixture(t, "gui_default_request.json"), &base); err != nil {
		t.Fatal(err)
	}
	var pairs []struct{ Country, AccountType string }
	if err := json.Unmarshal(readFixture(t, "gui_account_pairs.json"), &pairs); err != nil {
		t.Fatal(err)
	}
	if len(pairs) == 0 {
		t.Fatal("no account pairs in fixture")
	}

	currency := map[string]string{"ZA": "ZAR", "US": "USD", "GB": "GBP"}
	for _, p := range pairs {
		t.Run(p.Country+"/"+p.AccountType, func(t *testing.T) {
			profile := base["investorProfile"].(map[string]any)
			profile["country"] = p.Country
			profile["accountType"] = p.AccountType
			profile["currency"] = currency[p.Country]
			body, _ := json.Marshal(base)

			rec := postDiscover(t, newAPI(guiFixtureETF), body)

			if rec.Code == http.StatusBadRequest {
				t.Fatalf("server rejected a pair the GUI offers: %s", rec.Body.String())
			}
		})
	}
}
