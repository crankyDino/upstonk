package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"upstonk/internal/domain"
)

// TFSASouthAfricaRules implements eligibility rules for SA TFSA accounts
// Based on: Income Tax Act 1962, Section 12T
// Reference: https://www.sars.gov.za/types-of-tax/personal-income-tax/tax-free-savings-and-investment-account/
type TFSASouthAfricaRules struct {
	version string
}

func NewTFSASouthAfricaRules() *TFSASouthAfricaRules {
	return &TFSASouthAfricaRules{
		version: "tfsa_za_v1.0_2025",
	}
}

func (r *TFSASouthAfricaRules) Name() string {
	return "TFSA_ZA"
}

func (r *TFSASouthAfricaRules) Version() string {
	return r.version
}

func (r *TFSASouthAfricaRules) AppliesTo(country, accountType string) bool {
	return strings.ToUpper(country) == "ZA" && strings.ToLower(accountType) == "tfsa"
}

// Evaluate performs comprehensive TFSA eligibility check
func (r *TFSASouthAfricaRules) Evaluate(ctx context.Context, etf domain.ETF) domain.EligibilityResult {
	result := domain.EligibilityResult{
		RuleVersion:  r.version,
		EvaluatedAt:  time.Now(),
		Status:       domain.StatusEligible,
		IsEligible:   true,
		Confidence:   domain.ConfidenceHigh,
		RulesPassed:  []string{},
		RulesFailed:  []string{},
		RulesSkipped: []string{},
		Evidence:     []domain.EligibilityEvidence{},
		Reasons:      []string{},
	}

	// Rule 1: Must be JSE-listed
	r.checkJSEListing(&result, etf)

	// Rule 2: Must be ZAR-denominated or approved foreign currency
	r.checkCurrency(&result, etf)

	// Rule 3: Must not be leveraged or inverse
	r.checkETFStructure(&result, etf)

	// Rule 4: Must be from approved provider
	r.checkProvider(&result, etf)

	// Rule 5: Must be approved by SARS (implicitly via JSE listing + provider)
	r.checkImplicitApproval(&result, etf)

	// Rule 6: Check for synthetic replication concerns
	r.checkReplication(&result, etf)

	// Determine final status
	r.finalizeResult(&result, etf)

	return result
}

func (r *TFSASouthAfricaRules) checkJSEListing(result *domain.EligibilityResult, etf domain.ETF) {
	criterion := "jse_listing"
	expected := "Listed on JSE (Johannesburg Stock Exchange)"
	actual := fmt.Sprintf("Exchange: %s, Country: %s", etf.Exchange, etf.ExchangeCountry)

	isJSE := strings.ToUpper(etf.Exchange) == "JSE" ||
		strings.ToUpper(etf.ExchangeCountry) == "ZA" ||
		strings.Contains(strings.ToUpper(etf.Exchange), "JOHANNESBURG")

	evidence := domain.EligibilityEvidence{
		Criterion: criterion,
		Expected:  expected,
		Actual:    actual,
		Result:    "fail",
	}

	// Find supporting data source
	for _, ds := range etf.DataSources {
		if ds.Type == "ExchangeListing" {
			evidence.DataSource = ds
			break
		}
	}

	if isJSE {
		evidence.Result = "pass"
		result.RulesPassed = append(result.RulesPassed, criterion)
		result.Reasons = append(result.Reasons, "✓ Listed on JSE")
	} else {
		evidence.Result = "fail"
		result.RulesFailed = append(result.RulesFailed, criterion)
		result.Reasons = append(result.Reasons, "✗ Not listed on JSE - TFSA requires JSE-listed instruments")
		result.IsEligible = false
	}

	result.Evidence = append(result.Evidence, evidence)
}

