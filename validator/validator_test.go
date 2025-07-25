package validator

import (
	"testing"

	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
)

func TestValidateRange(t *testing.T) {
	v := NewValidator()
	
	// Test valid ranges
	for _, validRange := range constants.ValidRanges {
		if err := v.ValidateRange(validRange); err != nil {
			t.Errorf("Expected valid range %s to pass validation, got error: %v", validRange, err)
		}
	}
	
	// Test invalid range
	if err := v.ValidateRange("invalid"); err == nil {
		t.Error("Expected invalid range to fail validation")
	}
}

func TestValidatePurity(t *testing.T) {
	v := NewValidator()
	
	// Test valid purity
	if err := v.ValidatePurity("110"); err != nil {
		t.Errorf("Expected valid purity to pass validation, got error: %v", err)
	}
	
	// Test invalid purity length
	if err := v.ValidatePurity("11"); err == nil {
		t.Error("Expected invalid purity length to fail validation")
	}
	
	// Test invalid purity characters
	if err := v.ValidatePurity("112"); err == nil {
		t.Error("Expected invalid purity characters to fail validation")
	}
}