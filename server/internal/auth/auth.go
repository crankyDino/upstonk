// Package auth protects the API with Supabase Auth and decides membership.
//
// Supabase owns identities: the GUI signs in with supabase-js and sends the
// access token as "Authorization: Bearer <JWT>". The token is verified
// against the project's JWKS (public keys; no secret on the server), its
// issuer, audience "authenticated" and expiry. Whether that identity may use
// UPSTONK is decided here, from the upstonk.app_users table:
//
//   - an email in ADMIN_EMAILS is admitted (and always kept) as admin;
//   - with invite_only off, any new account is admitted as a member;
//   - otherwise a new account needs an open invitation whose link token
//     (X-Invite-Token header) and email both match;
//   - disabled users are refused.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/store/storedb"
)

// Config for the gate.
type Config struct {
	SupabaseURL string   // https://<ref>.supabase.co
	AdminEmails []string // always admins
}

// Member is the signed-in, admitted user of a request.
type Member struct {
	ID    pgtype.UUID
	Email string
	Role  string // admin | member
}

func (m Member) IsAdmin() bool { return m.Role == "admin" }

// Queries exposes the membership store to the HTTP handlers.
func (g *Gate) Queries() *storedb.Queries { return g.q }

// UUIDString renders the member's id.
func (m Member) UUIDString() string { return uuidString(m.ID) }

// Gate verifies tokens and admits members.
type Gate struct {
	keys   keyfunc.Keyfunc
	issuer string
	admins map[string]bool
	q      *storedb.Queries
}

// New builds a gate for a Supabase project. It fetches the project's JWKS
// (and refreshes it in the background).
func New(cfg Config, pool *pgxpool.Pool) (*Gate, error) {
	base := strings.TrimRight(cfg.SupabaseURL, "/")
	if base == "" {
		return nil, errors.New("auth: SupabaseURL is required")
	}
	if pool == nil {
		return nil, errors.New("auth: a database is required for membership (set DATABASE_URL)")
	}
	keys, err := keyfunc.NewDefaultCtx(context.Background(), []string{base + "/auth/v1/.well-known/jwks.json"})
	if err != nil {
		return nil, fmt.Errorf("auth: JWKS: %w", err)
	}
	admins := map[string]bool{}
	for _, e := range cfg.AdminEmails {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			admins[e] = true
		}
	}
	return &Gate{keys: keys, issuer: base + "/auth/v1", admins: admins, q: storedb.New(pool)}, nil
}

type ctxKey struct{}

// MemberFrom returns the admitted member of a request that passed the gate.
func MemberFrom(ctx context.Context) (Member, bool) {
	m, ok := ctx.Value(ctxKey{}).(Member)
	return m, ok
}

// Require wraps a handler: 401 without a valid token, 403 when not admitted.
func (g *Gate) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { // CORS preflight carries no token
			next.ServeHTTP(w, r)
			return
		}
		id, email, err := g.verify(r.Header.Get("Authorization"))
		if err != nil {
			deny(w, http.StatusUnauthorized, "UNAUTHORIZED", "Sign in required")
			return
		}
		m, err := g.admit(r.Context(), id, email, r.Header.Get("X-Invite-Token"))
		switch {
		case errors.Is(err, errNotAdmitted):
			deny(w, http.StatusForbidden, "NOT_INVITED", "This app is invite-only. Ask a member for an invite link.")
			return
		case errors.Is(err, errDisabled):
			deny(w, http.StatusForbidden, "ACCOUNT_DISABLED", "Your account has been disabled.")
			return
		case err != nil:
			log.Printf("auth: admit %s: %v", email, err)
			deny(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not check membership")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, m)))
	})
}

// RequireAdmin wraps a handler that only admins may use (after Require).
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m, ok := MemberFrom(r.Context()); !ok || !m.IsAdmin() {
			deny(w, http.StatusForbidden, "ADMIN_ONLY", "Admins only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *Gate) verify(header string) (pgtype.UUID, string, error) {
	var id pgtype.UUID
	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || raw == "" {
		return id, "", errors.New("no bearer token")
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, g.keys.Keyfunc,
		jwt.WithValidMethods([]string{"ES256", "RS256"}),
		jwt.WithIssuer(g.issuer),
		jwt.WithAudience("authenticated"),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return id, "", err
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if err := id.Scan(sub); err != nil || email == "" {
		return id, "", errors.New("token without a user id or email")
	}
	return id, strings.ToLower(email), nil
}

var (
	errNotAdmitted = errors.New("not admitted")
	errDisabled    = errors.New("disabled")
)

func (g *Gate) admit(ctx context.Context, id pgtype.UUID, email, inviteToken string) (Member, error) {
	u, err := g.q.GetAppUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := g.firstAdmission(ctx, id, email, inviteToken); err != nil {
			return Member{}, err
		}
		u, err = g.q.GetAppUser(ctx, id)
	}
	if err != nil {
		return Member{}, err
	}
	if u.Status == "disabled" {
		return Member{}, errDisabled
	}
	if g.admins[email] && u.Role != "admin" { // ADMIN_EMAILS are always admins
		if err := g.q.SetAppUserRole(ctx, storedb.SetAppUserRoleParams{ID: id, Role: "admin"}); err != nil {
			return Member{}, err
		}
		u.Role = "admin"
	}
	_ = g.q.TouchAppUser(ctx, storedb.TouchAppUserParams{ID: id, Email: email})
	return Member{ID: id, Email: email, Role: u.Role}, nil
}

func (g *Gate) firstAdmission(ctx context.Context, id pgtype.UUID, email, inviteToken string) error {
	if g.admins[email] {
		return g.q.InsertAppUser(ctx, storedb.InsertAppUserParams{ID: id, Email: email, Role: "admin"})
	}
	settings, err := g.q.GetSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.InviteOnly {
		return g.q.InsertAppUser(ctx, storedb.InsertAppUserParams{ID: id, Email: email, Role: "member"})
	}
	if inviteToken == "" {
		return errNotAdmitted
	}
	inv, err := g.q.FindOpenInvitation(ctx, storedb.FindOpenInvitationParams{Token: inviteToken, Email: email})
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotAdmitted
	}
	if err != nil {
		return err
	}
	if err := g.q.InsertAppUser(ctx, storedb.InsertAppUserParams{ID: id, Email: email, Role: "member", InvitedBy: inv.InvitedBy}); err != nil {
		return err
	}
	return g.q.UseInvitation(ctx, storedb.UseInvitationParams{ID: inv.ID, UsedBy: id})
}

func deny(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "code": code, "message": msg})
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
