package config

import (
	"testing"

	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	
	if config.Range != constants.DefaultRange {
		t.Errorf("Expected range %s, got %s", constants.DefaultRange, config.Range)
	}
	
	if config.Purity != constants.DefaultPurity {
		t.Errorf("Expected purity %s, got %s", constants.DefaultPurity, config.Purity)
	}
	
	if config.Categories != constants.DefaultCategories {
		t.Errorf("Expected categories %s, got %s", constants.DefaultCategories, config.Categories)
	}
}