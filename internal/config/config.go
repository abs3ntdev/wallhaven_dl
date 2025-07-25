// Package config provides configuration management for wallhaven_dl
package config

import (
	"os"
	"path/filepath"

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/constants"
)

// Config holds application configuration
type Config struct {
	// Search parameters
	Range       string   `json:"range"`
	Purity      string   `json:"purity"`
	Categories  string   `json:"categories"`
	Sort        string   `json:"sort"`
	Order       string   `json:"order"`
	Page        int      `json:"page"`
	Ratios      []string `json:"ratios"`
	AtLeast     string   `json:"at_least"`

	// Paths
	DownloadPath string `json:"download_path"`
	ScriptPath   string `json:"script_path"`

	// Cleanup settings
	CleanupMode     string `json:"cleanup_mode"`
	CleanupOlderThan string `json:"cleanup_older_than"`
	DryRun          bool   `json:"dry_run"`

	// API settings
	APIKey string `json:"-"` // Never serialize API key

	// Application settings
	LogLevel string `json:"log_level"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Range:           constants.DefaultRange,
		Purity:          constants.DefaultPurity,
		Categories:      constants.DefaultCategories,
		Sort:            constants.DefaultSort,
		Order:           constants.DefaultOrder,
		Page:            constants.DefaultMaxPages,
		Ratios:          constants.DefaultRatios,
		AtLeast:         constants.DefaultAtLeast,
		DownloadPath:    filepath.Join(os.Getenv("HOME"), "Pictures", "Wallpapers"),
		ScriptPath:      "",
		CleanupMode:     constants.CleanupModeUnused,
		CleanupOlderThan: constants.DefaultCleanupOlderThan,
		DryRun:          false,
		APIKey:          os.Getenv("WH_API_KEY"),
		LogLevel:        "info",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	validators := []func() error{
		c.validateRange,
		c.validatePurity,
		c.validateCategories,
		c.validateSort,
		c.validateOrder,
		c.validatePaths,
	}

	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validateRange() error {
	for _, valid := range constants.ValidRanges {
		if c.Range == valid {
			return nil
		}
	}
	return NewValidationError("range", c.Range, "must be one of: "+joinStrings(constants.ValidRanges))
}

func (c *Config) validatePurity() error {
	if len(c.Purity) != 3 {
		return NewValidationError("purity", c.Purity, "must be 3 characters long")
	}
	for _, char := range c.Purity {
		if char != '0' && char != '1' {
			return NewValidationError("purity", c.Purity, "must contain only '0' and '1'")
		}
	}
	return nil
}

func (c *Config) validateCategories() error {
	if len(c.Categories) != 3 {
		return NewValidationError("categories", c.Categories, "must be 3 characters long")
	}
	for _, char := range c.Categories {
		if char != '0' && char != '1' {
			return NewValidationError("categories", c.Categories, "must contain only '0' and '1'")
		}
	}
	return nil
}

func (c *Config) validateSort() error {
	for _, valid := range constants.ValidSorts {
		if c.Sort == valid {
			return nil
		}
	}
	return NewValidationError("sort", c.Sort, "must be one of: "+joinStrings(constants.ValidSorts))
}

func (c *Config) validateOrder() error {
	for _, valid := range constants.ValidOrders {
		if c.Order == valid {
			return nil
		}
	}
	return NewValidationError("order", c.Order, "must be one of: "+joinStrings(constants.ValidOrders))
}

func (c *Config) validatePaths() error {
	if c.DownloadPath == "" {
		return NewValidationError("downloadPath", c.DownloadPath, "cannot be empty")
	}
	
	if c.ScriptPath != "" {
		if _, err := os.Stat(c.ScriptPath); os.IsNotExist(err) {
			return NewValidationError("scriptPath", c.ScriptPath, "file does not exist")
		}
	}
	
	return nil
}

// Helper functions
func joinStrings(strings []string) string {
	result := ""
	for i, s := range strings {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return "config validation failed for field '" + e.Field + "' with value '" + e.Value + "': " + e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}