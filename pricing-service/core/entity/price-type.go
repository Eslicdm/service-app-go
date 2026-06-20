package entity

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PriceType represents the type of a price. The canonical Go identifiers are
// uppercase, but the JSON wire format matches the Spring reference (@JsonValue):
// lowercase "free", "half-price", "full-price". Unmarshal accepts both casings.
type PriceType string

const (
	PriceTypeFree      PriceType = "FREE"
	PriceTypeHalfPrice PriceType = "HALF_PRICE"
	PriceTypeFullPrice PriceType = "FULL_PRICE"
)

// wire maps each PriceType to its lowercase JSON value (Spring @JsonValue).
func (pt PriceType) wire() string {
	switch pt {
	case PriceTypeFree:
		return "free"
	case PriceTypeHalfPrice:
		return "half-price"
	case PriceTypeFullPrice:
		return "full-price"
	}
	return string(pt)
}

// IsValid checks if the PriceType is one of the defined valid types.
func (pt PriceType) IsValid() bool {
	switch pt {
	case PriceTypeFree, PriceTypeHalfPrice, PriceTypeFullPrice:
		return true
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface -> lowercase wire value.
func (pt PriceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(pt.wire())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Accepts both the lowercase wire value ("free"/"half-price") and the
// uppercase enum name ("FREE"/"HALF_PRICE") for backward compatibility.
func (pt *PriceType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("price type should be a string, got %s", data)
	}

	parsed, err := PriceTypeFromString(s)
	if err != nil {
		return err
	}
	*pt = parsed
	return nil
}

// PriceTypeFromString parses a string into a PriceType, accepting both casings.
func PriceTypeFromString(s string) (PriceType, error) {
	switch strings.ToLower(s) {
	case "free":
		return PriceTypeFree, nil
	case "half-price", "half_price":
		return PriceTypeHalfPrice, nil
	case "full-price", "full_price":
		return PriceTypeFullPrice, nil
	}
	return "", fmt.Errorf("invalid price type: %s", s)
}
