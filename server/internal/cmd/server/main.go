package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"upstonk/internal/api/handlers"
	"upstonk/internal/api/middleware"
	"upstonk/internal/config"
	"upstonk/internal/service/discovery"
	"upstonk/internal/service/eligibility"
	"upstonk/internal/service/eligibility/rules"
	"upstonk/internal/service/ranking"
	"upstonk/internal/service/search"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	searchProvider := initializeSearchProvider(cfg)
	eligibilityEngine := initializeEligibilityEngine()
	rankingEngine := initializeRankingEngine()

	discoveryService := discovery.NewService(
		searchProvider,
		eligibilityEngine,
		rankingEngine,
	)

	// Initialize handlers
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)

	// Setup router
	router := setupRouter(discoveryHandler)

	// Create server
	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      router,
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

func setupRouter(discoveryHandler *handlers.DiscoveryHandler) *mux.Router {
	router := mux.NewRouter()

	// Global middleware
	router.Use(middleware.RequestLogger)
	router.Use(middleware.Recovery)
	router.Use(middleware.CORS)
	router.Use(middleware.Timeout(30 * time.Second))

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Discovery endpoint
	v1.HandleFunc("/discover", discoveryHandler.HandleDiscovery).Methods("POST", "OPTIONS")

	// Health check
	v1.HandleFunc("/health", discoveryHandler.HandleHealth).Methods("GET")

	// Documentation endpoint
	router.HandleFunc("/", serveDocumentation).Methods("GET")

	return router
}

func initializeSearchProvider(cfg *config.Config) search.Provider {
	// For development, use stub provider with mock data
	// In production, replace with real providers:
	// - JSE API client
	// - Fact sheet scrapers
	// - Financial data API clients (Bloomberg, Reuters, etc.)
	// - Caching layer

	return search.NewStubProvider()
}

func initializeEligibilityEngine() eligibility.Engine {
	engine := eligibility.NewEngine()

	// Register eligibility rules
	engine.RegisterRule(rules.NewTFSASouthAfricaRules())
	// engine.RegisterRule(rules.NewISAUKRules())
	// engine.RegisterRule(rules.NewIRAUSRules())

	return engine
}

func initializeRankingEngine() ranking.Engine {
	return ranking.NewWeightedScorer()
}

func serveDocumentation(w http.ResponseWriter, r *http.Request) {
	documentation := `
<!DOCTYPE html>
<html>
<head>
    <title>ETF Discovery API</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 50px auto; padding: 20px; }
        h1 { color: #2c3e50; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { background: #e8f5e9; padding: 15px; margin: 20px 0; border-left: 4px solid #4caf50; }
    </style>
</head>
<body>
    <h1>ETF Discovery API v1.0</h1>
    <p>Production-ready API for discovering eligible ETFs based on portfolio exposure requirements.</p>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <h3>POST /api/v1/discover</h3>
        <p>Discover ETFs matching your criteria</p>
        <p><strong>Request:</strong> See example payload in documentation</p>
        <p><strong>Response:</strong> Ranked list of eligible ETFs with eligibility justifications</p>
    </div>
    
    <div class="endpoint">
        <h3>GET /api/v1/health</h3>
        <p>Health check endpoint</p>
    </div>

    <h2>Supported Countries & Account Types</h2>
    <ul>
        <li><strong>South Africa (ZA)</strong>: TFSA, Standard</li>
        <li><strong>United States (US)</strong>: Coming soon</li>
        <li><strong>United Kingdom (GB)</strong>: Coming soon</li>
    </ul>

    <h2>Design Principles</h2>
    <ul>
        <li><strong>Never Hallucinate Eligibility</strong>: Unknown eligibility is marked as such</li>
        <li><strong>Source Attribution</strong>: All data includes provenance</li>
        <li><strong>Explainability</strong>: Clear reasoning for all eligibility decisions</li>
        <li><strong>Fail-Safe Compliance</strong>: Conservative approach to regulatory compliance</li>
    </ul>

    <h2>Example Request</h2>
    <pre>
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "ZA",
      "accountType": "tfsa",
      "currency": "ZAR"
    },
    "exposure": {
      "assets": {
        "sectors": ["technology"],
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["usa"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.75
    },
    "outputOptions": {
      "maxResults": 10,
      "explainEligibility": true
    }
  }'
    </pre>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(documentation))
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
