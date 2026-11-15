package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/router"
	"upstonk/internal/app"
	"upstonk/internal/config"
	"upstonk/internal/service/discovery"
	"upstonk/internal/store"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Optional Postgres store: migrate at start-up, then serve ZA from it.
	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		pool, err = store.Open(ctx, cfg.DatabaseURL)
		if err == nil {
			err = store.Migrate(ctx, pool)
		}
		cancel()
		if err != nil {
			log.Fatalf("Database: %v", err)
		}
		defer pool.Close()
	}

	// Initialize services
	searchProvider := app.NewSearchProvider(cfg.AlphaVantageKey, pool)
	eligibilityEngine := app.NewEligibilityEngine()
	rankingEngine := app.NewRankingEngine()

	discoveryService := discovery.NewService(
		searchProvider,
		eligibilityEngine,
		rankingEngine,
	)

	// Initialize handlers
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)

	// Setup router
	apiHandler := router.New(discoveryHandler)

	// Create server
	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      apiHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting ETF Discovery API on %s", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	gracefulShutdown(server)
}

func gracefulShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
