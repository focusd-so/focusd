//go:build !production

package identity

func GetHMACSecret() []byte {
	return []byte("dev-mode-secret")
}
