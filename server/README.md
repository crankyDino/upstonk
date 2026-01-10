# UPSTONK API

Production-ready REST API for discovering eligible ETFs based on configurable portfolio exposure requirements. Built for serious retail investors and fintech platforms.

## 🎯 Core Features

- **Intelligent ETF Search**: Multi-source aggregation from exchanges, fact sheets, and financial data providers
- **Explicit Eligibility Rules**: Codified, versioned compliance rules (TFSA, ISA, IRA support)
- **Never Hallucinates**: Unknown eligibility is marked as such, with confidence levels
- **Source Attribution**: Every data point includes provenance and references
- **Explainable Rankings**: Transparent weighted scoring with customizable preferences
- **Production-Ready**: Comprehensive error handling, timeouts, logging, and graceful shutdown

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     HTTP Layer                          │
│              (Handlers, Validation, DTOs)               │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Service Layer                          │
│         (ETF Discovery, Orchestration Logic)            │
└──┬──────────────┬──────────────┬────────────────────┬──┘
   │              │              │                    │
┌──▼──────┐  ┌───▼──────┐  ┌───▼──────────┐  ┌─────▼─────┐
│ Search  │  │Eligibility│  │   Ranking    │  │   Cache   │
│ Service │  │  Rules    │  │   Engine     │  │  Service  │
└──┬──────┘  └───┬──────┘  └───┬──────────┘  └─────┬─────┘
   │              │              │                    │
└──────────────── Data Access Layer ──────────────────────┘
```

### Design Principles

1. **Fail-Safe Compliance**: Conservative approach to regulatory rules
2. **Modularity**: Pluggable search providers, eligibility rules, and ranking algorithms
3. **Testability**: Clear interfaces and dependency injection
4. **Observability**: Comprehensive logging and request tracking

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Access to JSE API (for South African ETFs)

### Installation

```bash
# Clone repository
git clone https://upstonk-api.git
cd upstonk-api

# Install dependencies
go mod download

# Set environment variables
export SERVER_ADDRESS=":8080"
export ENVIRONMENT="development"
export JSE_API_KEY="your_api_key"

# Run the server
go run cmd/server/main.go
```

### Docker Deployment

```bash
docker build -t upstonk-api .
docker run -p 8080:8080 \
  -e JSE_API_KEY="your_key" \
  upstonk-api
```

## 📡 API Endpoints

### `POST /api/v1/discover`

Discover ETFs matching portfolio exposure requirements.

**Request Body:**

```json
{
  "investorProfile": {
    "country": "ZA",
    "accountType": "tfsa",
    "currency": "ZAR",
    "riskTolerance": "moderate",
    "timeHorizonYears": 20
  },
  "exposure": {
    "assets": {
      "companies": ["Apple", "Microsoft"],
      "sectors": ["technology", "healthcare"],
      "assetClasses": ["equity"]
    },
    "geography": {
      "markets": ["usa", "china"],
      "emergingMarkets": true,
      "developedMarkets": true
    }
  },
  "investmentVehicles": ["etf"],
  "constraints": {
    "tfsaEligibleOnly": true,
    "maxTER": 0.75,
    "excludeSyntheticETFs": true,
    "excludeLeveragedETFs": true
  },
  "rankingPreferences": {
    "priority": ["lowest_fees", "tracking_accuracy", "liquidity"],
    "weighting": {
      "fees": 0.4,
      "tracking": 0.3,
      "liquidity": 0.2,
      "diversification": 0.1
    }
  },
  "outputOptions": {
    "maxResults": 10,
    "includeAlternatives": true,
    "includeSourceLinks": true,
    "explainEligibility": true
  }
}
```

**Response:**

See `example_response.json` for complete response structure.

### `GET /api/v1/health`

Health check endpoint.

## 🔐 Eligibility Rules

### South Africa TFSA Rules (tfsa_za_v1.0_2025)

Based on Income Tax Act 1962, Section 12T:

1. ✅ **JSE Listing**: Must be listed on Johannesburg Stock Exchange
2. ✅ **Currency**: ZAR or approved foreign currencies (USD for select ETFs)
3. ✅ **No Leverage**: Leveraged ETFs prohibited
4. ✅ **No Inverse**: Inverse ETFs prohibited
5. ✅ **Approved Provider**: From recognized SA ETF providers
6. ⚠️ **Replication**: Physical preferred, synthetic requires verification

**Confidence Levels:**

- **High**: All criteria verified from primary sources
- **Medium**: Some criteria verified, minor gaps
- **Low**: Significant data missing
- **None**: Insufficient data for determination

### Adding New Country Rules

```go
// Implement the Rule interface
type MyCountryRules struct {
    version string
}

