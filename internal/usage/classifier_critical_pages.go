// Critical no-block safeguards for transaction-like pages.
//
// Why this exists:
// Blocking the wrong page can break real-world user actions with consequences, such as
// failed payments, lost checkout sessions, duplicate charges, expired 3DS challenges,
// or interrupted booking/reservation confirmations. Those are high-cost interruptions,
// so we bias toward safety and avoid classifying such flows as distracting.
//
// How it works:
//  1. Deterministic URL/provider signals catch high-confidence payment and booking flows
//     (e.g. known gateway hosts and checkout/billing/confirmation paths) and force neutral.
//  2. Suspicious context detection marks ambiguous cases so callers can apply a fail-safe
//     neutral outcome when LLM classification is uncertain (error/low confidence).
//
// This keeps strict protection where it matters while reducing false positives from weak
// keyword-only title/content matches.
package usage

import (
	"net/url"
	"strings"
)

var deterministicCriticalHosts = []string{
	"checkout.stripe.com",
	"billing.stripe.com",
	"checkout.paypal.com",
	"checkout.adyen.com",
	"secure.adyen.com",
	"pay.braintreegateway.com",
	"secure.authorize.net",
	"accept.authorize.net",
	"checkout.klarna.com",
	"pay.klarna.com",
	"checkout.affirm.com",
	"pay.affirm.com",
	"checkout.afterpay.com",
	"pay.afterpay.com",
	"cash.app",
	"square.link",
	"pay.shopify.com",
	"checkout.shopify.com",
}

var deterministicCriticalPathTokens = []string{
	"checkout",
	"checkoutnow",
	"payment",
	"payments",
	"pay",
	"paynow",
	"billing",
	"invoice",
	"invoices",
	"transaction",
	"3dsecure",
	"3ds",
	"book",
	"booking",
	"bookings",
	"reservation",
	"reservations",
	"confirm",
	"confirmation",
}

var suspiciousCriticalTokens = []string{
	"checkout",
	"payment",
	"payments",
	"pay",
	"billing",
	"invoice",
	"invoices",
	"transaction",
	"3ds",
	"reservation",
	"reservations",
	"booking",
	"bookings",
	"confirmation",
}

var suspiciousCriticalPhrases = []string{
	"complete payment",
	"secure checkout",
	"confirm payment",
	"confirm your booking",
	"complete your booking",
	"3d secure",
	"reservation confirmation",
	"booking confirmation",
	"pay now",
}

var pathBasedProviderSignals = map[string][]string{
	"paypal.com":   {"checkout", "checkoutnow", "webscr", "billing", "invoice", "pay", "subscription"},
	"stripe.com":   {"checkout", "billing", "payment", "invoice", "portal", "pay"},
	"adyen.com":    {"checkout", "payment", "pay"},
	"braintree":    {"checkout", "payment", "pay"},
	"klarna.com":   {"checkout", "payment", "pay"},
	"affirm.com":   {"checkout", "payment", "pay"},
	"afterpay.com": {"checkout", "payment", "pay"},
	"squareup.com": {"checkout", "payment", "pay", "invoice"},
	"shopify.com":  {"checkout", "payment", "billing"},
}

func isDeterministicCriticalNoBlockURL(rawURL *url.URL) bool {
	if rawURL == nil {
		return false
	}

	host := strings.ToLower(strings.TrimPrefix(rawURL.Hostname(), "www."))
	if host == "" {
		return false
	}

	if hostMatchesAny(host, deterministicCriticalHosts) {
		return true
	}

	pathSignals := normalizeForMatch(strings.Join([]string{rawURL.Path, rawURL.RawQuery, rawURL.Fragment}, " "))
	for providerSignal, tokens := range pathBasedProviderSignals {
		if strings.Contains(host, providerSignal) && hasAnyToken(pathSignals, tokens) {
			return true
		}
	}

	if hasAtLeastNTokens(pathSignals, []string{"checkout", "payment", "billing", "invoice", "3ds", "reservation", "booking", "confirm", "confirmation"}, 2) {
		return true
	}

	if hasAnyToken(pathSignals, deterministicCriticalPathTokens) {
		if hostMatchesAny(host, []string{"paypal.com", "stripe.com", "adyen.com", "braintreegateway.com", "klarna.com", "affirm.com", "afterpay.com", "squareup.com", "shopify.com"}) {
			return true
		}

		for _, indicator := range []string{"pay", "checkout", "billing", "booking", "reservation", "invoice"} {
			if strings.Contains(host, indicator) {
				return true
			}
		}
	}

	return false
}

func isSuspiciousCriticalContext(u *url.URL, title, mainContent string) bool {
	if u == nil && title == "" && mainContent == "" {
		return false
	}

	var urlSignals string
	if u != nil {
		urlSignals = strings.Join([]string{u.Path, u.RawQuery, u.Fragment}, " ")
	}

	normalized := normalizeForMatch(strings.Join([]string{urlSignals, title, mainContent}, " "))
	if hasAnyPhrase(normalized, suspiciousCriticalPhrases) {
		return true
	}

	return hasAtLeastNTokens(normalized, suspiciousCriticalTokens, 2)
}

func hostMatchesAny(host string, candidates []string) bool {
	for _, candidate := range candidates {
		if host == candidate || strings.HasSuffix(host, "."+candidate) {
			return true
		}
	}

	return false
}

func normalizeForMatch(input string) string {
	input = strings.ToLower(input)

	return strings.Join(strings.FieldsFunc(input, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}), " ")
}

func hasAnyToken(normalized string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(" "+normalized+" ", " "+token+" ") {
			return true
		}
	}

	return false
}

func hasAtLeastNTokens(normalized string, tokens []string, required int) bool {
	count := 0
	for _, token := range tokens {
		if strings.Contains(" "+normalized+" ", " "+token+" ") {
			count++
			if count >= required {
				return true
			}
		}
	}

	return false
}

func hasAnyPhrase(normalized string, phrases []string) bool {
	for _, phrase := range phrases {
		normalizedPhrase := normalizeForMatch(phrase)
		if normalizedPhrase == "" {
			continue
		}

		if strings.Contains(" "+normalized+" ", " "+normalizedPhrase+" ") {
			return true
		}
	}

	return false
}