func (r *TFSASouthAfricaRules) checkCurrency(result *domain.EligibilityResult, etf domain.ETF) {
	criterion := "currency_denomination"
	expected := "ZAR (South African Rand) or USD for approved foreign ETFs"
	actual := fmt.Sprintf("Currency: %s", etf.Currency)

	approvedCurrencies := map[string]bool{
		"ZAR": true,
		"USD": true, // Some JSE-listed ETFs are USD-denominated
	}

	evidence := domain.EligibilityEvidence{
		Criterion: criterion,
		Expected:  expected,
		Actual:    actual,
	}

	if approvedCurrencies[strings.ToUpper(etf.Currency)] {
		evidence.Result = "pass"
		result.RulesPassed = append(result.RulesPassed, criterion)
		result.Reasons = append(result.Reasons, fmt.Sprintf("✓ Currency: %s", etf.Currency))
	} else if etf.Currency == "" {
		evidence.Result = "unknown"
		result.RulesSkipped = append(result.RulesSkipped, criterion)
		result.Confidence = domain.ConfidenceMedium
		result.Reasons = append(result.Reasons, "⚠ Currency not specified - manual verification required")
	} else {
		evidence.Result = "fail"
		result.RulesFailed = append(result.RulesFailed, criterion)
		result.Reasons = append(result.Reasons, fmt.Sprintf("✗ Currency %s may not be TFSA-eligible", etf.Currency))
		result.IsEligible = false
	}

	result.Evidence = append(result.Evidence, evidence)
}

func (r *TFSASouthAfricaRules) checkETFStructure(result *domain.EligibilityResult, etf domain.ETF) {
	// Check leveraged
	leveragedCriterion := "no_leverage"
	if etf.IsLeveraged {
		result.RulesFailed = append(result.RulesFailed, leveragedCriterion)
		result.Reasons = append(result.Reasons, "✗ Leveraged ETFs are not permitted in TFSAs")
		result.IsEligible = false
		result.Evidence = append(result.Evidence, domain.EligibilityEvidence{
			Criterion: leveragedCriterion,
			Expected:  "Non-leveraged ETF",
			Actual:    "Leveraged ETF detected",
			Result:    "fail",
		})
	} else {
		result.RulesPassed = append(result.RulesPassed, leveragedCriterion)
		result.Reasons = append(result.Reasons, "✓ Not leveraged")
	}

	// Check inverse
	inverseCriterion := "no_inverse"
	if etf.IsInverse {
		result.RulesFailed = append(result.RulesFailed, inverseCriterion)
		result.Reasons = append(result.Reasons, "✗ Inverse ETFs are not permitted in TFSAs")
		result.IsEligible = false
		result.Evidence = append(result.Evidence, domain.EligibilityEvidence{
			Criterion: inverseCriterion,
			Expected:  "Standard tracking ETF",
			Actual:    "Inverse ETF detected",
			Result:    "fail",
		})
	} else {
		result.RulesPassed = append(result.RulesPassed, inverseCriterion)
		result.Reasons = append(result.Reasons, "✓ Not inverse")
	}
}

func (r *TFSASouthAfricaRules) checkProvider(result *domain.EligibilityResult, etf domain.ETF) {
	criterion := "approved_provider"

	// Known TFSA-approved providers in South Africa
	approvedProviders := map[string]bool{
		"satrix":       true,
		"coreshares":   true,
		"1nvest":       true,
		"cloud atlas":  true,
		"absa":         true,
		"standardbank": true,
		"sygnia":       true,
		"ashburton":    true,
	}

	providerLower := strings.ToLower(etf.Provider)

	// Check if provider is in approved list
	isApproved := false
	for approved := range approvedProviders {
		if strings.Contains(providerLower, approved) {
			isApproved = true
			break
		}
	}

	evidence := domain.EligibilityEvidence{
		Criterion: criterion,
		Expected:  "Recognized SA ETF provider",
		Actual:    fmt.Sprintf("Provider: %s", etf.Provider),
	}

	if isApproved {
		evidence.Result = "pass"
		result.RulesPassed = append(result.RulesPassed, criterion)
		result.Reasons = append(result.Reasons, fmt.Sprintf("✓ Approved provider: %s", etf.Provider))
	} else if etf.Provider == "" {
		evidence.Result = "unknown"
		result.RulesSkipped = append(result.RulesSkipped, criterion)
		result.Confidence = domain.ConfidenceLow
		result.Reasons = append(result.Reasons, "⚠ Provider not identified - verification required")
	} else {
		evidence.Result = "unknown"
		result.RulesSkipped = append(result.RulesSkipped, criterion)
		result.Confidence = domain.ConfidenceMedium
		result.Reasons = append(result.Reasons, fmt.Sprintf("⚠ Provider '%s' not in known approved list - verify with platform", etf.Provider))
	}

	result.Evidence = append(result.Evidence, evidence)
}

