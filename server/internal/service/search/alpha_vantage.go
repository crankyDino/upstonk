package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"upstonk/internal/domain"
)

// AlphaVantageProvider uses Alpha Vantage API for ETF data
type AlphaVantageProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewAlphaVantageProvider(apiKey string) *AlphaVantageProvider {
	if apiKey == "" {
		apiKey = "demo" // Alpha Vantage provides a demo key
	}

	return &AlphaVantageProvider{
		apiKey:  apiKey,
		baseURL: "https://www.alphavantage.co/query",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search implements the Provider interface
func (p *AlphaVantageProvider) Search(ctx context.Context, criteria Criteria) ([]domain.ETF, error) {
	// Get relevant tickers based on criteria
	tickers := p.getTickersForCriteria(criteria)

	etfs := make([]domain.ETF, 0)

	// Fetch each ETF (limited by API rate limits)
	for i, ticker := range tickers {
		// Alpha Vantage free tier: 25 requests/day, so be conservative
		if i >= 10 {
			break
		}

		etf, err := p.GetETFProfile(ctx, ticker)
		if err != nil {
			continue // Skip failed fetches
		}

		etfs = append(etfs, etf)
	}

	return etfs, nil
}

// getTickersForCriteria maps criteria to known ETF tickers
func (p *AlphaVantageProvider) getTickersForCriteria(criteria Criteria) []string {
	tickers := make([]string, 0)

	// Map sectors to ETFs
	for _, sector := range criteria.Sectors {
		switch strings.ToLower(sector) {
		case "technology":
			tickers = append(tickers, "QQQ", "XLK", "VGT")
		case "healthcare":
			tickers = append(tickers, "XLV", "VHT")
		case "financial", "financials":
			tickers = append(tickers, "XLF", "VFH")
		case "energy":
			tickers = append(tickers, "XLE", "VDE")
		}
	}

	// Map markets to ETFs
	for _, market := range criteria.Markets {
		switch strings.ToLower(market) {
		case "usa", "us", "united states":
			tickers = append(tickers, "SPY", "VOO", "VTI")
		case "emerging", "emerging markets":
			tickers = append(tickers, "EEM", "VWO")
		case "international", "world":
			tickers = append(tickers, "VEU", "VXUS")
		case "europe":
			tickers = append(tickers, "VGK", "EZU")
		case "china":
			tickers = append(tickers, "FXI", "MCHI")
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

// GetETFProfile fetches detailed ETF information from Alpha Vantage
func (p *AlphaVantageProvider) GetETFProfile(ctx context.Context, ticker string) (domain.ETF, error) {
	// Fetch overview
	overview, err := p.fetchOverview(ctx, ticker)
	if err != nil {
		return domain.ETF{}, err
	}

	// Parse numeric values from strings
	ter, _ := strconv.ParseFloat(overview.NetExpenseRatio, 64)
	aum, _ := strconv.ParseFloat(overview.NetAssets, 64)

	etf := domain.ETF{
		Ticker:          ticker,
		Name:            overview.Name,
		Exchange:        overview.Exchange,
		ExchangeCountry: p.getCountryFromExchange(overview.Exchange),
		Currency:        overview.Currency,
		AssetClass:      overview.AssetType,
		Provider:        overview.FundFamily,
		TER:             ter * 100, // Convert to percentage
		AUM:             aum,
		InceptionDate:   p.parseDate(overview.InceptionDate),
		IsLeveraged:     strings.ToUpper(overview.Leveraged) == "YES",
		IsInverse:       false, // Not provided by Alpha Vantage
		IsPhysical:      true,  // Assume physical unless stated otherwise
		DataSources: []domain.DataSource{
			{
				Type:        "API",
				Provider:    "Alpha Vantage",
				URL:         fmt.Sprintf("https://www.alphavantage.co/query?function=ETF_PROFILE&symbol=%s", ticker),
				AccessDate:  time.Now(),
				Reliability: "Primary",
			},
		},
		LastUpdated: time.Now(),
	}

	// Add sector allocations
	for _, sector := range overview.Sectors {
		weight, _ := strconv.ParseFloat(sector.Weight, 64)
		if weight > 0 {
			etf.SectorExposure = append(etf.SectorExposure, domain.SectorAllocation{
				Sector:     sector.Sector,
				Percentage: weight * 100, // Convert to percentage
			})
		}
	}

	// Add top holdings
	for _, holding := range overview.Holdings {
		weight, _ := strconv.ParseFloat(holding.Weight, 64)
		if weight > 0 && holding.Description != "n/a" {
			etf.TopHoldings = append(etf.TopHoldings, domain.Holding{
				Name:   holding.Description,
				Ticker: holding.Symbol,
				Weight: weight * 100, // Convert to percentage
			})
		}
	}

	return etf, nil
}

func (p *AlphaVantageProvider) fetchOverview(ctx context.Context, ticker string) (AlphaVantageOverview, error) {
	url := fmt.Sprintf("%s?function=ETF_PROFILE&symbol=%s&apikey=%s",
		p.baseURL, ticker, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return AlphaVantageOverview{}, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return AlphaVantageOverview{}, err
	}
	defer resp.Body.Close()

	var result AlphaVantageOverview
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return AlphaVantageOverview{}, err
	}

	return result, nil
}

func (p *AlphaVantageProvider) fetchQuote(ctx context.Context, ticker string) (AlphaVantageQuote, error) {
	url := fmt.Sprintf("%s?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		p.baseURL, ticker, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return AlphaVantageQuote{}, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return AlphaVantageQuote{}, err
	}
	defer resp.Body.Close()

	var result struct {
		GlobalQuote AlphaVantageQuote `json:"Global Quote"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return AlphaVantageQuote{}, err
	}

	return result.GlobalQuote, nil
}

func (p *AlphaVantageProvider) getCountryFromExchange(exchange string) string {
	exchangeMap := map[string]string{
		"NYSE": "US", "NASDAQ": "US", "AMEX": "US",
		"LSE": "GB",
		"JSE": "ZA",
		"TSE": "JP",
		"FRA": "DE",
	}

	if country, ok := exchangeMap[exchange]; ok {
		return country
	}
	return "US" // Default
}

func (p *AlphaVantageProvider) parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

type AlphaVantageOverview struct {
	Symbol            string `json:"symbol"`
	Name              string `json:"name"`
	Exchange          string `json:"exchange"`
	Currency          string `json:"currency"`
	AssetType         string `json:"asset_type"`
	FundFamily        string `json:"fund_family"`
	NetAssets         string `json:"net_assets"`         // String, needs conversion
	NetExpenseRatio   string `json:"net_expense_ratio"`  // String, needs conversion
	PortfolioTurnover string `json:"portfolio_turnover"` // String, needs conversion
	DividendYield     string `json:"dividend_yield"`     // String, needs conversion
	InceptionDate     string `json:"inception_date"`
	Leveraged         string `json:"leveraged"` // "YES" or "NO"
	Sectors           []struct {
		Sector string `json:"sector"`
		Weight string `json:"weight"` // String, needs conversion
	} `json:"sectors"`
	Holdings []struct {
		Symbol      string `json:"symbol"`
		Description string `json:"description"`
		Weight      string `json:"weight"` // String, needs conversion
	} `json:"holdings"`
}

type AlphaVantageQuote struct {
	Symbol string `json:"01. symbol"`
	Price  string `json:"05. price"`
	Volume int64  `json:"06. volume"`
}
