package router

import "net/http"

func serveDocumentation(w http.ResponseWriter, r *http.Request) {
	documentation := `
<!DOCTYPE html>
<html>
<head>
    <title>ETF Discovery API</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 50px auto; padding: 20px; }
        h1 { color: #2c3e50; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { background: #e8f5e9; padding: 15px; margin: 20px 0; border-left: 4px solid #4caf50; }
    </style>
</head>
<body>
    <h1>ETF Discovery API v1.0</h1>
    <p>API for discovering eligible ETFs based on portfolio exposure requirements. Full documentation lives in <code>DOCS/</code> in the repository.</p>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <h3>POST /api/v1/discover</h3>
        <p>Discover ETFs matching your criteria</p>
        <p><strong>Request:</strong> See example payload in documentation</p>
        <p><strong>Response:</strong> Ranked list of eligible ETFs with eligibility justifications</p>
    </div>
    
    <div class="endpoint">
        <h3>GET /api/v1/health</h3>
        <p>Health check endpoint</p>
    </div>

    <h2>Supported Countries & Account Types</h2>
    <ul>
        <li><strong>South Africa (ZA)</strong>: TFSA (rule TFSA_ZA), Standard</li>
        <li><strong>United States (US)</strong>: IRA and Roth IRA (rule IRA_US), 401(k) (rule 401K_US), Standard</li>
        <li><strong>United Kingdom (GB)</strong>: ISA (rule ISA_UK), Standard</li>
    </ul>
    <p>Standard accounts are eligible by default. Where eligibility depends on your account
    provider or plan, results are <code>conditional</code> (or <code>unknown</code> for 401(k) plans).</p>

    <h2>Design Principles</h2>
    <ul>
        <li><strong>Never Hallucinate Eligibility</strong>: Unknown eligibility is marked as such</li>
        <li><strong>Source Attribution</strong>: All data includes provenance</li>
        <li><strong>Explainability</strong>: Clear reasoning for all eligibility decisions</li>
        <li><strong>Fail-Safe Compliance</strong>: Conservative approach to regulatory compliance</li>
    </ul>

    <h2>Example Request</h2>
    <pre>
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
    </pre>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(documentation))
}
