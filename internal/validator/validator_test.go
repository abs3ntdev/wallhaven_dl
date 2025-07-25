package validator

import (
	"testing"

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/constants"
)

func TestValidator_ValidateRange(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 1d", constants.Range1Day, false},
		{"valid 1y", constants.Range1Year, false},
		{"invalid value", "2y", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRange(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePurity(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 110", "110", false},
		{"valid 001", "001", false},
		{"invalid length", "11", true},
		{"invalid chars", "abc", true},
		{"mixed chars", "1a0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePurity(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateCategories(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 010", "010", false},
		{"valid 111", "111", false},
		{"invalid length", "01", true},
		{"invalid chars", "xyz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCategories(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategories() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateSort(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid toplist", constants.SortToplist, false},
		{"valid random", constants.SortRandom, false},
		{"invalid value", "newest", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSort(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateRating(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid 1", 1, false},
		{"valid 5", 5, false},
		{"invalid 0", 0, true},
		{"invalid 6", 6, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRating(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRating() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}