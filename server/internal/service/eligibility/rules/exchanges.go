package rules

import "strings"

// UK "recognised stock exchange" designations (Income Tax Act 2007 s1005),
// from HMRC's tables, accessed 2026-11-15:
// https://www.gov.uk/government/publications/recognised-stock-exchanges-section-1005-income-tax-act-2007/shorter-tables
// Keys are the exchange codes our data sources report (Yahoo exchangeName and
// plain names). See DOCS/research/etf-eligibility-isa-ira-401k.md §1.2.

// usNationalExchanges are SEC-registered national securities exchanges, which
// HMRC designates as recognised; their main markets are "listed".
var usNationalExchanges = map[string]bool{
	"NYQ": true, "NYSE": true, // NYSE
	"PCX": true, "NYSEARCA": true, "NYSE ARCA": true, // NYSE Arca
	"ASE": true, "NYSEAMERICAN": true, "NYSE AMERICAN": true, // NYSE American
	"NMS": true, "NGM": true, "NCM": true, "NAS": true, "NASDAQ": true, // Nasdaq tiers
	"BTS": true, "BATS": true, "CBOE BZX": true, // Cboe BZX (formerly BATS)
}

// notRecognisedMarkets are markets HMRC expressly says are not recognised
// stock exchanges (OTC Bulletin Board, Pink Sheets).
var notRecognisedMarkets = map[string]bool{
	"PNK": true, "OTC": true, "OBB": true, "OQB": true, "OQX": true, "OTCBB": true, "PINK": true,
}

// segmentAmbiguous are recognised exchanges where only some market segments
// count as "listed" (e.g. LSE Main Market yes, AIM no), so the exchange code
// alone doesn't settle it.
var segmentAmbiguous = map[string]bool{
	"LSE": true, "LON": true, "IOB": true, // London Stock Exchange (Main Market vs AIM)
	"GER": true, "FRA": true, "ETR": true, "XETR": true, // German exchanges (Regulierter Markt vs Freiverkehr)
	"PAR": true, "ISE": true, "DUB": true, // Euronext Paris / Dublin (regulated vs Growth/Access)
}

type listing int

const (
	listingUnknown listing = iota
	listingRecognised
	listingSegmentAmbiguous
	listingNotRecognised
)

func classifyListing(exchange string) listing {
	e := strings.ToUpper(strings.TrimSpace(exchange))
	switch {
	case e == "":
		return listingUnknown
	case usNationalExchanges[e]:
		return listingRecognised
	case notRecognisedMarkets[e]:
		return listingNotRecognised
	case segmentAmbiguous[e]:
		return listingSegmentAmbiguous
	}
	return listingUnknown
}

func isUSListing(exchange string) bool {
	return usNationalExchanges[strings.ToUpper(strings.TrimSpace(exchange))]
}
