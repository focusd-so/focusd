package usage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsDeterministicCriticalNoBlockURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawURL   string
		expected bool
	}{
		{
			name:     "detects stripe checkout host",
			rawURL:   "https://checkout.stripe.com/c/pay/cs_test_123",
			expected: true,
		},
		{
			name:     "detects paypal checkout host",
			rawURL:   "https://checkout.paypal.com/checkoutnow?token=123",
			expected: true,
		},
		{
			name:     "detects provider path flow",
			rawURL:   "https://www.paypal.com/checkoutnow?token=abc",
			expected: true,
		},
		{
			name:     "does not match hostname only",
			rawURL:   "https://paypal.com/",
			expected: false,
		},
		{
			name:     "non critical page",
			rawURL:   "https://example.com/news",
			expected: false,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := isDeterministicCriticalNoBlockURL(tc.rawURL)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsSuspiciousCriticalContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rawURL      string
		title       string
		mainContent string
		expected    bool
	}{
		{
			name:        "detects strong phrase from content",
			rawURL:      "https://example.com/flow",
			mainContent: "Please complete payment to confirm your booking",
			expected:    true,
		},
		{
			name:     "detects multi token from title",
			rawURL:   "https://example.com/flow",
			title:    "Reservation payment confirmation",
			expected: true,
		},
		{
			name:     "does not trigger for weak single mention",
			rawURL:   "https://example.com/blog/payment-systems",
			title:    "How payment systems work",
			expected: false,
		},
		{
			name:     "non critical page",
			rawURL:   "https://example.com/news",
			title:    "Top engineering trends",
			expected: false,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := isSuspiciousCriticalContext(tc.rawURL, tc.title, tc.mainContent)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestClassifyObviouslyWebsite_CriticalNoBlockOverride(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	resp, err := svc.classifyObviouslyWebsite(context.Background(), "https://amazon.com/checkout?payment=true")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, ClassificationNeutral, resp.Classification)
	require.Equal(t, ClassificationSourceObviously, resp.ClassificationSource)
}

func TestClassifyObviouslyWebsite_StillBlocksNonCriticalShoppingPages(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	resp, err := svc.classifyObviouslyWebsite(context.Background(), "https://amazon.com/deals")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, ClassificationDistracting, resp.Classification)
	require.Equal(t, ClassificationSourceObviously, resp.ClassificationSource)
}
