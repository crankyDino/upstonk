package handlers

import (
	"encoding/json"
	"net/http"

	"upstonk/internal/auth"
)

// HandleMe returns the signed-in member: GET /api/v1/me.
func HandleMe(w http.ResponseWriter, r *http.Request) {
	m, ok := auth.MemberFrom(r.Context())
	if !ok {
		http.Error(w, `{"error":"UNAUTHORIZED","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id": m.UUIDString(), "email": m.Email, "role": m.Role})
}
