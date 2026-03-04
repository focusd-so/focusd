package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
)

type ServiceImpl struct {
	gormDB *gorm.DB

	productIDs map[apiv1.CheckoutProduct]string
}

func NewServiceImpl(gormDB *gorm.DB, productIDs map[apiv1.CheckoutProduct]string) (*ServiceImpl, error) {
	if err := gormDB.AutoMigrate(&User{}, &UserDevice{}, &HandshakeNonce{}, &LLMProxyUsage{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &ServiceImpl{gormDB: gormDB, productIDs: productIDs}, nil
}

var _ apiv1connect.ApiServiceHandler = (*ServiceImpl)(nil)

// DeviceHandshake performs the initial authentication flow for a device:
// This is a public endpoint and doesn't use the standard AuthInterceptor.
//
// 1. Verify HMAC Signature (App Attestation) manually since this is a public endpoint.
// 2. Find or create a shadow user based on the device fingerprint.
// 3. Mint a PASETO session token for the user.
// 4. Return the session token.
func (s *ServiceImpl) DeviceHandshake(ctx context.Context, req *connect.Request[apiv1.DeviceHandshakeRequest]) (*connect.Response[apiv1.DeviceHandshakeResponse], error) {
	if err := s.verifyHMAC(req); err != nil {
		slog.Error("failed to verify hmac", "error", err)
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("signature verification failed: %w", err))
	}

	if req.Msg.DeviceFingerprint == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fingerprint required"))
	}

	user, err := s.upsertShadowUser(ctx, req.Msg.DeviceFingerprint)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("db error: %w", err))
	}

	sessionToken, err := MintToken(user, user.Role)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to mint session: %w", err))
	}

	accountTier := apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL
	var trialEndsAt int64 = 0

	switch user.Tier {
	case string(TierTrial):
		accountTier = apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL
		trialEndsAt = user.TierChangedAt + 7*24*60*60
	case string(TierPlus):
		accountTier = apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS
	case string(TierPro):
		accountTier = apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO
	case string(TierFree):
		accountTier = apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_FREE
	}

	return connect.NewResponse(&apiv1.DeviceHandshakeResponse{
		SessionToken: sessionToken,
		UserId:       user.ID,
		AccountTier:  accountTier,
		TrialEndsAt:  trialEndsAt,
	}), nil
}

func (s *ServiceImpl) verifyHMAC(req *connect.Request[apiv1.DeviceHandshakeRequest]) error {
	timestampStr := req.Header().Get("X-Timestamp")
	nonce := req.Header().Get("X-Nonce")
	signature := req.Header().Get("X-Signature")

	if timestampStr == "" || signature == "" {
		return errors.New("missing security headers")
	}

	// Replay Attack Check (Timestamp window: 30 seconds)
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return errors.New("invalid timestamp")
	}

	now := time.Now().Unix()
	if now-ts > 30 || ts-now > 30 {
		return errors.New("request expired")
	}

	// Replay Attack Check (Nonce)
	var existing HandshakeNonce
	if err = s.gormDB.Where("nonce = ?", nonce).First(&existing).Error; err == nil {
		return errors.New("duplicate nonce")
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("db error: %w", err)
	}

	if err := s.gormDB.Create(&HandshakeNonce{Nonce: nonce, CreatedAt: now, ExpiresAt: now + 30}).Error; err != nil {
		return fmt.Errorf("db error: %w", err)
	}

	// Reconstruct the String-to-Sign
	// Must match Client Logic EXACTLY: "BodyJson+Timestamp+Nonce"
	// Note: In ConnectRPC, we don't always have raw JSON body easily accessbile
	// in the handler object without middleware.
	// SIMPLIFICATION: Sign the Fingerprint field specifically, not whole JSON.
	payload := req.Msg.DeviceFingerprint + timestampStr + nonce

	secret := []byte("dev-mode-secret")
	if os.Getenv("HMAC_SECRET_KEY") != "" {
		hmacSecret := os.Getenv("HMAC_SECRET_KEY")
		secret, err = hex.DecodeString(hmacSecret)
		if err != nil {
			return errors.New("internal server error")
		}
		slog.Info("using secret", "secret", hex.EncodeToString(secret))
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("invalid signature")
	}

	return nil
}

func (s *ServiceImpl) upsertShadowUser(_ context.Context, fingerprint string) (User, error) {
	var userDevice UserDevice

	if err := s.gormDB.Preload("User").Where("fingerprint = ?", fingerprint).First(&userDevice).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return User{}, err
		}
	}

	user := userDevice.User

	if user.ID == 0 {
		user = User{
			Role:          "anonymous",
			Tier:          string(TierTrial),
			TierChangedAt: time.Now().Unix(),
			CreatedAt:     time.Now().Unix(),
			Devices:       []UserDevice{{Fingerprint: fingerprint, CreatedAt: time.Now().Unix()}},
		}
	}

	// If user has been trialing for more than 7 days, change tier to free
	sevenDaysAgo := time.Now().Unix() - 7*24*60*60
	if user.Tier == string(TierTrial) && user.TierChangedAt < sevenDaysAgo {
		user.Tier = string(TierFree)
	}

	if err := s.gormDB.Save(&user).Error; err != nil {
		return User{}, err
	}

	return user, nil
}
