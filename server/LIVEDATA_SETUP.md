# Live Data Integration Setup

The API now fetches real-time ETF data from multiple public sources.

## Data Sources

### 1. **Yahoo Finance API** (Free, No API Key Required)
- **Coverage**: Global ETFs
- **Data**: Prices, volumes, holdings, sectors
- **Rate Limit**: ~2000 requests/hour
- **Reliability**: High

### 2. **Alpha Vantage API** (Free Tier Available)
- **Coverage**: US and major global ETFs
- **Data**: Detailed ETF profiles, holdings, performance
- **Free Tier**: 25 requests/day
- **Premium**: 75-1200 requests/minute
- **Sign up**: https://www.alphavantage.co/support/#api-key

### 3. **ETF.com API** (Free, Public)
- **Coverage**: 3000+ ETFs
- **Data**: Screener data, expense ratios, AUM
- **No API key required**

### 4. **JSE Website Scraping** (South Africa)
- **Coverage**: All JSE-listed ETFs
- **Data**: Listings, providers, basic info
- **Rate Limit**: Reasonable usage only

## Setup Instructions

### Step 1: Get API Keys (Optional but Recommended)

#### Alpha Vantage (Recommended for better data)
1. Visit https://www.alphavantage.co/support/#api-key
2. Enter your email
3. Receive API key instantly (free)
4. Add to `.env`:
```bash
ALPHA_VANTAGE_API_KEY=your_key_here
```

**Note**: Without an API key, the system uses "demo" which has limited data.

### Step 2: Update Environment Variables

```bash
# .env file
SERVER_ADDRESS=:8080
ENVIRONMENT=production

# Alpha Vantage API (get free key at alphavantage.co)
ALPHA_VANTAGE_API_KEY=your_actual_key_here

# JSE API (if you have access)
JSE_API_KEY=

# Cache settings
CACHE_ENABLED=true
CACHE_TTL_SECONDS=3600

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Step 3: Install and Run

```bash
# Install dependencies
go mod tidy

# Build
go build -o bin/upstonk cmd/server/main.go

# Run
./bin/upstonk
```

Or simply:
```bash
go run cmd/server/main.go
```

## How It Works

### Multi-Source Aggregation

The API uses an intelligent aggregation strategy:

```
Request → AggregatedProvider
            ├─→ LiveProvider (Yahoo Finance + ETF.com + JSE)
            ├─→ AlphaVantageProvider (detailed ETF data)
            └─→ StubProvider (fallback for dev)
                    ↓
              Parallel Fetch
                    ↓
              Deduplicate & Merge
                    ↓
              Cache Results (1 hour TTL)
                    ↓
              Return to Client
```

### Data Merging Strategy

When multiple sources return the same ETF:
1. **Deduplicate** by ticker/ISIN
2. **Merge fields** preferring:
   - Primary sources over secondary
   - More complete data over sparse
   - Recent data over old
3. **Combine holdings** from all sources
4. **Aggregate sectors** and geographies

### Caching

- **TTL**: 1 hour (configurable)
- **Strategy**: In-memory cache per criteria
- **Key**: Country + sectors + markets
- **Benefits**: Reduces API calls, faster responses

## Testing Live Data

### Example 1: Technology ETFs

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
      "includeSourceLinks": true,
      "explainEligibility": true
    }
  }'
```

**Expected**: Live data from Yahoo Finance, Alpha Vantage for QQQ, XLK, VGT, SOXX, etc.

### Example 2: Emerging Markets

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
        "markets": ["emerging markets", "china", "india"]
      }
    },
    "investmentVehicles": ["etf"],
    "constraints": {
      "tfsaEligibleOnly": true
    },
    "outputOptions": {
      "maxResults": 5,
      "includeSourceLinks": true
    }
  }'
```

**Expected**: Live data for EEM, VWO, IEMG, etc.

## Verifying Live Data

Check the response for `dataSources` field:

```json
{
  "results": [
    {
      "ticker": "QQQ",
      "name": "Invesco QQQ Trust",
      "dataSources": [
        {
          "type": "API",
          "provider": "Yahoo Finance",
          "url": "https://finance.yahoo.com/quote/QQQ",
          "date": "2025-01-10",
          "reliability": "Primary"
        },
        {
          "type": "API",
          "provider": "Alpha Vantage",
          "reliability": "Primary"
        }
      ]
    }
  ]
}
```

## Rate Limits & Best Practices

### Alpha Vantage Free Tier
- **Limit**: 25 requests/day
- **Strategy**: Cache aggressively (1 hour TTL)
- **Upgrade**: $50/month for 75 req/min

### Yahoo Finance
- **Limit**: ~2000/hour (unofficial)
- **Strategy**: Reasonable spacing, respect robots.txt
- **Fallback**: Always have Alpha Vantage or stub

### Caching Strategy
```bash
# First request: Fetches from all sources (slow: 2-5 seconds)
# Subsequent requests: Served from cache (fast: <50ms)
# Cache expires: After 1 hour
```

## Troubleshooting

### Issue: "No results found"
**Cause**: API rate limits exceeded or network issues
**Solution**:
1. Check logs for specific errors
2. Verify API keys are set
3. Wait a few minutes and retry
4. Use stub provider as fallback

### Issue: "Incomplete data"
**Cause**: Some sources may be down or rate limited
**Solution**: API automatically falls back to other sources

### Issue: "Slow responses"
**Cause**: First request fetches from all sources
**Solution**:
- Enable caching (default: on)
- Results are cached for 1 hour
- Pre-warm cache for common queries

### Issue: Alpha Vantage 429 errors
**Cause**: Exceeded 25 requests/day on free tier
**Solution**:
- Wait until next day
- Upgrade to premium tier
- Rely on Yahoo Finance (no key needed)

## Production Deployment

### Recommended Configuration

```bash
# Production .env
ENVIRONMENT=production
SERVER_ADDRESS=:8080

# Always use real API keys in production
ALPHA_VANTAGE_API_KEY=premium_key_here

# Enable caching
CACHE_ENABLED=true
CACHE_TTL_SECONDS=3600

# Production logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Monitoring

Add monitoring for:
- API response times per source
- Cache hit rates
- Error rates per provider
- Rate limit warnings

### Scaling Considerations

1. **Add Redis for distributed caching**
2. **Implement request queuing** for rate limits
3. **Add more data sources** for redundancy
4. **Use CDN** for static ETF data
5. **Implement circuit breakers** for failing sources

## Data Source Comparison

| Source | Coverage | Rate Limit | Cost | Data Quality |
|--------|----------|------------|------|--------------|
| Yahoo Finance | Global | 2000/hr | Free | High |
| Alpha Vantage | Global | 25/day free | $50/mo | Very High |
| ETF.com | 3000+ ETFs | Unlimited | Free | Medium |
| JSE Scraping | SA only | Reasonable | Free | High |
| Stub (Dev) | Limited | Unlimited | Free | Low |

## Next Steps

1. ✅ API now pulls live data
2. ⏳ Add more specialized providers (Morningstar, Bloomberg)
3. ⏳ Implement Redis caching for production
4. ⏳ Add rate limit handling with exponential backoff
5. ⏳ Create admin dashboard for monitoring data sources
6. ⏳ Implement webhook updates for real-time data changes

## Support

If you encounter issues:
1. Check API key is valid
2. Review logs: `tail -f logs/app.log`
3. Test individual sources
4. Open an issue with error details