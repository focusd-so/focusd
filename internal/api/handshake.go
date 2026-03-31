package api

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
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
	"github.com/focusd-so/focusd/gen/api/v1/apiv1connect"
	"github.com/focusd-so/focusd/internal/identity"
	"github.com/focusd-so/focusd/internal/native"
)

const (
	KeychainService = "focusd-engine"
	KeychainUser    = "auth-token"
)

// PerformHandshake performs the device handshake to obtain a new token.
// It uses the provided client to make the call.
func PerformHandshake(ctx context.Context, client apiv1connect.ApiServiceClient) (string, error) {
	deviceFingerPrint, err := native.GetIdentity()
	if err != nil {
		return "", err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := generateNonce()
	stringToSign := deviceFingerPrint + timestamp + nonce

	secret := identity.GetHMACSecret()
	slog.Info("using secret", "secret", hex.EncodeToString(secret))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: deviceFingerPrint,
		AppVersion:        viper.GetString("app_version"),
	})

	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	slog.Info("performing handshake", "device_fingerprint", deviceFingerPrint)

	resp, err := client.DeviceHandshake(ctx, req)
	if err != nil {
		return "", err
	}

	slog.Info("handshake successful")

	if err := keyring.Set(KeychainService, KeychainUser, resp.Msg.GetSessionToken()); err != nil {
		slog.Warn("failed to store token in keyring", "error", err)
	}

	return resp.Msg.GetSessionToken(), nil
}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
