// Package authtest stands in for Supabase Auth in tests: it signs ES256
// access tokens with its own key and serves the matching JWKS over HTTP, so
// the real verifier runs end to end without touching Supabase.
package authtest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer mimics a Supabase project URL.
type Issuer struct {
	URL string // stands in for https://<ref>.supabase.co
	key *ecdsa.PrivateKey
	kid string
}

// New starts a JWKS server for a fresh ES256 key; it stops when the test ends.
func New(t *testing.T) *Issuer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	iss := &Issuer{key: key, kid: "test-key"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		b := func(i *big.Int) string { return base64.RawURLEncoding.EncodeToString(i.FillBytes(make([]byte, 32))) }
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": iss.kid,
			"x": b(key.PublicKey.X), "y": b(key.PublicKey.Y),
		}}})
	}))
	t.Cleanup(srv.Close)
	iss.URL = srv.URL
	return iss
}

// Token returns a Supabase-style access token for a user, valid for an hour.
func (i *Issuer) Token(t *testing.T, userID, email string) string {
	return i.TokenExpiring(t, userID, email, time.Now().Add(time.Hour))
}

// TokenExpiring is Token with an explicit expiry.
func (i *Issuer) TokenExpiring(t *testing.T, userID, email string, exp time.Time) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss":   i.URL + "/auth/v1",
		"aud":   "authenticated",
		"sub":   userID,
		"email": email,
		"role":  "authenticated",
		"iat":   time.Now().Unix(),
		"exp":   exp.Unix(),
	})
	tok.Header["kid"] = i.kid
	s, err := tok.SignedString(i.key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
