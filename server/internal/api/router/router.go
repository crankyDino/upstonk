// Package router builds the HTTP handler tree for the API.
package router

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/middleware"
	"upstonk/internal/auth"
)

type options struct {
	gate *auth.Gate
}

// Option configures the router.
type Option func(*options)

// WithAuth protects every API route except /api/v1/health with Supabase
// Auth and membership checks. Without it the API is open (local use).
func WithAuth(g *auth.Gate) Option { return func(o *options) { o.gate = g } }

// New returns the fully wired API handler: global middleware, v1 routes and
// the documentation page.
func New(discoveryHandler *handlers.DiscoveryHandler, opts ...Option) http.Handler {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	r := mux.NewRouter()

	// Global middleware
	r.Use(middleware.RequestLogger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS)
	r.Use(middleware.Timeout(30 * time.Second))

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/health", discoveryHandler.HandleHealth).Methods("GET")

	// Everything else needs a signed-in member when auth is on.
	protected := v1.NewRoute().Subrouter()
	if o.gate != nil {
		protected.Use(o.gate.Require)
		protected.HandleFunc("/me", handlers.HandleMe).Methods("GET")
		membership := handlers.NewMembershipHandler(o.gate.Queries())
		protected.HandleFunc("/invitations", membership.HandleCreateInvitation).Methods("POST")
		protected.HandleFunc("/notice", membership.HandleNotice).Methods("GET")

		admin := protected.PathPrefix("/admin").Subrouter()
		admin.Use(auth.RequireAdmin)
		admin.HandleFunc("/settings", membership.HandleGetSettings).Methods("GET")
		admin.HandleFunc("/settings", membership.HandlePutSettings).Methods("PUT")
		admin.HandleFunc("/users", membership.HandleListUsers).Methods("GET")
		admin.HandleFunc("/users/{id}", membership.HandlePatchUser).Methods("PATCH")
		admin.HandleFunc("/invitations", membership.HandleListInvitations).Methods("GET")
		admin.HandleFunc("/invitations/{id}", membership.HandleRevokeInvitation).Methods("DELETE")
		admin.HandleFunc("/data-status", membership.HandleDataStatus).Methods("GET")
	}
	protected.HandleFunc("/discover", discoveryHandler.HandleDiscovery).Methods("POST", "OPTIONS")
	protected.HandleFunc("/discover/{type}", discoveryHandler.HandleTopPerformers).Methods("GET")

	r.HandleFunc("/", serveDocumentation).Methods("GET")

	return r
}
