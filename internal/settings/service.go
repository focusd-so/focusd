package settings

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/viper"
)

// This service exposes interface for the frontend to interact with
// viper to read/write settings through wails3 bindings mechanism
type Service struct{}

func NewService(configDir string) (*Service, error) {
	defaultCfg := DefaultConfig()

	// Set defaults
	viper.SetDefault("idle_threshold_seconds", defaultCfg.IdleThresholdSeconds)
	viper.SetDefault("history_retention_days", defaultCfg.HistoryRetentionDays)
	viper.SetDefault("distraction_allowance_minutes", defaultCfg.DistractionAllowanceMinutes)
	viper.SetDefault("custom_rules_js", defaultCfg.CustomRulesJS)
	viper.SetDefault("classification_llm_provider", defaultCfg.ClassificationLLMProvider)
	viper.SetDefault("app_version", "dev")

	// Config file settings
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(filepath.Join(configDir, "config.yaml")); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return &Service{}, nil
}

func (s *Service) GetConfig() *AppConfig {
	return GetConfig()
}

func (s *Service) SaveConfig(config AppConfig) error {
	viper.Set("idle_threshold_seconds", config.IdleThresholdSeconds)
	viper.Set("history_retention_days", config.HistoryRetentionDays)
	viper.Set("distraction_allowance_minutes", config.DistractionAllowanceMinutes)
	viper.Set("custom_rules_js", config.CustomRulesJS)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config to disk: %w", err)
	}

	return nil
}

func GetConfig() *AppConfig {
	config := DefaultConfig()
	if err := viper.Unmarshal(&config); err != nil {
		slog.Warn("failed to unmarshal config", "error", err)
	}

	return &config
}

func GetCustomRulesJS() string {
	config := GetConfig()

	if len(config.CustomRulesJS) == 0 {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(config.CustomRulesJS[0])
	if err != nil {
		slog.Warn("failed to decode custom rules", "error", err)
		return ""
	}

	return string(decoded)
}
