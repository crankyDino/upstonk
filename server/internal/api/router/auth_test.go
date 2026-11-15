package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/router"
	"upstonk/internal/app"
	"upstonk/internal/auth"
	"upstonk/internal/domain"
	"upstonk/internal/service/discovery"
	"upstonk/internal/testutil/authtest"
	"upstonk/internal/testutil/dbtest"
	"upstonk/internal/testutil/fakeprovider"
)

const adminEmail = "owner@example.com"

// newAuthedAPI wires the real router with Supabase-style auth (fake JWKS),
// membership in a fresh test database, and ADMIN_EMAILS = owner@example.com.
func newAuthedAPI(t *testing.T) (http.Handler, *authtest.Issuer) {
	t.Helper()
	iss := authtest.New(t)
	pool := dbtest.New(t)
	gate, err := auth.New(auth.Config{SupabaseURL: iss.URL, AdminEmails: []string{adminEmail}}, pool)
	if err != nil {
		t.Fatal(err)
	}
	svc := discovery.NewService(fakeprovider.New(domain.ETF{Ticker: "STX40", Exchange: "JSE"}), app.NewEligibilityEngine(), app.NewRankingEngine())
	return router.New(handlers.NewDiscoveryHandler(svc), router.WithAuth(gate)), iss
}

type call struct {
	method, path, token, inviteToken string
	body                             any
}

func do(t *testing.T, api http.Handler, c call) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if c.body != nil {
		b, _ := json.Marshal(c.body)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(c.method, c.path, body)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.inviteToken != "" {
		req.Header.Set("X-Invite-Token", c.inviteToken)
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec
}

var discoverBody = map[string]any{
	"investorProfile":    map[string]string{"country": "ZA", "accountType": "standard", "currency": "ZAR"},
	"exposure":           map[string]any{"assets": map[string]any{"assetClasses": []string{"equity"}}},
	"investmentVehicles": []string{"etf"},
}

func TestWithAuthHealthIsPublicButDiscoveryNeedsAToken(t *testing.T) {
	api, _ := newAuthedAPI(t)

	if rec := do(t, api, call{method: "GET", path: "/api/v1/health"}); rec.Code != http.StatusOK {
		t.Fatalf("health: %d, want 200", rec.Code)
	}
	if rec := do(t, api, call{method: "POST", path: "/api/v1/discover", body: discoverBody}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("discover without token: %d, want 401", rec.Code)
	}
}

func TestBadTokensAreRejected(t *testing.T) {
	api, iss := newAuthedAPI(t)
	other := authtest.New(t) // a different signing key

	cases := map[string]string{
		"expired":     iss.TokenExpiring(t, "u1", adminEmail, time.Now().Add(-time.Minute)),
		"foreign key": other.Token(t, "u1", adminEmail),
		"garbage":     "not-a-jwt",
	}
	for name, tok := range cases {
		if rec := do(t, api, call{method: "POST", path: "/api/v1/discover", token: tok, body: discoverBody}); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s token: %d, want 401", name, rec.Code)
		}
	}
}

func TestAdminEmailIsAdmittedAsAdmin(t *testing.T) {
	api, iss := newAuthedAPI(t)
	tok := iss.Token(t, "11111111-1111-1111-1111-111111111111", adminEmail)

	if rec := do(t, api, call{method: "POST", path: "/api/v1/discover", token: tok, body: discoverBody}); rec.Code != http.StatusOK {
		t.Fatalf("discover: %d, want 200; %s", rec.Code, rec.Body.String())
	}
	rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: tok})
	var me struct{ Email, Role string }
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if rec.Code != http.StatusOK || me.Email != adminEmail || me.Role != "admin" {
		t.Fatalf("me: %d %+v, want admin", rec.Code, me)
	}
}

func TestInviteOnlyIsOnByDefaultSoUninvitedUsersAreForbidden(t *testing.T) {
	api, iss := newAuthedAPI(t)
	tok := iss.Token(t, "22222222-2222-2222-2222-222222222222", "stranger@example.com")

	if rec := do(t, api, call{method: "POST", path: "/api/v1/discover", token: tok, body: discoverBody}); rec.Code != http.StatusForbidden {
		t.Fatalf("uninvited: %d, want 403", rec.Code)
	}
}
