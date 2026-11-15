package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

const ownerID = "11111111-1111-1111-1111-111111111111"

// invite has the signed-in user create an invitation and returns its token.
func invite(t *testing.T, api http.Handler, inviterToken, email string) string {
	t.Helper()
	rec := do(t, api, call{method: "POST", path: "/api/v1/invitations", token: inviterToken, body: map[string]string{"email": email}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invitation: %d %s", rec.Code, rec.Body.String())
	}
	var inv struct {
		Token, Link, Email string
		ExpiresAt          time.Time
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &inv)
	if inv.Token == "" || inv.Link != "/invite/"+inv.Token || inv.Email != strings.ToLower(email) || !inv.ExpiresAt.After(time.Now()) {
		t.Fatalf("invitation = %+v", inv)
	}
	return inv.Token
}

func TestInvitedPersonGetsInOnceWithTheirLink(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)
	link := invite(t, api, owner, "Friend@Example.com")
	friend := iss.Token(t, "33333333-3333-3333-3333-333333333333", "friend@example.com")

	if rec := do(t, api, call{method: "POST", path: "/api/v1/discover", token: friend, inviteToken: link, body: discoverBody}); rec.Code != http.StatusOK {
		t.Fatalf("invited friend: %d, want 200; %s", rec.Code, rec.Body.String())
	}
	// Admitted for good: later calls need no invite token.
	rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: friend})
	var me struct{ Role string }
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if rec.Code != http.StatusOK || me.Role != "member" {
		t.Fatalf("me: %d %+v, want member", rec.Code, me)
	}
	// The link is used up: another account with the same email can't reuse it.
	twin := iss.Token(t, "44444444-4444-4444-4444-444444444444", "friend@example.com")
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: twin, inviteToken: link}); rec.Code != http.StatusForbidden {
		t.Fatalf("reused link: %d, want 403", rec.Code)
	}
}

func TestInviteLinkOnlyWorksForTheInvitedEmail(t *testing.T) {
	api, iss := newAuthedAPI(t)
	link := invite(t, api, iss.Token(t, ownerID, adminEmail), "friend@example.com")
	someoneElse := iss.Token(t, "55555555-5555-5555-5555-555555555555", "other@example.com")

	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: someoneElse, inviteToken: link}); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong email with link: %d, want 403", rec.Code)
	}
}

func TestMembersCanInviteToo(t *testing.T) {
	api, iss := newAuthedAPI(t)
	link := invite(t, api, iss.Token(t, ownerID, adminEmail), "friend@example.com")
	friend := iss.Token(t, "33333333-3333-3333-3333-333333333333", "friend@example.com")
	do(t, api, call{method: "GET", path: "/api/v1/me", token: friend, inviteToken: link})

	second := invite(t, api, friend, "cousin@example.com") // a member, not an admin
	cousin := iss.Token(t, "66666666-6666-6666-6666-666666666666", "cousin@example.com")
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: cousin, inviteToken: second}); rec.Code != http.StatusOK {
		t.Fatalf("member-invited cousin: %d, want 200", rec.Code)
	}
}

// joinAsMember invites email (by the owner) and signs that user in once.
func joinAsMember(t *testing.T, api http.Handler, iss *authtest.Issuer, id, email string) string {
	t.Helper()
	link := invite(t, api, iss.Token(t, ownerID, adminEmail), email)
	tok := iss.Token(t, id, email)
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: tok, inviteToken: link}); rec.Code != http.StatusOK {
		t.Fatalf("join %s: %d", email, rec.Code)
	}
	return tok
}

func TestAdminAPIIsForAdminsOnly(t *testing.T) {
	api, iss := newAuthedAPI(t)
	friend := joinAsMember(t, api, iss, "33333333-3333-3333-3333-333333333333", "friend@example.com")

	for _, path := range []string{"/api/v1/admin/settings", "/api/v1/admin/users", "/api/v1/admin/invitations", "/api/v1/admin/data-status"} {
		if rec := do(t, api, call{method: "GET", path: path, token: friend}); rec.Code != http.StatusForbidden {
			t.Errorf("member GET %s: %d, want 403", path, rec.Code)
		}
	}
}

