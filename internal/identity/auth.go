package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
	"github.com/focusd-so/focusd/internal/native"
)

var (
	lastHandshakeAt int64
	accountTier     apiv1.DeviceHandshakeResponse_AccountTier
	trialEndsAt     int64
	token           string
)

const (
	KeychainService = "focusd-engine"
	KeychainUser    = "auth-token"
)

func ScheduleHandshake(ctx context.Context, client apiv1connect.ApiServiceClient) error {
	slog.Info("scheduling handshake")

	// Perform the handshake immediately to load the token and account tier.
	if err := PerformHandshake(ctx, client); err != nil {
		return fmt.Errorf("failed to perform handshake: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(4 * time.Minute):
				if err := PerformHandshake(ctx, client); err != nil {
					slog.Error("failed to perform handshake", "error", err)
				}
			}
		}
	}()

	return nil
}

func GetToken(ctx context.Context) (string, error) {
	return token, nil
}

func GetAccountTier() apiv1.DeviceHandshakeResponse_AccountTier {
	return accountTier
}

func GetTrialEndsAt() int64 {
	return trialEndsAt
}

func PerformHandshake(ctx context.Context, client apiv1connect.ApiServiceClient) error {
	deviceFingerPrint, err := native.GetIdentity()
	if err != nil {
		return fmt.Errorf("failed to get device fingerprint: %w", err)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := generateNonce()
	stringToSign := deviceFingerPrint + timestamp + nonce

	secret := GetHMACSecret()
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: deviceFingerPrint,
	})

	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	slog.Info("performing handshake", "device_fingerprint", deviceFingerPrint)

	resp, err := client.DeviceHandshake(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to perform handshake: %w", err)
	}

	slog.Info("handshake successful")

	accountTier = resp.Msg.GetAccountTier()
	trialEndsAt = resp.Msg.GetTrialEndsAt()
	lastHandshakeAt = time.Now().Unix()
	token = resp.Msg.GetSessionToken()

	return nil
}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
