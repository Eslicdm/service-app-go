package entity

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PriceType represents the type of a price (e.g., FREE, HALF_PRICE, FULL_PRICE).
type PriceType string

const (
	PriceTypeFree      PriceType = "FREE"
	PriceTypeHalfPrice PriceType = "HALF_PRICE"
	PriceTypeFullPrice PriceType = "FULL_PRICE"
)

// IsValid checks if the PriceType is one of the defined valid types.
func (pt PriceType) IsValid() bool {
	switch pt {
	case PriceTypeFree, PriceTypeHalfPrice, PriceTypeFullPrice:
		return true
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface.
func (pt PriceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(pt))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (pt *PriceType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("price type should be a string, got %s", data)
	}

	// Normalize by replacing hyphens with underscores and then converting to uppercase
	normalizedS := strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
	switch normalizedS {
	case string(PriceTypeFree):
		*pt = PriceTypeFree
	case string(PriceTypeHalfPrice):
		*pt = PriceTypeHalfPrice
	case string(PriceTypeFullPrice):
		*pt = PriceTypeFullPrice
	default:
		return fmt.Errorf("invalid price type: %s", s)
	}
	return nil
}
