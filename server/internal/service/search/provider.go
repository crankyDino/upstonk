package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"upstonk/internal/domain"
)

// LiveProvider fetches real ETF data from public sources
type LiveProvider struct {
	httpClient *http.Client
	userAgent  string
}

func NewLiveProvider() Provider {
	return &LiveProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Mozilla/5.0 (compatible; ETFDiscoveryBot/1.0)",
	}
}

func (p *LiveProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	etfs := make([]domain.ETF, 0)

	// Strategy: Search multiple sources and aggregate results
	// For ZA country, prioritize JSE-listed ETFs

	// 1. Search JSE website for South African ETFs (always search if ZA)
	if criteria.Country == "ZA" || contains(criteria.Markets, "south africa") {
		jseETFs, err := p.searchJSE(ctx, criteria)
		if err != nil {
			log.Printf("JSE search error: %v", err)
		} else {
			log.Printf("JSE search found %d ETFs", len(jseETFs))
			etfs = append(etfs, jseETFs...)
		}
	}

	// 2. For ZA country, don't search global ETFs - only JSE-listed ETFs are eligible
	// Only search global sources if not ZA
	if criteria.Country != "ZA" {
		// Search ETF.com API for global ETFs
		globalETFs, err := p.searchETFDotCom(ctx, criteria)
		if err == nil {
			etfs = append(etfs, globalETFs...)
		}

		// Search Yahoo Finance for ETF data
		yahooETFs, err := p.searchYahooFinance(ctx, criteria)
		if err == nil {
			etfs = append(etfs, yahooETFs...)
		}
	}

	// Deduplicate by ISIN/Ticker
	etfs = p.deduplicate(etfs)

	log.Printf("Total ETFs found before filtering: %d", len(etfs))

	// Filter by criteria
	filtered := make([]domain.ETF, 0)
	for _, etf := range etfs {
		if p.matchesCriteria(etf, criteria) {
			log.Printf("ETF %s (%s) matches criteria", etf.Ticker, etf.Exchange)
			filtered = append(filtered, etf)
		}
	}

	log.Printf("Total ETFs after filtering: %d", len(filtered))
	return filtered, nil
}

// searchJSE searches for JSE-listed ETFs using Yahoo Finance
// JSE ETFs on Yahoo Finance use .JO suffix (e.g., STXEMG.JO)
func (p *LiveProvider) searchJSE(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	etfs := make([]domain.ETF, 0)

	// Get JSE ETF tickers based on criteria
	jseTickers := p.getJSETickersForCriteria(criteria)
	log.Printf("Searching for JSE ETFs with tickers: %v", jseTickers)

	if len(jseTickers) == 0 {
		log.Printf("No JSE tickers found for criteria: markets=%v, assetClasses=%v", criteria.Markets, criteria.AssetClasses)
		return etfs, nil
	}

	// Fetch each JSE ETF from Yahoo Finance
	for _, ticker := range jseTickers {
		// Add .JO suffix for JSE listings on Yahoo Finance
		yahooTicker := ticker + ".JO"
		log.Printf("Fetching JSE ETF: %s (Yahoo ticker: %s)", ticker, yahooTicker)
		
		etf, err := p.fetchYahooFinanceETF(ctx, yahooTicker)
		if err != nil {
			log.Printf("Failed to fetch %s: %v", yahooTicker, err)
			// Skip failed fetches, continue with others
			continue
		}

		// Ensure exchange info is set correctly for JSE (override any Yahoo Finance data)
		etf.Exchange = "JSE"
		etf.ExchangeCountry = "ZA"
		if etf.Currency == "" {
			etf.Currency = "ZAR"
		}

		// Add JSE data source
		etf.DataSources = append(etf.DataSources, domain.DataSource{
			Type:        "ExchangeListing",
			Provider:    "JSE",
			URL:         fmt.Sprintf("https://www.jse.co.za/trade/etfs"),
			AccessDate:  time.Now(),
			Reliability: "Primary",
		})

		log.Printf("Successfully fetched JSE ETF: %s (%s) - Exchange: %s, Currency: %s", etf.Ticker, etf.Name, etf.Exchange, etf.Currency)
		etfs = append(etfs, etf)
	}

	log.Printf("JSE search completed: found %d ETFs", len(etfs))
	return etfs, nil
}

