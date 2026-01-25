#!/bin/bash

# Test script for live ETF data fetching

API_URL="http://localhost:8080"

echo "🧪 Testing ETF Discovery API with Live Data"
echo "==========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo "Test 1: Health Check"
echo "--------------------"
response=$(curl -s "${API_URL}/api/v1/health")
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ API is running${NC}"
    echo "$response" | jq '.'
else
    echo -e "${RED}✗ API is not responding${NC}"
    exit 1
fi
echo ""

# Test 2: Technology ETFs (USA)
echo "Test 2: Fetching US Technology ETFs"
echo "------------------------------------"
response=$(curl -s -X POST "${API_URL}/api/v1/discover" \
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
      "tfsaEligibleOnly": false,
      "maxTER": 1.0
    },
    "outputOptions": {
      "maxResults": 5,
      "includeSourceLinks": true,
      "explainEligibility": false
    }
  }')

if [ $? -eq 0 ]; then
    result_count=$(echo "$response" | jq '.results | length')
    if [ "$result_count" -gt 0 ]; then
        echo -e "${GREEN}✓ Found $result_count ETFs${NC}"
        echo ""
        echo "Sample Result:"
        echo "$response" | jq '.results[0] | {ticker, name, ter, provider, dataSources: .dataSources | map(.provider)}'
        echo ""
        echo "Data Sources Used:"
        echo "$response" | jq -r '.summary.dataSourcesQueried[]' | while read source; do
            echo "  - $source"
        done
    else
        echo -e "${YELLOW}⚠ No results found${NC}"
        echo "$response" | jq '.warnings'
    fi
else
    echo -e "${RED}✗ Request failed${NC}"
fi
echo ""

# Test 3: Emerging Markets ETFs
echo "Test 3: Fetching Emerging Markets ETFs"
echo "--------------------------------------"
response=$(curl -s -X POST "${API_URL}/api/v1/discover" \
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
        "markets": ["emerging markets"],
        "emergingMarkets": true
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "maxTER": 1.0
    },
    "outputOptions": {
      "maxResults": 3,
      "includeSourceLinks": true
    }
  }')

if [ $? -eq 0 ]; then
    result_count=$(echo "$response" | jq '.results | length')
    if [ "$result_count" -gt 0 ]; then
        echo -e "${GREEN}✓ Found $result_count ETFs${NC}"
        echo ""
        echo "Results:"
        echo "$response" | jq '.results[] | {ticker, name, aum}'
    else
        echo -e "${YELLOW}⚠ No results found${NC}"
    fi
else
    echo -e "${RED}✗ Request failed${NC}"
fi
echo ""

# Test 4: Cache Performance
echo "Test 4: Testing Cache Performance"
echo "----------------------------------"
echo "First request (uncached):"
start_time=$(date +%s%N)
curl -s -X POST "${API_URL}/api/v1/discover" \
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
    "outputOptions": {
      "maxResults": 5
    }
  }' > /dev/null
end_time=$(date +%s%N)
first_request_time=$(( (end_time - start_time) / 1000000 ))
echo "Time: ${first_request_time}ms"

echo ""
echo "Second request (should be cached):"
start_time=$(date +%s%N)
response=$(curl -s -X POST "${API_URL}/api/v1/discover" \
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
    "outputOptions": {
      "maxResults": 5
    }
  }')
end_time=$(date +%s%N)
second_request_time=$(( (end_time - start_time) / 1000000 ))
echo "Time: ${second_request_time}ms"

cache_hit=$(echo "$response" | jq -r '.cacheHit')
if [ "$cache_hit" = "true" ]; then
    speedup=$(( first_request_time / second_request_time ))
    echo -e "${GREEN}✓ Cache working! ${speedup}x faster${NC}"
else
    echo -e "${YELLOW}⚠ Cache miss (might not be implemented yet)${NC}"
fi
echo ""

# Summary
echo "==========================================="
echo "✅ Live Data Integration Test Complete"
echo ""
echo "Recommendations:"
echo "1. Set ALPHA_VANTAGE_API_KEY for better data"
echo "2. Monitor data source reliability"
echo "3. Check logs for any API errors"
echo ""
echo "Get free Alpha Vantage key: https://www.alphavantage.co/support/#api-key"