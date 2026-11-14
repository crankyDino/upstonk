package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