// searchETFDotCom searches ETF.com's public API
func (p *LiveProvider) searchETFDotCom(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	// Build search query
	query := p.buildSearchQuery(criteria)

	// ETF.com screener endpoint (public API)
	apiURL := fmt.Sprintf("https://www.etf.com/api/screener?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ETF.com request failed: %d", resp.StatusCode)
	}

	var result ETFComResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return p.convertETFComToETFs(result), nil
}

// searchYahooFinance uses Yahoo Finance API
func (p *LiveProvider) searchYahooFinance(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	etfs := make([]domain.ETF, 0)

	// Get common ETF tickers based on criteria
	tickers := p.getETFTickersForCriteria(criteria)

	for _, ticker := range tickers {
		etf, err := p.fetchYahooFinanceETF(ctx, ticker)
		if err != nil {
			continue // Skip failed fetches
		}
		etfs = append(etfs, etf)
	}

	return etfs, nil
}

// fetchYahooFinanceETF fetches a single ETF from Yahoo Finance
func (p *LiveProvider) fetchYahooFinanceETF(ctx context.Context, ticker string) (domain.ETF, error) {
	// Yahoo Finance quote endpoint
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s", ticker)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return domain.ETF{}, err
	}
	req.Header.Set("User-Agent", p.userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return domain.ETF{}, err
	}
	defer resp.Body.Close()

	var result YahooFinanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.ETF{}, err
	}

	if len(result.QuoteResponse.Result) == 0 {
		return domain.ETF{}, fmt.Errorf("no data for ticker %s", ticker)
	}

	quote := result.QuoteResponse.Result[0]

	// Fetch additional details from Yahoo Finance
	details, _ := p.fetchYahooETFDetails(ctx, ticker)

	// Extract base ticker (remove exchange suffix like .JO)
	baseTicker := quote.Symbol
	if strings.Contains(baseTicker, ".") {
		parts := strings.Split(baseTicker, ".")
		baseTicker = parts[0]
	}

	// Determine exchange and country
	exchange := quote.Exchange
	exchangeCountry := p.getCountryFromExchange(quote.Exchange)

	// Handle JSE tickers specifically
	if strings.HasSuffix(ticker, ".JO") {
		exchange = "JSE"
		exchangeCountry = "ZA"
	}

	return domain.ETF{
		Ticker:             baseTicker,
		Name:               quote.LongName,
		Exchange:           exchange,
		ExchangeCountry:    exchangeCountry,
		Currency:           quote.Currency,
		TER:                details.ExpenseRatio * 100, // Convert to percentage
		AUM:                float64(quote.MarketCap),
		AverageDailyVolume: float64(quote.AverageDailyVolume3Month),
		Provider:           details.FundFamily,
		AssetClass:         details.AssetClass,
		TrackingIndex:      details.Index,
		IsPhysical:         !details.IsSynthetic,
		IsSynthetic:        details.IsSynthetic,
		IsLeveraged:        strings.Contains(strings.ToLower(quote.LongName), "leveraged") || strings.Contains(strings.ToLower(quote.LongName), "2x") || strings.Contains(strings.ToLower(quote.LongName), "3x"),
		IsInverse:          strings.Contains(strings.ToLower(quote.LongName), "inverse") || strings.Contains(strings.ToLower(quote.LongName), "short"),
		GeographicExposure: details.Geography,
		SectorExposure:     details.Sectors,
		TopHoldings:        details.Holdings,
		DataSources: []domain.DataSource{
			{
				Type:        "API",
				Provider:    "Yahoo Finance",
				URL:         fmt.Sprintf("https://finance.yahoo.com/quote/%s", ticker),
				AccessDate:  time.Now(),
				Reliability: "Primary",
			},
		},
		LastUpdated: time.Now(),
	}, nil
}

// fetchYahooETFDetails fetches detailed ETF information
func (p *LiveProvider) fetchYahooETFDetails(ctx context.Context, ticker string) (ETFDetails, error) {
	// Yahoo Finance profile endpoint
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=fundProfile,topHoldings,fundPerformance", ticker)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return ETFDetails{}, err
	}
	req.Header.Set("User-Agent", p.userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ETFDetails{}, err
	}
	defer resp.Body.Close()

	var result YahooDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ETFDetails{}, err
	}

	return p.parseYahooDetails(result), nil
}

