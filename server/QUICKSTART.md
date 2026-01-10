# Quick Start Guide

## Initial Setup

### 1. Run the setup script

```bash
chmod +x setup.sh
./setup.sh
```

This will create the complete directory structure and configuration files.

### 2. Place the source files

Copy each artifact into its corresponding location:

```
upstonk/
├── cmd/server/main.go                          ← main_server artifact
├── internal/
│   ├── api/
│   │   ├── dto/
│   │   │   └── request.go                      ← api_dtos artifact
│   │   ├── handlers/
│   │   │   └── discovery.go                    ← discovery_handler artifact
│   │   └── middleware/
│   │       └── (middleware.go)                 ← from interfaces_config
│   ├── domain/
│   │   └── etf.go                              ← etf_domain_models artifact
│   ├── service/
│   │   ├── discovery/
│   │   │   └── discovery.go                    ← discovery_service artifact
│   │   ├── eligibility/
│   │   │   ├── rules/
│   │   │   │   └── tfsa_za.go                  ← eligibility_engine artifact
│   │   │   └── (engine.go + interface.go)      ← from interfaces_config
│   │   ├── ranking/
│   │   │   └── (scorer.go + interface.go)      ← from interfaces_config
│   │   └── search/
│   │       └── (interface.go)                  ← from interfaces_config
│   └── config/
│       └── config.go                           ← from interfaces_config
├── go.mod                                      ← go_mod_file artifact
└── README.md                                   ← readme_doc artifact
```

### 3. Fix Import Paths

All imports should use `upstonk/` prefix instead of `upstonk/`:

**Example in cmd/server/main.go:**
```go
import (
    "upstonk/internal/api/handlers"
    "upstonk/internal/config"
    // ... etc
)
```

### 4. Install Dependencies

```bash
go mod tidy
```

This should now work without errors!

## Running the Application

### Option 1: Direct Run

```bash
# Set environment variables
export SERVER_ADDRESS=":8080"
export ENVIRONMENT="development"

# Run
go run cmd/server/main.go
```

### Option 2: Build and Run

```bash
make build
./bin/upstonk
```

### Option 3: Docker

```bash
make docker-build
make docker-run
```

## Testing the API

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Discover ETFs
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
```

## Stub Implementation Note

Since we don't have actual JSE API access yet, you'll need to create stub implementations:

### Create internal/service/search/stub.go

```go
package search

import (
    "context"
    "upstonk/internal/domain"
    "time"
)

// StubProvider provides mock ETF data for development
type StubProvider struct{}

func NewStubProvider() Provider {
    return &StubProvider{}
}

func (s *StubProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
    // Return mock ETFs for testing
    return []domain.ETF{
        {
            Ticker: "STXNDQ",
            Name: "Satrix NASDAQ 100 ETF",
            ISIN: "ZAE000195568",
            Exchange: "JSE",
            ExchangeCountry: "ZA",
            Provider: "Satrix",
            Currency: "ZAR",
            IsPhysical: true,
            TER: 0.45,
            AUM: 2850000000,
            AssetClass: "Equity",
            TrackingIndex: "NASDAQ-100",
            InceptionDate: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
            DataSources: []domain.DataSource{
                {
                    Type: "Mock",
                    Provider: "StubProvider",
                    AccessDate: time.Now(),
                    Reliability: "Primary",
                },
            },
        },
    }, nil
}
```

### Update cmd/server/main.go initializeSearchProvider:

```go
func initializeSearchProvider(cfg *config.Config) search.Provider {
    // Use stub provider for development
    return search.NewStubProvider()
}
```

## Common Issues

### Issue: "cannot find package"
**Solution:** Ensure all import paths use `upstonk/` prefix

### Issue: "undefined: search.NewAggregatedProvider"
**Solution:** Use `search.NewStubProvider()` until real providers are implemented

### Issue: Port 8080 already in use
**Solution:** Change `SERVER_ADDRESS` environment variable to `:8081` or another port

## Next Steps

1. ✅ Get basic server running with stub data
2. Implement real JSE API client
3. Add fact sheet scraping
4. Implement caching layer
5. Add comprehensive tests
6. Set up CI/CD pipeline

## Support

If you encounter issues:
1. Check that all files are in correct locations
2. Verify import paths use `upstonk/` prefix
3. Run `go mod tidy` to ensure dependencies are correct
4. Check logs for specific error messages