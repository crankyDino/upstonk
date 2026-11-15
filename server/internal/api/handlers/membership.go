package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/auth"
	"upstonk/internal/store/storedb"
)

// InviteValidity is how long an invite link stays usable.
const InviteValidity = 14 * 24 * time.Hour

// MembershipHandler serves invitations for members and the admin API.
type MembershipHandler struct {
	q *storedb.Queries
}

func NewMembershipHandler(q *storedb.Queries) *MembershipHandler { return &MembershipHandler{q: q} }

// HandleCreateInvitation: POST /api/v1/invitations {"email": "..."}.
// Any active member may invite; the response carries the link to share.
func (h *MembershipHandler) HandleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	m, _ := auth.MemberFrom(r.Context())
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_JSON", "Body must be JSON with an email")
		return
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(body.Email))
	if err != nil || addr.Name != "" {
		writeErr(w, http.StatusBadRequest, "INVALID_EMAIL", "A plain email address is required")
		return
	}
	token, err := randomToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not create invitation")
		return
	}
	inv, err := h.q.CreateInvitation(r.Context(), storedb.CreateInvitationParams{
		Email: addr.Address, Token: token, InvitedBy: m.ID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(InviteValidity), Valid: true},
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not create invitation")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": inv.ID, "email": inv.Email, "token": inv.Token, "link": "/invite/" + inv.Token,
		"expiresAt": inv.ExpiresAt.Time,
	})
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "code": code, "message": msg})
}
