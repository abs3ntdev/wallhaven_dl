// Package config provides configuration management for wallhaven_dl
package config

import (
	"os"
	"path/filepath"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/constants"
)

// Config holds application configuration
type Config struct {
	Range      string   `json:"range"`
	Purity     string   `json:"purity"`
	Categories string   `json:"categories"`
	Sort       string   `json:"sort"`
	Order      string   `json:"order"`
	MaxPages   int      `json:"max_pages"`
	Ratios     []string `json:"ratios"`
	AtLeast    string   `json:"at_least"`

	DownloadPath string `json:"download_path"`
	ScriptPath   string `json:"script_path"`
}

// GetDefaultDownloadPath returns the default download path
func GetDefaultDownloadPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Pictures", "Wallpapers")
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Range:        constants.DefaultRange,
		Purity:       constants.DefaultPurity,
		Categories:   constants.DefaultCategories,
		Sort:         constants.DefaultSort,
		Order:        constants.DefaultOrder,
		MaxPages:     constants.DefaultMaxPages,
		Ratios:       constants.DefaultRatios,
		AtLeast:      constants.DefaultAtLeast,
		DownloadPath: GetDefaultDownloadPath(),
	}
}