func (r *TFSASouthAfricaRules) checkImplicitApproval(result *domain.EligibilityResult, etf domain.ETF) {
	// In South Africa, JSE listing + local provider generally implies TFSA eligibility
	// However, we should be cautious

	hasJSE := false
	for _, rule := range result.RulesPassed {
		if rule == "jse_listing" {
			hasJSE = true
			break
		}
	}

	hasProvider := false
	for _, rule := range result.RulesPassed {
		if rule == "approved_provider" {
			hasProvider = true
			break
		}
	}

	criterion := "implicit_sars_approval"
	if hasJSE && hasProvider {
		result.RulesPassed = append(result.RulesPassed, criterion)
		result.Reasons = append(result.Reasons, "✓ JSE-listed by approved provider (typical TFSA eligibility path)")
	} else {
		result.RulesSkipped = append(result.RulesSkipped, criterion)
		result.Reasons = append(result.Reasons, "⚠ Cannot confirm implicit TFSA approval - recommend platform verification")
		if result.Confidence == domain.ConfidenceHigh {
			result.Confidence = domain.ConfidenceMedium
		}
	}
}

func (r *TFSASouthAfricaRules) checkReplication(result *domain.EligibilityResult, etf domain.ETF) {
	criterion := "replication_method"

	if etf.IsSynthetic {
		result.RulesSkipped = append(result.RulesSkipped, criterion)
		result.Reasons = append(result.Reasons, "⚠ Synthetic replication - verify TFSA approval with provider")
		result.Confidence = domain.ConfidenceMedium

		result.Evidence = append(result.Evidence, domain.EligibilityEvidence{
			Criterion: criterion,
			Expected:  "Physical replication preferred",
			Actual:    "Synthetic replication",
			Result:    "unknown",
		})
	} else if etf.IsPhysical {
		result.RulesPassed = append(result.RulesPassed, criterion)
		result.Reasons = append(result.Reasons, "✓ Physical replication")
	}
}

func (r *TFSASouthAfricaRules) finalizeResult(result *domain.EligibilityResult, etf domain.ETF) {
	// Determine final status based on rules
	if len(result.RulesFailed) > 0 {
		result.Status = domain.StatusIneligible
		result.IsEligible = false
	} else if len(result.RulesSkipped) > 3 || result.Confidence == domain.ConfidenceLow {
		result.Status = domain.StatusUnknown
		result.IsEligible = false
		result.Reasons = append(result.Reasons,
			"⚠ Insufficient data to confirm eligibility - recommend verification with SARS or your platform")
	} else if len(result.RulesSkipped) > 0 {
		result.Status = domain.StatusConditional
		result.IsEligible = true // Conditional means eligible but with warnings
		result.Reasons = append(result.Reasons,
			"✓ Likely eligible but verification recommended before investing")
	} else {
		result.Status = domain.StatusEligible
		result.IsEligible = true
	}

	// Lower confidence if we have missing data
	if result.Confidence == domain.ConfidenceHigh && len(result.RulesSkipped) > 0 {
		result.Confidence = domain.ConfidenceMedium
	}
}
