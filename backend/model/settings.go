package model

import (
	"dacapo/backend/utils"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppSettings struct {
	Language          string `yaml:"language"`
	RunOnStartup      bool   `yaml:"run_on_startup"`
	SchedulerCron     string `yaml:"scheduler_cron"`
	AutoActionTrigger string `yaml:"auto_action_trigger"`
	AutoActionCron    string `yaml:"auto_action_cron"`
	AutoActionType    string `yaml:"auto_action_type"`
	MaxBgConcurrent   int    `yaml:"max_bg_concurrent"`
	ServerChanSendKey string `yaml:"serverchan_sendkey"`
}

const settingsPath = "settings.yml"

// LoadSettings loads settings from YAML file
func LoadSettings() (*AppSettings, error) {
	settings := &AppSettings{
		// Set default values
		Language:          "", // Empty initially for auto-detection
		RunOnStartup:      false,
		SchedulerCron:     "",
		AutoActionTrigger: "scheduler_end",
		AutoActionCron:    "",
		AutoActionType:    "none",
		MaxBgConcurrent:   0,  // Default to 0 (no limit)
		ServerChanSendKey: "", // Default to empty (disabled)
	} // Create settings directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return settings, err
	}

	// Check if settings file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// Create settings file with default values
		if err := SaveSettings(settings); err != nil {
			utils.Logger.Warn("Failed to create default settings file:", err)
		}
		return settings, nil
	}

	// Read existing settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		utils.Logger.Warn("Failed to read settings file:", err)
		return settings, err
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, settings); err != nil {
		utils.Logger.Warn("Failed to parse settings file:", err)
		return settings, err
	}

	return settings, nil
}

// SaveSettings saves settings to YAML file
func SaveSettings(settings *AppSettings) error {
	// Create settings directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(settingsPath, data, 0644)
}

// UpdateSettings updates specific fields in the settings
func UpdateSettings(updates *ReqUpdateSettings) error {
	settings, err := LoadSettings()
	if err != nil {
		return err
	}

	// Update only provided fields
	if updates.Language != "" {
		settings.Language = updates.Language
	}
	if updates.RunOnStartup != nil {
		settings.RunOnStartup = *updates.RunOnStartup
	}
	if updates.SchedulerCron != nil {
		settings.SchedulerCron = *updates.SchedulerCron
	}
	if updates.AutoActionTrigger != nil {
		settings.AutoActionTrigger = *updates.AutoActionTrigger
	}
	if updates.AutoActionCron != nil {
		settings.AutoActionCron = *updates.AutoActionCron
	}
	if updates.AutoActionType != nil {
		settings.AutoActionType = *updates.AutoActionType
	}
	if updates.MaxBgConcurrent != nil {
		if *updates.MaxBgConcurrent >= 0 {
			settings.MaxBgConcurrent = *updates.MaxBgConcurrent
		}
	}
	if updates.ServerChanSendKey != nil {
		settings.ServerChanSendKey = *updates.ServerChanSendKey
	}

	return SaveSettings(settings)
}
