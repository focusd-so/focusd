package usage

import (
	"net/url"
	"strings"
)

var criticalPageKeywords = []string{
	"checkout",
	"payment",
	"payments",
	"pay-now",
	"paynow",
	"billing",
	"invoice",
	"invoices",
	"transaction",
	"3d-secure",
	"3ds",
	"reservation",
	"reservations",
	"booking",
	"bookings",
	"book-now",
	"booknow",
	"confirm-booking",
	"confirmation",
}

// isCriticalNoBlockPage returns true for payment/booking flows that should never be blocked.
// It intentionally checks URL path/query/fragment and available page context,
// but does not match hostname-only terms to avoid overly broad allow-listing.
func isCriticalNoBlockPage(rawURL, title, mainContent string) bool {
	if rawURL == "" && title == "" && mainContent == "" {
		return false
	}

	var urlSignals string
	if u, err := url.Parse(rawURL); err == nil {
		urlSignals = strings.ToLower(strings.Join([]string{u.Path, u.RawQuery, u.Fragment}, " "))
	}

	context := strings.ToLower(strings.Join([]string{urlSignals, title, mainContent}, " "))
	for _, keyword := range criticalPageKeywords {
		if strings.Contains(context, keyword) {
			return true
		}
	}

	return false
}
