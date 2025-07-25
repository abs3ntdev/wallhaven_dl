package config

import (
	"testing"

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/constants"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Range != constants.DefaultRange {
		t.Errorf("Expected range %s, got %s", constants.DefaultRange, cfg.Range)
	}
	if cfg.Sort != constants.DefaultSort {
		t.Errorf("Expected sort %s, got %s", constants.DefaultSort, cfg.Sort)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  NewConfig(),
			wantErr: false,
		},
		{
			name: "invalid range",
			config: &Config{
				Range:       "invalid",
				Purity:      constants.DefaultPurity,
				Categories:  constants.DefaultCategories,
				Sort:        constants.DefaultSort,
				Order:       constants.DefaultOrder,
				DownloadPath: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "invalid purity",
			config: &Config{
				Range:       constants.DefaultRange,
				Purity:      "12", // too short
				Categories:  constants.DefaultCategories,
				Sort:        constants.DefaultSort,
				Order:       constants.DefaultOrder,
				DownloadPath: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "empty download path",
			config: &Config{
				Range:       constants.DefaultRange,
				Purity:      constants.DefaultPurity,
				Categories:  constants.DefaultCategories,
				Sort:        constants.DefaultSort,
				Order:       constants.DefaultOrder,
				DownloadPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}