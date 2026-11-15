package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/router"
	"upstonk/internal/app"
	"upstonk/internal/service/discovery"
	"upstonk/internal/testutil/fakeprovider"
)

// The single-instance deployment: the API server also serves the built GUI.
func newSingleInstance() http.Handler {
	web := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>UPSTONK</title>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	svc := discovery.NewService(fakeprovider.New(), app.NewEligibilityEngine(), app.NewRankingEngine())
	return router.New(handlers.NewDiscoveryHandler(svc),
		router.WithWebApp(web, router.PublicConfig{SupabaseURL: "https://ref.supabase.co", SupabasePublishableKey: "pk_test"}))
}

func get(t *testing.T, api http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSingleInstanceServesTheGUIAndItsClientSideRoutes(t *testing.T) {
	api := newSingleInstance()
	for _, path := range []string{"/", "/admin", "/invite/abc123"} {
		rec := get(t, api, path)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>UPSTONK</title>") {
			t.Errorf("GET %s: %d %q, want index.html", path, rec.Code, rec.Body.String())
		}
	}
	if rec := get(t, api, "/assets/app.js"); rec.Code != http.StatusOK || rec.Body.String() != "console.log('app')" {
		t.Errorf("asset: %d %q", rec.Code, rec.Body.String())
	}
}

func TestUnknownAPIPathsAreJSONNotFoundNotTheGUI(t *testing.T) {
	rec := get(t, newSingleInstance(), "/api/v1/nope")
	if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "<title>") {
		t.Fatalf("GET /api/v1/nope: %d %q, want a 404 that isn't the GUI", rec.Code, rec.Body.String())
	}
}

func TestConfigJSCarriesOnlyThePublicSupabaseSettings(t *testing.T) {
	rec := get(t, newSingleInstance(), "/config.js")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("config.js: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, `"supabaseUrl":"https://ref.supabase.co"`) || !strings.Contains(body, `"supabasePublishableKey":"pk_test"`) {
		t.Fatalf("config.js = %s", body)
	}
}
