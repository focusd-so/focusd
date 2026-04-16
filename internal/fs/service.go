package fs

import (
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	configDir string
}

func NewService(configDir string) *Service {
	return &Service{
		configDir: configDir,
	}
}

// ReadConfigFileSync securely reads a file from the .focusd config directory
// It is intended to be called synchronously by the frontend.
func (s *Service) ReadConfigFileSync(filename string) (string, error) {
	// Security check to prevent path traversal
	cleanName := filepath.Clean(filename)
	if strings.Contains(cleanName, "..") {
		return "", os.ErrNotExist
	}

	fullPath := filepath.Join(s.configDir, cleanName)
	bytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
