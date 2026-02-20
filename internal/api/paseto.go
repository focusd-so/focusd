package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/o1egl/paseto"
)

// KeyManager handles rotation. Keys are stored in env var:
// PASETO_KEYS="HEX_KEY_NEW,HEX_KEY_OLD"
type KeyManager struct{}

func (km KeyManager) GetActiveKey() ([]byte, error) {
	keys := strings.Split(os.Getenv("PASETO_KEYS"), ",")
	if len(keys) == 0 || keys[0] == "" {
		return nil, errors.New("PASETO_KEYS not configured")
	}
	return hex.DecodeString(strings.TrimSpace(keys[0]))
}

func (km KeyManager) GetAllKeys() ([][]byte, error) {
	rawKeys := strings.Split(os.Getenv("PASETO_KEYS"), ",")
	var parsedKeys [][]byte

	for _, k := range rawKeys {
		if k == "" {
			continue
		}
		b, err := hex.DecodeString(strings.TrimSpace(k))
		if err != nil {
			return nil, fmt.Errorf("invalid hex key: %v", err)
		}
		parsedKeys = append(parsedKeys, b)
	}

	if len(parsedKeys) == 0 {
		return nil, errors.New("no valid keys found")
	}
	return parsedKeys, nil
}

// UserClaims represents the data inside the encrypted token
type UserClaims struct {
	UserID    int64     `json:"sub"`
	Role      string    `json:"role"` // "anonymous" or "pro"
	ExpiresAt time.Time `json:"exp"`
	Tier      string    `json:"tier"`
}

// Valid checks if token is expired
func (c *UserClaims) Valid() error {
	if time.Now().After(c.ExpiresAt) {
		return errors.New("token expired")
	}
	return nil
}

// MintToken creates a new encrypted PASETO token
func MintToken(user User, role string) (string, error) {
	km := KeyManager{}
	key, err := km.GetActiveKey()
	if err != nil {
		return "", err
	}

	claims := UserClaims{
		UserID:    user.ID,
		Role:      role,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5m Session
		Tier:      user.Tier,
	}

	// Sign & Encrypt (v2.local)
	return paseto.NewV2().Encrypt(key, claims, nil)
}

// ValidateToken decrypts the token trying all available keys
func ValidateToken(tokenStr string) (*UserClaims, error) {
	km := KeyManager{}
	keys, err := km.GetAllKeys()
	if err != nil {
		return nil, err
	}

	var claims UserClaims
	var lastErr error

	// Try keys in order (Active -> Old)
	for _, key := range keys {
		err := paseto.NewV2().Decrypt(tokenStr, key, &claims, nil)
		if err == nil {
			// Decrypt success, check expiration
			if expErr := claims.Valid(); expErr != nil {
				return nil, expErr
			}
			return &claims, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("invalid token: %v", lastErr)
}
