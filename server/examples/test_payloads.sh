#!/bin/bash

# Example 1: US ETFs - Emerging Markets (should return live data from Yahoo Finance)
echo "=== Example 1: US ETFs - Emerging Markets ==="
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "US",
      "accountType": "standard",
      "currency": "USD"
    },
    "exposure": {
      "assets": {
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["emerging markets", "china", "india"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "allowedExchanges": ["NYSE", "NASDAQ"],
      "maxTER": 1.0
    },
    "outputOptions": {
      "maxResults": 5,
      "includeSourceLinks": true
    }
  }' | jq

echo -e "\n\n=== Example 2: US ETFs - Technology Sector ==="
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "US",
      "accountType": "standard",
      "currency": "USD"
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
      "maxTER": 0.5
    },
    "outputOptions": {
      "maxResults": 10,
      "includeSourceLinks": true
    }
  }' | jq

echo -e "\n\n=== Example 3: ZA JSE ETFs - Emerging Markets (if Yahoo Finance has JSE data) ==="
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
        "markets": ["emerging markets", "china", "india"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": false,
      "allowedExchanges": ["JSE"]
    },
    "outputOptions": {
      "maxResults": 5,
      "includeSourceLinks": true
    }
  }' | jq

echo -e "\n\n=== Example 4: All Exchanges - No Exchange Filter ==="
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "investorProfile": {
      "country": "US",
      "accountType": "standard",
      "currency": "USD"
    },
    "exposure": {
      "assets": {
        "assetClasses": ["equity"]
      },
      "geography": {
        "markets": ["usa"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {},
    "outputOptions": {
      "maxResults": 5,
      "includeSourceLinks": true
    }
  }' | jq
