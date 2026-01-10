# UPSTONK API - Endpoints

## Base URL

```
http://localhost:8080
```

---

## 1. Health Check

### `GET /api/v1/health`

Check if the API is running and healthy.

**Request:**

```bash
curl http://localhost:8080/api/v1/health
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2025-01-10T14:23:47Z",
  "service": "etf-discovery-api",
  "version": "1.0.0"
}
```

---

## 2. Discover ETFs

### `POST /api/v1/discover`

Discover ETFs matching investor profile and exposure requirements.

### Example 1: Basic TFSA Discovery (South Africa)

**Request:**

```bash
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

**Response:**

```json
{
  "requestId": "req_a7f3c4d2-8e9a-4b1c-9f2d-3e4a5b6c7d8e",
  "results": [
    {
      "ticker": "STXNDQ",
      "name": "Satrix NASDAQ 100 ETF",
      "isin": "ZAE000195568",
      "exchange": "JSE",
      "provider": "Satrix",
      "assetClass": "Equity",
      "trackingIndex": "NASDAQ-100 Index",
      "geographicFocus": "United States",
      "sectorFocus": "Technology",
      "ter": 0.45,
      "aum": 2850000000,
      "currency": "ZAR",
      "averageDailyVolume": 1250000,
      "eligibility": {
        "status": "eligible",
        "isEligible": true,
        "confidence": "high",
        "justification": "✓ Listed on JSE; ✓ Currency: ZAR; ✓ Not leveraged; ✓ Not inverse; ✓ Approved provider: Satrix; ✓ Physical replication; ✓ JSE-listed by approved provider (typical TFSA eligibility path)",
        "rulesPassed": [
          "jse_listing",
          "currency_denomination",
          "no_leverage",
          "no_inverse",
          "approved_provider",
          "replication_method",
          "implicit_sars_approval"
        ],
        "rulesFailed": [],
        "warnings": [],
        "ruleVersion": "tfsa_za_v1.0_2025"
      },
      "matchScore": 92.5,
      "rankingScore": 88.3,
      "rank": 1
    }
  ],
  "summary": {
    "totalSearched": 47,
    "totalEligible": 38,
    "totalIneligible": 3,
    "totalUnknown": 6,
    "searchDurationMs": 2847,
    "dataSourcesQueried": ["JSE", "FactSheets", "Provider APIs"]
  },
  "warnings": [],
  "generatedAt": "2025-01-10T14:23:47Z",
  "cacheHit": false
}
```

---

### Example 2: Multi-Asset Portfolio with Risk Preferences

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "ZA",
      "accountType": "tfsa",
      "currency": "ZAR",
      "riskTolerance": "moderate",
      "timeHorizonYears": 20
    },
    "exposure": {
      "assets": {
        "companies": ["Apple", "Microsoft", "Tencent"],
        "sectors": ["technology", "healthcare"],
        "assetClasses": ["equity", "bonds"]
      },
      "geography": {
        "markets": ["usa", "china", "asia"],
        "emergingMarkets": true,
        "developedMarkets": true
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.75,
      "minAUM": 100000000,
      "excludeSyntheticETFs": true,
      "excludeLeveragedETFs": true,
      "excludeInverseETFs": true,
      "minLiquidity": 100000
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
      "explainEligibility": true,
      "includeWarnings": true
    }
  }'
```

---

### Example 3: Emerging Markets Focus

**Request:**

```bash
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
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["china", "india", "brazil"],
        "emergingMarkets": true,
        "developedMarkets": false,
        "excludeCountries": ["RU"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.60,
      "physicalOnly": false
    },
    "rankingPreferences": {
      "priority": ["lowest_fees", "liquidity"],
      "weighting": {
        "fees": 0.6,
        "liquidity": 0.4
      }
    },
    "outputOptions": {
      "maxResults": 5,
      "includeAlternatives": false,
      "includeSourceLinks": true,
      "explainEligibility": true
    }
  }'
```

---

