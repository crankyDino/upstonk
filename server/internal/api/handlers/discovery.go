package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"upstonk/internal/api/dto"
	"upstonk/internal/service/discovery"
)

type DiscoveryHandler struct {
	service   *discovery.Service
	validator *validator.Validate
}

func NewDiscoveryHandler(service *discovery.Service) *DiscoveryHandler {
	return &DiscoveryHandler{
		service:   service,
		validator: validator.New(),
	}
}

// HandleDiscovery is the main endpoint: POST /api/v1/discover
func (h *DiscoveryHandler) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := uuid.New().String()

	// Add request ID to context
	ctx = context.WithValue(ctx, "requestID", requestID)

	// Set timeout for entire request
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Parse request body
	var req dto.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, requestID, http.StatusBadRequest, "INVALID_JSON",
			"Failed to parse request body", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		validationErrors := h.formatValidationErrors(err)
		h.respondError(w, requestID, http.StatusBadRequest, "VALIDATION_ERROR",
			"Request validation failed", validationErrors)
		return
	}

	// Business validation
	if err := h.validateBusinessRules(&req); err != nil {
		h.respondError(w, requestID, http.StatusBadRequest, "INVALID_REQUEST",
			err.Error(), "")
		return
	}

	// Execute discovery
	result, err := h.service.DiscoverETFs(ctx, req)
	if err != nil {
		h.handleServiceError(w, requestID, err)
		return
	}

	// Build response
	response := dto.DiscoveryResponse{
		RequestID:    requestID,
		Results:      result.Results,
		Alternatives: result.Alternatives,
		Summary:      result.Summary,
		Warnings:     result.Warnings,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		CacheHit:     result.CacheHit,
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *DiscoveryHandler) validateBusinessRules(req *dto.DiscoveryRequest) error {
	// Check if account type is supported for country
	if !h.isAccountTypeSupported(req.InvestorProfile.Country, req.InvestorProfile.AccountType) {
		return fmt.Errorf("account type '%s' is not supported for country '%s'",
			req.InvestorProfile.AccountType, req.InvestorProfile.Country)
	}

	// Validate investment vehicles
	validVehicles := map[string]bool{"etf": true, "stock": true, "bond": true, "fund": true}
	for _, vehicle := range req.InvestmentVehicles {
		if !validVehicles[vehicle] {
			return fmt.Errorf("unsupported investment vehicle: %s", vehicle)
		}
	}

	// Validate that at least some exposure is specified
	hasExposure := len(req.Exposure.Assets.Companies) > 0 ||
		len(req.Exposure.Assets.Sectors) > 0 ||
		len(req.Exposure.Assets.AssetClasses) > 0 ||
		len(req.Exposure.Geography.Markets) > 0

	if !hasExposure {
		return fmt.Errorf("at least one exposure criterion must be specified")
	}

	// Validate ranking weights sum to 1.0 if provided
	if len(req.RankingPreferences.Weighting) > 0 {
		sum := 0.0
		for _, weight := range req.RankingPreferences.Weighting {
			sum += weight
		}
		if sum < 0.99 || sum > 1.01 {
			return fmt.Errorf("ranking weights must sum to 1.0, got %.2f", sum)
		}
	}

	// Set defaults for output options
	if req.OutputOptions.MaxResults == 0 {
		req.OutputOptions.MaxResults = 10
	}

	return nil
}

func (h *DiscoveryHandler) isAccountTypeSupported(country, accountType string) bool {
	// Registry of supported account types per country
	supportedCombinations := map[string]map[string]bool{
		"ZA": {
			"tfsa":     true,
			"standard": true,
			"ira":      false,
		},
		"US": {
			"ira":      true,
			"roth_ira": true,
			"401k":     true,
			"standard": true,
		},
		"GB": {
			"isa":      true,
			"standard": true,
		},
	}

	countryMap, exists := supportedCombinations[country]
	if !exists {
		return false
	}

	return countryMap[accountType]
}

func (h *DiscoveryHandler) handleServiceError(w http.ResponseWriter, requestID string, err error) {
	// Type-switch on error to provide specific error codes
	switch err.(type) {
	case *discovery.NoResultsError:
		h.respondError(w, requestID, http.StatusNotFound, "NO_RESULTS",
			"No ETFs found matching the criteria", err.Error())
	case *discovery.UnsupportedCountryError:
		h.respondError(w, requestID, http.StatusBadRequest, "UNSUPPORTED_COUNTRY",
			"Country not supported", err.Error())
	case *discovery.DataSourceError:
		h.respondError(w, requestID, http.StatusServiceUnavailable, "DATA_SOURCE_ERROR",
			"Unable to retrieve data from external sources", err.Error())
	default:
		h.respondError(w, requestID, http.StatusInternalServerError, "INTERNAL_ERROR",
			"An internal error occurred", "Please try again later")
	}
}