func TestTurningInviteOnlyOffLetsNewSignUpsIn(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)

	rec := do(t, api, call{method: "GET", path: "/api/v1/admin/settings", token: owner})
	var s struct {
		InviteOnly bool    `json:"inviteOnly"`
		Notice     *string `json:"notice"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if rec.Code != http.StatusOK || !s.InviteOnly {
		t.Fatalf("settings: %d %+v, want inviteOnly on by default", rec.Code, s)
	}

	notice := "Data refreshed monthly"
	if rec := do(t, api, call{method: "PUT", path: "/api/v1/admin/settings", token: owner,
		body: map[string]any{"inviteOnly": false, "notice": notice}}); rec.Code != http.StatusOK {
		t.Fatalf("update settings: %d %s", rec.Code, rec.Body.String())
	}

	newcomer := iss.Token(t, "77777777-7777-7777-7777-777777777777", "newcomer@example.com")
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: newcomer}); rec.Code != http.StatusOK {
		t.Fatalf("sign-up with invite-only off: %d, want 200", rec.Code)
	}
}

const friendID = "33333333-3333-3333-3333-333333333333"

func TestAdminCanDisableEnableAndPromoteUsers(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)
	friend := joinAsMember(t, api, iss, friendID, "friend@example.com")
	patch := func(body map[string]string) int {
		return do(t, api, call{method: "PATCH", path: "/api/v1/admin/users/" + friendID, token: owner, body: body}).Code
	}

	if code := patch(map[string]string{"status": "disabled"}); code != http.StatusOK {
		t.Fatalf("disable: %d", code)
	}
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: friend}); rec.Code != http.StatusForbidden {
		t.Fatalf("disabled user: %d, want 403", rec.Code)
	}
	if code := patch(map[string]string{"status": "active", "role": "admin"}); code != http.StatusOK {
		t.Fatalf("enable + promote: %d", code)
	}
	if rec := do(t, api, call{method: "GET", path: "/api/v1/admin/users", token: friend}); rec.Code != http.StatusOK {
		t.Fatalf("promoted friend on admin API: %d, want 200", rec.Code)
	}
}

func TestTheLastActiveAdminCannotBeRemoved(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)
	do(t, api, call{method: "GET", path: "/api/v1/me", token: owner})

	for _, body := range []map[string]string{{"role": "member"}, {"status": "disabled"}} {
		if rec := do(t, api, call{method: "PATCH", path: "/api/v1/admin/users/" + ownerID, token: owner, body: body}); rec.Code != http.StatusConflict {
			t.Errorf("%v on the only admin: %d, want 409", body, rec.Code)
		}
	}
}

func TestRevokedInviteLinkNoLongerWorks(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)
	link := invite(t, api, owner, "friend@example.com")

	var list []struct {
		ID    int64
		Email string
	}
	_ = json.Unmarshal(do(t, api, call{method: "GET", path: "/api/v1/admin/invitations", token: owner}).Body.Bytes(), &list)
	if len(list) != 1 || list[0].Email != "friend@example.com" {
		t.Fatalf("invitations = %+v", list)
	}
	if rec := do(t, api, call{method: "DELETE", path: "/api/v1/admin/invitations/" + strconv.FormatInt(list[0].ID, 10), token: owner}); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", rec.Code)
	}

	friend := iss.Token(t, friendID, "friend@example.com")
	if rec := do(t, api, call{method: "GET", path: "/api/v1/me", token: friend, inviteToken: link}); rec.Code != http.StatusForbidden {
		t.Fatalf("revoked link: %d, want 403", rec.Code)
	}
}

func TestDataStatusReportsWhatIsStored(t *testing.T) {
	api, iss := newAuthedAPI(t)
	owner := iss.Token(t, ownerID, adminEmail)

	rec := do(t, api, call{method: "GET", path: "/api/v1/admin/data-status", token: owner})
	var status struct {
		Listings []any                        `json:"listings"`
		Prices   struct{ EtfsWithPrices int } `json:"prices"`
		Runs     []any                        `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("data status: %d %s", rec.Code, rec.Body.String())
	}
	if len(status.Listings) != 0 || status.Prices.EtfsWithPrices != 0 || len(status.Runs) != 0 {
		t.Fatalf("empty store should report nothing: %+v", status)
	}
}