### Example 4: Low-Cost Index Tracking

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "ZA",
      "accountType": "tfsa",
      "currency": "ZAR",
      "riskTolerance": "conservative",
      "timeHorizonYears": 30
    },
    "exposure": {
      "assets": {
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["usa", "global"],
        "developedMarkets": true
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.40,
      "minAUM": 500000000,
      "excludeSyntheticETFs": true,
      "excludeLeveragedETFs": true,
      "minLiquidity": 500000
    },
    "rankingPreferences": {
      "priority": ["lowest_fees", "tracking_accuracy"],
      "weighting": {
        "fees": 0.7,
        "tracking": 0.3
      }
    },
    "outputOptions": {
      "maxResults": 3,
      "includeAlternatives": true,
      "includeSourceLinks": true,
      "explainEligibility": true
    }
  }'
```

---

### Example 5: Sector-Specific Discovery

**Request:**

```bash
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
        "sectors": ["healthcare", "pharmaceuticals", "biotechnology"],
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["usa", "europe"],
        "developedMarkets": true
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true,
      "maxTER": 0.80
    },
    "rankingPreferences": {
      "priority": ["tracking_accuracy", "liquidity"],
      "weighting": {
        "fees": 0.3,
        "tracking": 0.4,
        "liquidity": 0.3
      }
    },
    "outputOptions": {
      "maxResults": 10,
      "includeAlternatives": true,
      "includeSourceLinks": true,
      "explainEligibility": true
    }
  }'
```

---

## Request Payload Reference

### InvestorProfile

| Field              | Type    | Required | Description                                              |
| ------------------ | ------- | -------- | -------------------------------------------------------- |
| `country`          | string  | Yes      | ISO 3166-1 alpha-2 country code (e.g., "ZA", "US", "GB") |
| `accountType`      | string  | Yes      | Account type (e.g., "tfsa", "ira", "isa", "standard")    |
| `currency`         | string  | Yes      | ISO 4217 currency code (e.g., "ZAR", "USD", "GBP")       |
| `riskTolerance`    | string  | No       | "conservative", "moderate", or "aggressive"              |
| `timeHorizonYears` | integer | No       | Investment time horizon (1-50 years)                     |

### Exposure

| Field                        | Type     | Description                                                 |
| ---------------------------- | -------- | ----------------------------------------------------------- |
| `assets.companies`           | string[] | Specific companies to target (e.g., ["Apple", "Microsoft"]) |
| `assets.sectors`             | string[] | Sectors (e.g., ["technology", "healthcare"])                |
| `assets.assetClasses`        | string[] | Asset classes (e.g., ["equity", "bonds", "commodities"])    |
| `assets.indices`             | string[] | Specific indices to track                                   |
| `geography.markets`          | string[] | Geographic markets (e.g., ["usa", "china", "europe"])       |
| `geography.emergingMarkets`  | boolean  | Include emerging markets                                    |
| `geography.developedMarkets` | boolean  | Include developed markets                                   |
| `geography.excludeCountries` | string[] | Countries to exclude                                        |

### Constraints

| Field                  | Type    | Description                        |
| ---------------------- | ------- | ---------------------------------- |
| `tfsaEligibleOnly`     | boolean | Only TFSA-eligible ETFs            |
| `maxTER`               | float   | Maximum Total Expense Ratio (0-5%) |
| `minAUM`               | float   | Minimum Assets Under Management    |
| `excludeSyntheticETFs` | boolean | Exclude synthetic replication      |
| `excludeLeveragedETFs` | boolean | Exclude leveraged ETFs             |
| `excludeInverseETFs`   | boolean | Exclude inverse ETFs               |
| `physicalOnly`         | boolean | Only physical replication          |
| `minLiquidity`         | float   | Minimum average daily volume       |

### RankingPreferences

| Field       | Type     | Description                                                                                               |
| ----------- | -------- | --------------------------------------------------------------------------------------------------------- |
| `priority`  | string[] | Order of importance: "lowest_fees", "tracking_accuracy", "liquidity", "diversification", "tax_efficiency" |
| `weighting` | object   | Custom weights (must sum to 1.0)                                                                          |

### OutputOptions

| Field                 | Type    | Description                            |
| --------------------- | ------- | -------------------------------------- |
| `maxResults`          | integer | Maximum results to return (1-100)      |
| `includeAlternatives` | boolean | Include alternative suggestions        |
| `includeSourceLinks`  | boolean | Include data source URLs               |
| `explainEligibility`  | boolean | Include detailed eligibility reasoning |
| `includeWarnings`     | boolean | Include warnings and caveats           |

---

## Error Responses

### 400 Bad Request - Invalid Input

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Request validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "details": "{\"country\":\"This field is required\"}"
  },
  "requestId": "req_123"
}
```

