package usage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCriticalNoBlockPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rawURL      string
		title       string
		mainContent string
		expected    bool
	}{
		{
			name:     "detects payment from url path",
			rawURL:   "https://example.com/checkout/confirm",
			expected: true,
		},
		{
			name:     "detects reservation from query",
			rawURL:   "https://example.com/travel?step=reservation_confirm",
			expected: true,
		},
		{
			name:     "detects booking from title",
			rawURL:   "https://example.com/flow",
			title:    "Finalize Booking",
			expected: true,
		},
		{
			name:        "detects invoice from content",
			rawURL:      "https://example.com/flow",
			mainContent: "Please complete payment for invoice #1234",
			expected:    true,
		},
		{
			name:     "does not match hostname only",
			rawURL:   "https://booking.com/",
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
			actual := isCriticalNoBlockPage(tc.rawURL, tc.title, tc.mainContent)
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
