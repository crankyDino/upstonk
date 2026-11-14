// Package router builds the HTTP handler tree for the API.
package router

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/middleware"
)

// New returns the fully wired API handler: global middleware, v1 routes and
// the documentation page.
func New(discoveryHandler *handlers.DiscoveryHandler) http.Handler {
	r := mux.NewRouter()

	// Global middleware
	r.Use(middleware.RequestLogger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS)
	r.Use(middleware.Timeout(30 * time.Second))

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/discover", discoveryHandler.HandleDiscovery).Methods("POST", "OPTIONS")
	v1.HandleFunc("/discover/{type}", discoveryHandler.HandleTopPerformers).Methods("GET")
	v1.HandleFunc("/health", discoveryHandler.HandleHealth).Methods("GET")

	r.HandleFunc("/", serveDocumentation).Methods("GET")

	return r
}
