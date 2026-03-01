package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/focusd-so/focusd/internal/api"
)

func TestLLMProxyHandler_RateLimiting(t *testing.T) {
	// 1. Setup Mock Gemini Server
	mockGemini := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{
								"text": "{\"classification\": \"distracting\"}"
							}
						]
					}
				}
			],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 10,
				"totalTokenCount": 110
			}
		}`))
	}))
	defer mockGemini.Close()

	// Use the mock server URL
	os.Setenv("GEMINI_BASE_URL", mockGemini.URL)
	defer os.Unsetenv("GEMINI_BASE_URL")

	// Set PASETO key for minting tokens
	os.Setenv("PASETO_KEYS", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	defer os.Unsetenv("PASETO_KEYS")

	// 2. Setup In-Memory Database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	err = db.AutoMigrate(&api.User{}, &api.UserDevice{}, &api.LLMUsageLog{})
	require.NoError(t, err, "failed to migrate database")

	// 3. Create a Free Tier User
	freeUser := api.User{
		Role:                "user",
		Tier:                string(api.TierFree),
		TierChangedAt:       time.Now().Unix(),
		CreatedAt:           time.Now().Unix(),
		PolarCustomerID:     "cus_free_123",
		PolarSubscriptionID: "sub_free_123",
	}
	err = db.Create(&freeUser).Error
	require.NoError(t, err, "failed to create test free user")

	// 4. Create an Unlimited User (e.g. Pro)
	proUser := api.User{
		Role:                "user",
		Tier:                string(api.TierPro),
		TierChangedAt:       time.Now().Unix(),
		CreatedAt:           time.Now().Unix(),
		PolarCustomerID:     "cus_pro_456",
		PolarSubscriptionID: "sub_pro_456",
	}
	err = db.Create(&proUser).Error
	require.NoError(t, err, "failed to create test pro user")

	// 5. Mint Tokens
	freeToken, err := api.MintToken(freeUser, freeUser.Role)
	require.NoError(t, err, "failed to mint token for free user")

	proToken, err := api.MintToken(proUser, proUser.Role)
	require.NoError(t, err, "failed to mint token for pro user")

	// 6. Setup the handler route
	handler := llmProxyHandler(db, "gemini")

	// Helper function to make requests
	makeRequest := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gemini/v1beta/models/gemini-3-flash-preview:generateContent", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	// 7. Test Free User Rate Limiting (5 allowed, 6th denied)
	for i := 0; i < 5; i++ {
		res := makeRequest(freeToken)
		assert.Equal(t, http.StatusOK, res.Code, "Expected OK for request %d", i+1)
	}

	// 8. Verify the 6th Request is Rate Limited
	resLimit := makeRequest(freeToken)
	assert.Equal(t, http.StatusTooManyRequests, resLimit.Code, "Expected Too Many Requests (429) for the 6th request")

	// Verify database state: User should have exactly 6 LLM items logged (5 allowed, 1 denied ... wait no, denied shouldn't make to LLM right?)
	// Actually, the 6th request is denied BEFORE it hits the LLM, so no log is generated. So the count should still be 5.
	var logsCount int64
	db.Model(&api.LLMUsageLog{}).Where("user_id = ?", freeUser.ID).Count(&logsCount)
	assert.Equal(t, int64(5), logsCount, "Free User should have exactly 5 distracting usage logs")

	// 9. Test Pro User (Should be allowed beyond 5)
	for i := 0; i < 10; i++ {
		res := makeRequest(proToken)
		assert.Equal(t, http.StatusOK, res.Code, "Expected OK for request %d to Pro User", i+1)
	}

	var proLogsCount int64
	db.Model(&api.LLMUsageLog{}).Where("user_id = ?", proUser.ID).Count(&proLogsCount)
	// Pro user responses are inspected and logs ARE created even for pro users in our logic
	assert.Equal(t, int64(10), proLogsCount, "Pro User should have 10 usage logs since the check is bypassed but logs are kept")
}

func TestLLMProxyHandler_NeutralClassification(t *testing.T) {
	// Neutral/productive classifications shouldn't increment the distracting count.
	mockGemini := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{
								"text": "{\"classification\": \"productive\"}"
							}
						]
					}
				}
			],
			"usageMetadata": {
				"promptTokenCount": 100,
				"candidatesTokenCount": 10,
				"totalTokenCount": 110
			}
		}`))
	}))
	defer mockGemini.Close()

	os.Setenv("GEMINI_BASE_URL", mockGemini.URL)
	defer os.Unsetenv("GEMINI_BASE_URL")
	os.Setenv("PASETO_KEYS", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	defer os.Unsetenv("PASETO_KEYS")

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&api.User{}, &api.UserDevice{}, &api.LLMUsageLog{})

	freeUser := api.User{
		Role:                "user",
		Tier:                string(api.TierFree),
		TierChangedAt:       time.Now().Unix(),
		CreatedAt:           time.Now().Unix(),
		PolarCustomerID:     "cus_free_neutral",
		PolarSubscriptionID: "sub_free_neutral",
	}
	db.Create(&freeUser)
	token, _ := api.MintToken(freeUser, freeUser.Role)

	handler := llmProxyHandler(db, "gemini")

	// Free user can make 10 requests since they are productive and thus not rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gemini/generateContent", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code, "Expected OK for productive request")
	}

	var logsCount int64
	db.Model(&api.LLMUsageLog{}).Count(&logsCount)
	assert.Equal(t, int64(10), logsCount, "10 usage logs should be created for productive classifications (but rate limits ignored)")
}
