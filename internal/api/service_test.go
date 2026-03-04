package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	apiv1 "github.com/focusd-so/focusd/gen/api/v1"
)

func TestDeviceHandshake(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	// 1. Setup Hex Secret
	secret := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	os.Setenv("HMAC_SECRET_KEY", secret)
	defer os.Unsetenv("HMAC_SECRET_KEY")

	// set PASETO_KEYS
	os.Setenv("PASETO_KEYS", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	defer os.Unsetenv("PASETO_KEYS")

	svc, err := NewServiceImpl(db, map[apiv1.CheckoutProduct]string{
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: "plus-product-id",
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  "pro-product-id",
	})
	require.NoError(t, err, "failed to create service")

	fingerprint := "test-device-fp"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "test-nonce-123"

	payload := fingerprint + timestamp + nonce
	secretBytes, _ := hex.DecodeString(secret)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	// ---------------------------------------------------------
	// Perform the first handshake
	// ---------------------------------------------------------
	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	resp, err := svc.DeviceHandshake(context.Background(), req)
	require.NoError(t, err, "failed to call device handshake")
	require.NotNil(t, resp, "response should not be nil")
	require.Equal(t, resp.Msg.UserId, int64(1), "user id should be 1")

	// ---------------------------------------------------------
	// Perform the second handshake, expect the existing user to be returned
	// ---------------------------------------------------------

	newNonce := "test-nonce-124"
	newPayload := fingerprint + timestamp + newNonce
	secretBytes, _ = hex.DecodeString(secret)
	mac = hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(newPayload))
	newSignature := hex.EncodeToString(mac.Sum(nil))

	req = connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", newNonce)
	req.Header().Set("X-Signature", newSignature)

	resp, err = svc.DeviceHandshake(context.Background(), req)
	require.NoError(t, err, "failed to call device handshake")
	require.NotNil(t, resp, "response should not be nil")
	require.Equal(t, resp.Msg.UserId, int64(1), "user id should be 1")
}

func TestDeviceHandshake_MissingSecurityHeaders(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	// Setup Hex Secret
	secret := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	os.Setenv("HMAC_SECRET_KEY", secret)
	defer os.Unsetenv("HMAC_SECRET_KEY")

	svc, err := NewServiceImpl(db, map[apiv1.CheckoutProduct]string{
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: "plus-product-id",
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  "pro-product-id",
	})
	require.NoError(t, err, "failed to create service")

	fingerprint := "test-device-fp"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "test-nonce-123"

	payload := fingerprint + timestamp + nonce
	secretBytes, _ := hex.DecodeString(os.Getenv("HMAC_SECRET_KEY"))
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payload))

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", "")

	resp, err := svc.DeviceHandshake(context.Background(), req)
	require.Error(t, err, "expected error")
	require.Nil(t, resp, "response should be nil")
	require.Equal(t, connect.CodePermissionDenied, err.(*connect.Error).Code())
	require.Equal(t, "permission_denied: signature verification failed: missing security headers", err.Error())
}

func TestDeviceHandshake_InvalidTimestamp(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	// Setup Hex Secret
	secret := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	os.Setenv("HMAC_SECRET_KEY", secret)
	defer os.Unsetenv("HMAC_SECRET_KEY")

	svc, err := NewServiceImpl(db, map[apiv1.CheckoutProduct]string{
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: "plus-product-id",
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  "pro-product-id",
	})
	require.NoError(t, err, "failed to create service")

	fingerprint := "test-device-fp"
	timestamp := fmt.Sprintf("%d", time.Now().Unix()+31)
	nonce := "test-nonce-123"

	payload := fingerprint + timestamp + nonce
	secretBytes, _ := hex.DecodeString(os.Getenv("HMAC_SECRET_KEY"))
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	resp, err := svc.DeviceHandshake(context.Background(), req)
	require.Error(t, err, "expected error")
	require.Nil(t, resp, "response should be nil")
	require.Equal(t, connect.CodePermissionDenied, err.(*connect.Error).Code())
	require.Equal(t, "permission_denied: signature verification failed: request expired", err.Error())
}

func TestDeviceHandshake_DuplicateNonce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	// set PASETO_KEYS
	os.Setenv("PASETO_KEYS", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	defer os.Unsetenv("PASETO_KEYS")

	// Setup Hex Secret
	secret := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	os.Setenv("HMAC_SECRET_KEY", secret)
	defer os.Unsetenv("HMAC_SECRET_KEY")

	svc, err := NewServiceImpl(db, map[apiv1.CheckoutProduct]string{
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: "plus-product-id",
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  "pro-product-id",
	})
	require.NoError(t, err, "failed to create service")

	fingerprint := "test-device-fp"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "test-nonce-123"

	payload := fingerprint + timestamp + nonce
	secretBytes, _ := hex.DecodeString(os.Getenv("HMAC_SECRET_KEY"))
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	resp, err := svc.DeviceHandshake(context.Background(), req)
	require.NoError(t, err, "failed to call device handshake")
	require.NotNil(t, resp, "response should not be nil")
	require.Equal(t, resp.Msg.UserId, int64(1), "user id should be 1")

	req = connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", signature)

	resp, err = svc.DeviceHandshake(context.Background(), req)
	require.Error(t, err, "expected error")
	require.Nil(t, resp, "response should be nil")
	require.Equal(t, connect.CodePermissionDenied, err.(*connect.Error).Code())
	require.Equal(t, "permission_denied: signature verification failed: duplicate nonce", err.Error())
}

func TestDeviceHandshake_InvalidSignature(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	// Setup Hex Secret
	secret := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	os.Setenv("HMAC_SECRET_KEY", secret)
	defer os.Unsetenv("HMAC_SECRET_KEY")

	svc, err := NewServiceImpl(db, map[apiv1.CheckoutProduct]string{
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PLUS: "plus-product-id",
		apiv1.CheckoutProduct_CHECKOUT_PRODUCT_PRO:  "pro-product-id",
	})
	require.NoError(t, err, "failed to create service")

	fingerprint := "test-device-fp"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "test-nonce-123"

	req := connect.NewRequest(&apiv1.DeviceHandshakeRequest{
		DeviceFingerprint: fingerprint,
	})
	req.Header().Set("X-Timestamp", timestamp)
	req.Header().Set("X-Nonce", nonce)
	req.Header().Set("X-Signature", "invalid-signature")

	resp, err := svc.DeviceHandshake(context.Background(), req)
	require.Error(t, err, "expected error")
	require.Nil(t, resp, "response should be nil")
	require.Equal(t, connect.CodePermissionDenied, err.(*connect.Error).Code())
	require.Equal(t, "permission_denied: signature verification failed: invalid signature", err.Error())
}
