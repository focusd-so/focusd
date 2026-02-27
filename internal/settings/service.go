package settings

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SettingsKey string

const (
	SettingsKeyCustomRules SettingsKey = "custom_rules"
	SettingsKeyAPIKey      SettingsKey = "api_key"
)

type Settings struct {
	ID        int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       SettingsKey `json:"key"`
	Value     string      `json:"value"`
	Version   int         `json:"version"`
	CreatedAt int64       `json:"created_at"`
}

type Service struct {
	db      *gorm.DB
	version string
}

func NewService(db *gorm.DB, version string) (*Service, error) {
	if err := db.Migrator().AutoMigrate(&Settings{}); err != nil {
		return nil, fmt.Errorf("failed to migrate settings table: %w", err)
	}

	return &Service{db: db, version: version}, nil
}

func (s *Service) GetVersion() string {
	return s.version
}

func (s *Service) Save(key SettingsKey, value string) error {
	setting := Settings{Key: key, Value: value}

	// get existing setting last version
	var existing Settings
	if err := s.db.Where("key = ?", key).Order("version DESC").First(&existing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to get existing setting: %w", err)
		}
	}

	setting.Version = existing.Version + 1
	setting.CreatedAt = time.Now().Unix()
	setting.Value = value

	if err := s.db.Create(&setting).Error; err != nil {
		return err
	}

	// Keep only the 10 most recent versions, delete older ones
	const maxVersionsToKeep = 10
	if setting.Version > maxVersionsToKeep {
		if err := s.db.Where("key = ? AND version <= ?", key, setting.Version-maxVersionsToKeep).
			Delete(&Settings{}).Error; err != nil {
			return fmt.Errorf("failed to cleanup old versions: %w", err)
		}
	}

	return nil
}

func (s *Service) GetLatest(key SettingsKey) (*Settings, error) {
	var setting Settings

	if err := s.db.Where("key = ?", key).Order("version DESC").First(&setting).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get setting: %w", err)
		}

		return nil, nil
	}

	return &setting, nil
}

// GetAll returns the latest version of each setting key.
//
// Returns:
//   - []Settings: A slice of settings with the latest version for each key
//   - error: Database error if the query fails
func (s *Service) GetAll() ([]Settings, error) {
	var settings []Settings

	// Get distinct keys and their latest versions using a subquery
	subQuery := s.db.Model(&Settings{}).
		Select("key, MAX(version) as max_version").
		Group("key")

	if err := s.db.Model(&Settings{}).
		Joins("JOIN (?) AS latest ON settings.key = latest.key AND settings.version = latest.max_version", subQuery).
		Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}

	return settings, nil
}

// GetVersionHistory returns the last N versions of a setting key, ordered by version descending.
//
// Parameters:
//   - key: The settings key to get history for
//   - limit: Maximum number of versions to return
//
// Returns:
//   - []Settings: A slice of settings versions, newest first
//   - error: Database error if the query fails
func (s *Service) GetVersionHistory(key SettingsKey, limit int) ([]Settings, error) {
	var settings []Settings
	if err := s.db.Where("key = ?", key).
		Order("version DESC").
		Limit(limit).
		Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}
	return settings, nil
}

// EnsureAPIKey creates a persistent API key if one does not already exist.
// Returns the API key (existing or newly created).
func (s *Service) EnsureAPIKey() (string, error) {
	existing, err := s.GetLatest(SettingsKeyAPIKey)
	if err != nil {
		return "", fmt.Errorf("failed to check for existing API key: %w", err)
	}
	if existing != nil && existing.Value != "" {
		return existing.Value, nil
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	key := hex.EncodeToString(b)

	if err := s.Save(SettingsKeyAPIKey, key); err != nil {
		return "", fmt.Errorf("failed to save API key: %w", err)
	}
	return key, nil
}

// GetAPIKey returns the stored API key, or empty string if none exists.
func (s *Service) GetAPIKey() (string, error) {
	setting, err := s.GetLatest(SettingsKeyAPIKey)
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", nil
	}
	return setting.Value, nil
}
