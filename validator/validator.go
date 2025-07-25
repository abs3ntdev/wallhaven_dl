// Package validator provides input validation functions
package validator

import (
	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/errors"
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
	return errors.NewValidationError("range", value, "must be one of: "+joinStrings(constants.ValidRanges))
}

// ValidatePurity validates purity parameter
func (v *Validator) ValidatePurity(value string) error {
	if len(value) != 3 {
		return errors.NewValidationError("purity", value, "must be 3 characters long")
	}
	for _, char := range value {
		if char != '0' && char != '1' {
			return errors.NewValidationError("purity", value, "must contain only '0' and '1'")
		}
	}
	return nil
}

// ValidateCategories validates categories parameter
func (v *Validator) ValidateCategories(value string) error {
	if len(value) != 3 {
		return errors.NewValidationError("categories", value, "must be 3 characters long")
	}
	for _, char := range value {
		if char != '0' && char != '1' {
			return errors.NewValidationError("categories", value, "must contain only '0' and '1'")
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
	return errors.NewValidationError("sort", value, "must be one of: "+joinStrings(constants.ValidSorts))
}

// ValidateOrder validates order parameter
func (v *Validator) ValidateOrder(value string) error {
	for _, valid := range constants.ValidOrders {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError("order", value, "must be one of: "+joinStrings(constants.ValidOrders))
}

// ValidateRating validates rating parameter
func (v *Validator) ValidateRating(value int) error {
	if value < constants.MinRating || value > constants.MaxRating {
		return errors.NewValidationError("rating", string(rune(value)), "must be between 1 and 5")
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
	return errors.NewValidationError("cleanup_mode", value, "must be one of: "+joinStrings(constants.ValidCleanupModes))
}

// Helper function to join strings
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