func (h *DiscoveryHandler) formatValidationErrors(err error) string {

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	errorMessages := make(map[string]string)
	for _, e := range validationErrors {

		field := e.Field()
		switch e.Tag() {
		case "required":
			errorMessages[field] = "This field is required"
		case "min":
			errorMessages[field] = fmt.Sprintf("Must be at least %s", e.Param())
		case "max":
			errorMessages[field] = fmt.Sprintf("Must be at most %s", e.Param())
		case "oneof":
			errorMessages[field] = fmt.Sprintf("Must be one of: %s", e.Param())
		default:
			errorMessages[field] = fmt.Sprintf("Validation failed on '%s'", e.Tag())
		}
	}

	result, _ := json.Marshal(errorMessages)
	return string(result)
}

func (h *DiscoveryHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *DiscoveryHandler) respondError(w http.ResponseWriter, requestID string, status int, code, message, details string) {
	errorResp := dto.ErrorResponse{
		Error:     code,
		Message:   message,
		Code:      code,
		RequestID: requestID,
	}

	if details != "" {
		errorResp.Details = map[string]string{"details": details}
	}

	h.respondJSON(w, status, errorResp)
}

// HandleHealth is a simple health check endpoint
func (h *DiscoveryHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "upstonk-api",
		"version":   "1.0.0",
	}
	h.respondJSON(w, http.StatusOK, response)
}

// HandleTopPerformers returns top performing stocks/ETFs based on asset class or investment vehicle
// GET /api/v1/discover/{type} where type is assetClass (equity, bond) or investmentVehicle (etf, stock)
func (h *DiscoveryHandler) HandleTopPerformers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := uuid.New().String()
	ctx = context.WithValue(ctx, "requestID", requestID)

	// Get type from URL path
	vars := mux.Vars(r)
	assetType := vars["type"]
	if assetType == "" {
		h.respondError(w, requestID, http.StatusBadRequest, "INVALID_PARAMETER",
			"Type parameter is required", "Use: /api/v1/discover/{assetClass or investmentVehicle}")
		return
	}

	// Determine if it's an asset class or investment vehicle
	var assetClass string
	var investmentVehicles []string

	assetTypeLower := strings.ToLower(assetType)
	switch assetTypeLower {
	case "equity", "equities", "stock", "stocks":
		assetClass = "equity"
		investmentVehicles = []string{"etf", "stock"}
	case "bond", "bonds", "fixed income":
		assetClass = "bond"
		investmentVehicles = []string{"etf", "bond"}
	case "etf", "etfs":
		investmentVehicles = []string{"etf"}
		assetClass = "equity" // Default for ETFs
	default:
		h.respondError(w, requestID, http.StatusBadRequest, "INVALID_TYPE",
			fmt.Sprintf("Unknown type: %s", assetType),
			"Valid types: equity, bond, etf, stock")
		return
	}

	// Build discovery request
	req := dto.DiscoveryRequest{
		InvestorProfile: dto.InvestorProfile{
			Country:     "US", // Default to US for global data
			AccountType: "standard",
			Currency:    "USD",
		},
		Exposure: dto.ExposureRequest{
			Assets: dto.AssetExposureRequest{
				AssetClasses: []string{assetClass},
			},
		},
		InvestmentVehicles: investmentVehicles,
		Constraints:        dto.Constraints{},
		OutputOptions: dto.OutputOptions{
			MaxResults:         20,
			IncludeSourceLinks: true,
		},
	}

	// Execute discovery
	result, err := h.service.DiscoverETFs(ctx, req)
	if err != nil {
		h.handleServiceError(w, requestID, err)
		return
	}

	// Build response
	response := dto.DiscoveryResponse{
		RequestID:    requestID,
		Results:      result.Results,
		Alternatives: result.Alternatives,
		Summary:      result.Summary,
		Warnings:     result.Warnings,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		CacheHit:     result.CacheHit,
	}

	h.respondJSON(w, http.StatusOK, response)
}