func (r *MyCountryRules) AppliesTo(country, accountType string) bool {
    return country == "XX" && accountType == "my_account"
}

func (r *MyCountryRules) Evaluate(ctx context.Context, etf domain.ETF) domain.EligibilityResult {
    // Implement eligibility logic
}

// Register in main.go
engine.RegisterRule(NewMyCountryRules())
```

## 📊 Data Sources

### Primary Sources

- **JSE Exchange Listings**: Real-time ETF data
- **Provider Fact Sheets**: Satrix, CoreShares, Cloud Atlas, Sygnia
- **Index Providers**: MSCI, S&P, FTSE

### Secondary Sources

- **Financial Data APIs**: Bloomberg, Reuters (future)
- **Regulatory Filings**: SARS documentation

### Caching Strategy

- ETF data: 1 hour TTL
- Eligibility rules: No expiry (versioned)
- Search results: 15 minutes TTL

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Example Test

```go
func TestTFSAEligibility(t *testing.T) {
    rules := rules.NewTFSASouthAfricaRules()

    etf := domain.ETF{
        Ticker:   "STXNDQ",
        Exchange: "JSE",
        Currency: "ZAR",
        Provider: "Satrix",
        IsLeveraged: false,
    }

    result := rules.Evaluate(context.Background(), etf)

    assert.True(t, result.IsEligible)
    assert.Equal(t, domain.StatusEligible, result.Status)
    assert.Equal(t, domain.ConfidenceHigh, result.Confidence)
}
```

## 🔧 Configuration

Environment variables:

| Variable           | Description                    | Default                    |
| ------------------ | ------------------------------ | -------------------------- |
| `SERVER_ADDRESS`   | Server bind address            | `:8080`                    |
| `ENVIRONMENT`      | Environment (dev/staging/prod) | `development`              |
| `JSE_API_KEY`      | JSE API key                    | -                          |
| `JSE_API_BASE_URL` | JSE API endpoint               | `https://api.jse.co.za/v1` |
| `CACHE_ENABLED`    | Enable result caching          | `true`                     |
| `LOG_LEVEL`        | Logging level                  | `info`                     |
| `LOG_FORMAT`       | Log format (json/text)         | `json`                     |

## 🎯 Roadmap

- [ ] Additional country support (US, UK, EU)
- [ ] Real-time price data integration
- [ ] Portfolio optimization suggestions
- [ ] Tax-loss harvesting recommendations
- [ ] GraphQL API
- [ ] WebSocket streaming for price updates
- [ ] Machine learning for ETF recommendations

## 🛡️ Security Considerations

1. **API Rate Limiting**: Implement per-client rate limits
2. **Authentication**: Add API key/OAuth for production
3. **Input Validation**: Comprehensive validation at all layers
4. **Data Sanitization**: Prevent injection attacks
5. **HTTPS Only**: Enforce TLS in production

## 📄 License

MIT License - See LICENSE file

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📞 Support

- **Documentation**: https://docs.etfdiscovery.com
- **Issues**: https://upstonk-api/issues
- **Email**: support@etfdiscovery.com

## ⚠️ Disclaimer

This API provides information for research purposes only. Always verify ETF eligibility with your financial institution and consult a qualified financial advisor before making investment decisions. The authors are not responsible for investment losses.

---

**Built with ❤️ for the South African investment community**