### 400 Bad Request - Unsupported Country/Account

```json
{
  "error": "UNSUPPORTED_COUNTRY",
  "message": "Country not supported",
  "code": "UNSUPPORTED_COUNTRY",
  "details": {
    "details": "account type 'ira' is not supported for country 'ZA'"
  },
  "requestId": "req_456"
}
```

### 404 Not Found - No Results

```json
{
  "error": "NO_RESULTS",
  "message": "No ETFs found matching the criteria",
  "code": "NO_RESULTS",
  "details": {
    "details": "No ETFs found matching the requested exposure criteria"
  },
  "requestId": "req_789"
}
```

### 500 Internal Server Error

```json
{
  "error": "INTERNAL_ERROR",
  "message": "An internal error occurred",
  "code": "INTERNAL_ERROR",
  "requestId": "req_abc"
}
```

### 503 Service Unavailable - Data Source Error

```json
{
  "error": "DATA_SOURCE_ERROR",
  "message": "Unable to retrieve data from external sources",
  "code": "DATA_SOURCE_ERROR",
  "details": {
    "details": "JSE API timeout"
  },
  "requestId": "req_def"
}
```

---

## Response Fields Reference

### ETFResult

| Field                | Type    | Description                                    |
| -------------------- | ------- | ---------------------------------------------- |
| `ticker`             | string  | ETF ticker symbol                              |
| `name`               | string  | Full ETF name                                  |
| `isin`               | string  | International Securities Identification Number |
| `exchange`           | string  | Exchange where listed                          |
| `provider`           | string  | ETF provider/issuer                            |
| `assetClass`         | string  | Primary asset class                            |
| `trackingIndex`      | string  | Index being tracked                            |
| `ter`                | float   | Total Expense Ratio (%)                        |
| `aum`                | float   | Assets Under Management                        |
| `currency`           | string  | Trading currency                               |
| `averageDailyVolume` | float   | Average daily trading volume                   |
| `eligibility`        | object  | Eligibility determination                      |
| `matchScore`         | float   | How well it matches criteria (0-100)           |
| `rankingScore`       | float   | Overall quality score (0-100)                  |
| `rank`               | integer | Rank in results                                |

### EligibilityDetail

| Field           | Type     | Description                                        |
| --------------- | -------- | -------------------------------------------------- |
| `status`        | string   | "eligible", "ineligible", "unknown", "conditional" |
| `isEligible`    | boolean  | Simple yes/no                                      |
| `confidence`    | string   | "high", "medium", "low", "none"                    |
| `justification` | string   | Human-readable explanation                         |
| `rulesPassed`   | string[] | Rules that passed                                  |
| `rulesFailed`   | string[] | Rules that failed                                  |
| `warnings`      | string[] | Warnings or caveats                                |
| `ruleVersion`   | string   | Version of rules used                              |

---

## Rate Limits

Currently no rate limits implemented. In production:

- **Anonymous**: 100 requests/hour
- **Authenticated**: 1000 requests/hour

---

## Support

- **GitHub Issues**: https://github.com/yourorg/upstonk/issues
- **Documentation**: See README.md
- **Email**: support@example.com
