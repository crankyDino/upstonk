package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgtype"

	"upstonk/internal/auth"
	"upstonk/internal/store/storedb"
)

// --- settings ---

// HandleGetSettings: GET /api/v1/admin/settings
func (h *MembershipHandler) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.q.GetSettings(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not read settings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"inviteOnly": s.InviteOnly, "notice": s.Notice, "updatedAt": s.UpdatedAt.Time})
}

// HandlePutSettings: PUT /api/v1/admin/settings {"inviteOnly": bool, "notice": string|null}
func (h *MembershipHandler) HandlePutSettings(w http.ResponseWriter, r *http.Request) {
	m, _ := auth.MemberFrom(r.Context())
	var body struct {
		InviteOnly *bool   `json:"inviteOnly"`
		Notice     *string `json:"notice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InviteOnly == nil {
		writeErr(w, http.StatusBadRequest, "INVALID_REQUEST", "inviteOnly (bool) is required; notice is optional")
		return
	}
	if body.Notice != nil && *body.Notice == "" {
		body.Notice = nil
	}
	if err := h.q.UpdateSettings(r.Context(), storedb.UpdateSettingsParams{
		InviteOnly: *body.InviteOnly, Notice: body.Notice, UpdatedBy: m.ID,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not save settings")
		return
	}
	h.HandleGetSettings(w, r)
}

// HandleNotice: GET /api/v1/notice — the banner every member sees.
func (h *MembershipHandler) HandleNotice(w http.ResponseWriter, r *http.Request) {
	s, err := h.q.GetSettings(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not read settings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notice": s.Notice})
}

// --- users ---

// HandleListUsers: GET /api/v1/admin/users
func (h *MembershipHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListAppUsers(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list users")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, u := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(u.ID), "email": u.Email, "role": u.Role, "status": u.Status,
			"invitedByEmail": u.InvitedByEmail, "createdAt": u.CreatedAt.Time, "lastSeenAt": tsPtr(u.LastSeenAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// HandlePatchUser: PATCH /api/v1/admin/users/{id} {"role"?: "admin"|"member", "status"?: "active"|"disabled"}
// The last active admin can't be demoted or disabled.
func (h *MembershipHandler) HandlePatchUser(w http.ResponseWriter, r *http.Request) {
	var id pgtype.UUID
	if err := id.Scan(mux.Vars(r)["id"]); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid user id")
		return
	}
	var body struct {
		Role   *string `json:"role"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		(body.Role == nil && body.Status == nil) ||
		(body.Role != nil && *body.Role != "admin" && *body.Role != "member") ||
		(body.Status != nil && *body.Status != "active" && *body.Status != "disabled") {
		writeErr(w, http.StatusBadRequest, "INVALID_REQUEST", `role ("admin"|"member") and/or status ("active"|"disabled") required`)
		return
	}
	ctx := r.Context()
	u, err := h.q.GetAppUser(ctx, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "No such user")
		return
	}
	losesAdmin := u.Role == "admin" && u.Status == "active" &&
		((body.Role != nil && *body.Role == "member") || (body.Status != nil && *body.Status == "disabled"))
	if losesAdmin {
		if n, err := h.q.CountActiveAdmins(ctx); err != nil || n <= 1 {
			writeErr(w, http.StatusConflict, "LAST_ADMIN", "There must be at least one active admin")
			return
		}
	}
	if body.Role != nil {
		if err := h.q.SetAppUserRole(ctx, storedb.SetAppUserRoleParams{ID: id, Role: *body.Role}); err != nil {
			writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not update role")
			return
		}
	}
	if body.Status != nil {
		if err := h.q.SetAppUserStatus(ctx, storedb.SetAppUserStatusParams{ID: id, Status: *body.Status}); err != nil {
			writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not update status")
			return
		}
	}
	h.HandleListUsers(w, r)
}

// --- invitations ---

// HandleListInvitations: GET /api/v1/admin/invitations
func (h *MembershipHandler) HandleListInvitations(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListInvitations(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list invitations")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, i := range rows {
		out = append(out, map[string]any{
			"id": i.ID, "email": i.Email, "invitedByEmail": i.InvitedByEmail,
			"createdAt": i.CreatedAt.Time, "expiresAt": i.ExpiresAt.Time,
			"usedAt": tsPtr(i.UsedAt), "revokedAt": tsPtr(i.RevokedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleRevokeInvitation: DELETE /api/v1/admin/invitations/{id}
func (h *MembershipHandler) HandleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_ID", "Invalid invitation id")
		return
	}
	n, err := h.q.RevokeInvitation(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not revoke invitation")
		return
	}
	if n == 0 {
		writeErr(w, http.StatusConflict, "NOT_OPEN", "Invitation not found, already used or already revoked")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- data status ---

// HandleDataStatus: GET /api/v1/admin/data-status — freshness and coverage per source.
func (h *MembershipHandler) HandleDataStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	listings, err1 := h.q.DataStatusListings(ctx)
	prices, err2 := h.q.DataStatusPrices(ctx)
	stale, err3 := h.q.DataStatusStalePrices(ctx)
	factsheets, err4 := h.q.DataStatusFactsheets(ctx)
	runs, err5 := h.q.ListLatestIngestRuns(ctx)
	for _, err := range []error{err1, err2, err3, err4, err5} {
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not read data status")
			return
		}
	}
	staleCodes := make([]string, 0, len(stale))
	for _, s := range stale {
		staleCodes = append(staleCodes, s.Code)
	}
	runList := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		var failed map[string]string
		_ = json.Unmarshal(run.Failed, &failed)
		runList = append(runList, map[string]any{
			"kind": run.Kind, "startedAt": run.StartedAt.Time, "finishedAt": tsPtr(run.FinishedAt),
			"stored": run.Stored, "failed": failed,
		})
	}
	listOut := make([]map[string]any, 0, len(listings))
	for _, l := range listings {
		listOut = append(listOut, map[string]any{
			"instrumentType": l.InstrumentType, "active": l.Active, "inactive": l.Inactive,
			"lastImport": dateStr(l.LastImport), "source": l.Source,
		})
	}
	factOut := make([]map[string]any, 0, len(factsheets))
	for _, f := range factsheets {
		factOut = append(factOut, map[string]any{
			"issuer": f.Issuer, "etfs": f.Etfs, "withFactsheet": f.WithFactsheet,
			"oldest": dateStr(f.Oldest), "newest": dateStr(f.Newest),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"listings": listOut,
		"prices": map[string]any{
			"etfsWithPrices": prices.EtfsWithPrices, "latestDay": dateStr(prices.LatestDay),
			"lastFetch": tsPtr(prices.LastFetch), "staleOrMissing": staleCodes,
		},
		"factsheets": factOut,
		"runs":       runList,
	})
}

func tsPtr(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

func dateStr(d pgtype.Date) any {
	if !d.Valid {
		return nil
	}
	return d.Time.Format("2006-01-02")
}

func uuidStr(u pgtype.UUID) string {
	return auth.Member{ID: u}.UUIDString()
}
