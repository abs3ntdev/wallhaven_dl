// Package validator provides input validation functions
package validator

import (
	"os"
	"strconv"
	"strings"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/errors"
)

// Validator provides validation methods
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateRange validates time range parameter
func (v *Validator) ValidateRange(value string) error {
	for _, valid := range constants.ValidRanges {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError("range", value, "must be one of: "+strings.Join(constants.ValidRanges, ", "))
}

// ValidatePurity validates purity parameter
func (v *Validator) ValidatePurity(value string) error {
	return validateBitString("purity", value)
}

// ValidateCategories validates categories parameter
func (v *Validator) ValidateCategories(value string) error {
	return validateBitString("categories", value)
}

func validateBitString(field, value string) error {
	if len(value) != 3 {
		return errors.NewValidationError(field, value, "must be 3 characters long")
	}
	for _, char := range value {
		if char != '0' && char != '1' {
			return errors.NewValidationError(field, value, "must contain only '0' and '1'")
		}
	}
	return nil
}

// ValidateSort validates sort parameter
func (v *Validator) ValidateSort(value string) error {
	for _, valid := range constants.ValidSorts {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError("sort", value, "must be one of: "+strings.Join(constants.ValidSorts, ", "))
}

// ValidateOrder validates order parameter
func (v *Validator) ValidateOrder(value string) error {
	for _, valid := range constants.ValidOrders {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError("order", value, "must be one of: "+strings.Join(constants.ValidOrders, ", "))
}

// ValidatePage validates the maximum page count parameter
func (v *Validator) ValidatePage(value int) error {
	if value < 1 {
		return errors.NewValidationError("page", strconv.Itoa(value), "must be at least 1")
	}
	return nil
}

// ValidateRating validates rating parameter
func (v *Validator) ValidateRating(value int) error {
	if value < constants.MinRating || value > constants.MaxRating {
		return errors.NewValidationError("rating", strconv.Itoa(value), "must be between 1 and 5")
	}
	return nil
}

// ValidateCleanupMode validates cleanup mode parameter
func (v *Validator) ValidateCleanupMode(value string) error {
	for _, valid := range constants.ValidCleanupModes {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError("cleanup_mode", value, "must be one of: "+strings.Join(constants.ValidCleanupModes, ", "))
}

// ValidateScriptPath validates that a script path, when provided, exists
func (v *Validator) ValidateScriptPath(value string) error {
	if value == "" {
		return nil
	}
	if _, err := os.Stat(value); err != nil {
		return errors.NewValidationError("scriptPath", value, "file does not exist")
	}
	return nil
}