// getJSETickersForCriteria returns JSE ETF tickers based on search criteria
func (p *LiveProvider) getJSETickersForCriteria(criteria Criteria) []string {
	tickers := make([]string, 0)

	// Map markets to JSE ETF tickers
	for _, market := range criteria.Markets {
		marketLower := strings.ToLower(market)
		
		// Check for emerging markets, China, India first (most specific)
		if strings.Contains(marketLower, "emerging") || strings.Contains(marketLower, "china") || strings.Contains(marketLower, "india") {
			// JSE ETFs for emerging markets exposure
			tickers = append(tickers, "STXEMG") // Satrix MSCI Emerging Markets
			tickers = append(tickers, "COREEM") // CoreShares MSCI Emerging Markets
		}
		
		// Check for Africa/South Africa
		if strings.Contains(marketLower, "africa") || strings.Contains(marketLower, "south africa") {
			// JSE ETFs for South African exposure
			tickers = append(tickers, "STX40") // Satrix Top 40 (SA equity)
			tickers = append(tickers, "STXRES") // Satrix RESI 10
		}
		
		// Check for US markets
		if strings.Contains(marketLower, "usa") || strings.Contains(marketLower, "us") || strings.Contains(marketLower, "united states") {
			// JSE ETFs for US exposure
			tickers = append(tickers, "STXNDQ") // Satrix NASDAQ 100
			tickers = append(tickers, "STX500") // Satrix S&P 500
		}
		
		// Check for world/global markets
		if strings.Contains(marketLower, "world") || strings.Contains(marketLower, "global") {
			// JSE ETFs for global exposure
			tickers = append(tickers, "STXWDM") // Satrix MSCI World
		}
		
		// Check for European markets
		if strings.Contains(marketLower, "europe") {
			// JSE ETFs for European exposure
			tickers = append(tickers, "STXEUR") // Satrix MSCI Europe
		}
	}

	// Map asset classes
	for _, assetClass := range criteria.AssetClasses {
		assetLower := strings.ToLower(assetClass)
		if strings.Contains(assetLower, "equity") {
			// Add general equity ETFs if not already added
			if len(tickers) == 0 {
				tickers = append(tickers, "STX40") // Satrix Top 40 (SA equity)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, ticker := range tickers {
		if !seen[ticker] {
			seen[ticker] = true
			unique = append(unique, ticker)
		}
	}

	return unique
}

// Helper: Build search query from criteria
func (p *LiveProvider) buildSearchQuery(criteria Criteria) string {
	parts := make([]string, 0)

	if len(criteria.Sectors) > 0 {
		parts = append(parts, strings.Join(criteria.Sectors, " OR "))
	}

	if len(criteria.Markets) > 0 {
		parts = append(parts, strings.Join(criteria.Markets, " OR "))
	}

	return strings.Join(parts, " ")
}

// Helper: Get ETF tickers based on criteria (for non-JSE ETFs)
func (p *LiveProvider) getETFTickersForCriteria(criteria Criteria) []string {
	tickers := make([]string, 0)

	// Map sectors/markets to known ETFs
	if contains(criteria.Sectors, "technology") {
		tickers = append(tickers, "QQQ", "XLK", "VGT", "SOXX")
	}

	if contains(criteria.Markets, "usa") || contains(criteria.Markets, "us") {
		tickers = append(tickers, "SPY", "VOO", "IVV", "VTI")
	}

	if contains(criteria.Markets, "emerging") || contains(criteria.Markets, "emerging markets") {
		tickers = append(tickers, "EEM", "VWO", "IEMG")
	}

	if contains(criteria.Markets, "china") {
		tickers = append(tickers, "FXI", "MCHI", "ASHR")
	}

	if contains(criteria.Markets, "india") {
		tickers = append(tickers, "INDA", "EPI", "INDY")
	}

	if contains(criteria.Sectors, "healthcare") {
		tickers = append(tickers, "XLV", "VHT", "IHI")
	}

	// If no specific criteria matched, add popular ETFs based on asset class
	if len(tickers) == 0 {
		for _, assetClass := range criteria.AssetClasses {
			assetLower := strings.ToLower(assetClass)
			if strings.Contains(assetLower, "equity") {
				// Popular equity ETFs
				tickers = append(tickers, "SPY", "VOO", "VTI", "QQQ", "IVV")
			} else if strings.Contains(assetLower, "bond") {
				// Popular bond ETFs
				tickers = append(tickers, "AGG", "BND", "TLT", "LQD")
			}
		}
	}

	// If still no tickers, add some default popular ETFs
	if len(tickers) == 0 {
		tickers = append(tickers, "SPY", "VOO", "QQQ", "VTI", "IVV")
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, ticker := range tickers {
		if !seen[ticker] {
			seen[ticker] = true
			unique = append(unique, ticker)
		}
	}

	return unique
}

// Helper: Get country from exchange code
func (p *LiveProvider) getCountryFromExchange(exchange string) string {
	exchangeMap := map[string]string{
		"NYQ": "US", "NMS": "US", "PCX": "US", "NAS": "US", "NCM": "US", "NGM": "US",
		"LSE": "GB", "LON": "GB",
		"JSE": "ZA", "JNB": "ZA",
		"FRA": "DE", "ETR": "DE", "XETR": "DE",
		"TSE": "JP", "TYO": "JP",
		"ASX": "AU",
		"TSX": "CA",
	}

	exchangeUpper := strings.ToUpper(exchange)
	if country, ok := exchangeMap[exchangeUpper]; ok {
		return country
	}
	return "UNKNOWN"
}

// Helper: Parse Yahoo details
func (p *LiveProvider) parseYahooDetails(response YahooDetailResponse) ETFDetails {
	details := ETFDetails{
		AssetClass: "Equity", // Default
		Geography:  domain.GeographicExposure{Regions: make(map[string]float64)},
	}

	if response.QuoteSummary.Result != nil && len(response.QuoteSummary.Result) > 0 {
		result := response.QuoteSummary.Result[0]

		if result.FundProfile.FundFamily != nil {
			details.FundFamily = *result.FundProfile.FundFamily
		}

		if result.FundProfile.CategoryName != nil {
			details.AssetClass = *result.FundProfile.CategoryName
		}

		// Parse expense ratio
		if result.FundProfile.FeesExpensesInvestment != nil {
			if result.FundProfile.FeesExpensesInvestment.AnnualReportExpenseRatio != nil {
				details.ExpenseRatio = *result.FundProfile.FeesExpensesInvestment.AnnualReportExpenseRatio
			}
		}

		// Parse holdings
		if result.TopHoldings.Holdings != nil {
			for _, holding := range result.TopHoldings.Holdings {
				details.Holdings = append(details.Holdings, domain.Holding{
					Name:   holding.HoldingName,
					Ticker: holding.Symbol,
					Weight: holding.HoldingPercent * 100,
				})
			}
		}

		// Parse sector allocations
		if result.TopHoldings.SectorWeightings != nil {
			for _, sector := range result.TopHoldings.SectorWeightings {
				for sectorName, weight := range sector {
					details.Sectors = append(details.Sectors, domain.SectorAllocation{
						Sector:     sectorName,
						Percentage: weight * 100,
					})
				}
			}
		}
	}

	return details
}

// Helper: Deduplicate ETFs by ISIN or Ticker
func (p *LiveProvider) deduplicate(etfs []domain.ETF) []domain.ETF {
	seen := make(map[string]bool)
	unique := make([]domain.ETF, 0)

	for _, etf := range etfs {
		key := etf.ISIN
		if key == "" {
			key = etf.Ticker
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, etf)
		}
	}

	return unique
}

// Helper: Check if ETF matches criteria
func (p *LiveProvider) matchesCriteria(etf domain.ETF, criteria Criteria) bool {
	// Match asset classes
	if len(criteria.AssetClasses) > 0 {
		matched := false
		for _, assetClass := range criteria.AssetClasses {
			if strings.Contains(strings.ToLower(etf.AssetClass), strings.ToLower(assetClass)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Match sectors
	if len(criteria.Sectors) > 0 {
		matched := false
		for _, sector := range criteria.Sectors {
			for _, etfSector := range etf.SectorExposure {
				if strings.Contains(strings.ToLower(etfSector.Sector), strings.ToLower(sector)) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Match markets/geography
	if len(criteria.Markets) > 0 {
		matched := false
		hasGeographicData := (etf.GeographicExposure.Regions != nil && len(etf.GeographicExposure.Regions) > 0) ||
			(etf.GeographicExposure.Countries != nil && len(etf.GeographicExposure.Countries) > 0)
		
		// If no geographic data available, be lenient and allow through
		// (Yahoo Finance may not always return geographic exposure data)
		if !hasGeographicData {
			log.Printf("ETF %s has no geographic exposure data, allowing through", etf.Ticker)
			matched = true
		} else {
			for _, market := range criteria.Markets {
				marketLower := strings.ToLower(market)
				
				// Check regions
				if etf.GeographicExposure.Regions != nil {
					for region := range etf.GeographicExposure.Regions {
						regionLower := strings.ToLower(region)
						if strings.Contains(regionLower, marketLower) || strings.Contains(marketLower, regionLower) {
							matched = true
							break
						}
					}
				}
				
				// Check countries
				if !matched && etf.GeographicExposure.Countries != nil {
					for country := range etf.GeographicExposure.Countries {
						countryLower := strings.ToLower(country)
						if strings.Contains(countryLower, marketLower) || strings.Contains(marketLower, countryLower) {
							matched = true
							break
						}
					}
				}
				
				// Special handling for common market names
				if !matched {
					if (marketLower == "china" || marketLower == "cn") && etf.GeographicExposure.Countries != nil {
						if _, exists := etf.GeographicExposure.Countries["CN"]; exists {
							matched = true
						}
					}
					if (marketLower == "india" || marketLower == "in") && etf.GeographicExposure.Countries != nil {
						if _, exists := etf.GeographicExposure.Countries["IN"]; exists {
							matched = true
						}
					}
					if (marketLower == "emerging markets" || marketLower == "emerging") && etf.GeographicExposure.Regions != nil {
						for region := range etf.GeographicExposure.Regions {
							if strings.Contains(strings.ToLower(region), "emerging") {
								matched = true
								break
							}
						}
					}
					// USA/US is common - if ETF is from US exchange, likely matches
					if (marketLower == "usa" || marketLower == "us" || marketLower == "united states") {
						if etf.ExchangeCountry == "US" || strings.Contains(strings.ToUpper(etf.Exchange), "NYSE") || 
						   strings.Contains(strings.ToUpper(etf.Exchange), "NASDAQ") || 
						   strings.Contains(strings.ToUpper(etf.Exchange), "NMS") {
							matched = true
						}
					}
				}
				
				if matched {
					break
				}
			}
		}
		
		// Only reject if we have geographic data and it doesn't match
		if !matched && hasGeographicData {
			return false
		}
	}

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// Response types for external APIs

type ETFComResponse struct {
	Data []struct {
		Ticker string  `json:"ticker"`
		Name   string  `json:"name"`
		TER    float64 `json:"expense_ratio"`
	} `json:"data"`
}

type YahooFinanceResponse struct {
	QuoteResponse struct {
		Result []YahooQuote `json:"result"`
	} `json:"quoteResponse"`
}

type YahooQuote struct {
	Symbol                   string `json:"symbol"`
	LongName                 string `json:"longName"`
	Exchange                 string `json:"exchange"`
	Currency                 string `json:"currency"`
	MarketCap                int64  `json:"marketCap"`
	AverageDailyVolume3Month int64  `json:"averageDailyVolume3Month"`
}

type YahooDetailResponse struct {
	QuoteSummary struct {
		Result []struct {
			FundProfile struct {
				FundFamily             *string `json:"fundFamily"`
				CategoryName           *string `json:"categoryName"`
				FeesExpensesInvestment *struct {
					AnnualReportExpenseRatio *float64 `json:"annualReportExpenseRatio"`
				} `json:"feesExpensesInvestment"`
			} `json:"fundProfile"`
			TopHoldings struct {
				Holdings []struct {
					HoldingName    string  `json:"holdingName"`
					Symbol         string  `json:"symbol"`
					HoldingPercent float64 `json:"holdingPercent"`
				} `json:"holdings"`
				SectorWeightings []map[string]float64 `json:"sectorWeightings"`
			} `json:"topHoldings"`
		} `json:"result"`
	} `json:"quoteSummary"`
}

type ETFDetails struct {
	FundFamily   string
	AssetClass   string
	Index        string
	ExpenseRatio float64
	IsSynthetic  bool
	Geography    domain.GeographicExposure
	Sectors      []domain.SectorAllocation
	Holdings     []domain.Holding
}

func (p *LiveProvider) convertETFComToETFs(response ETFComResponse) []domain.ETF {
	etfs := make([]domain.ETF, 0, len(response.Data))
	for _, item := range response.Data {
		etfs = append(etfs, domain.ETF{
			Ticker: item.Ticker,
			Name:   item.Name,
			TER:    item.TER,
			DataSources: []domain.DataSource{
				{
					Type:        "API",
					Provider:    "ETF.com",
					AccessDate:  time.Now(),
					Reliability: "Secondary",
				},
			},
			LastUpdated: time.Now(),
		})
	}
	return etfs
}
