package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PriceUpdateEventDTO mirrors the Spring member-service
// com.eslirodrigues.member.pricing.dto.PriceUpdateEventDTO. It is the JSON
// shape published by the pricing-service to RabbitMQ (and cached in Redis).
type PriceUpdateEventDTO struct {
	ID          string    `json:"id"`
	PriceType   PriceType `json:"priceType"`
	Value       float64   `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PriceType mirrors the pricing-service PriceType: JSON wire format is the
// lowercase value ("free"/"half-price"/"full-price"), and both casings are
// accepted on input (Spring @JsonValue / @JsonCreator).
type PriceType string

const (
	PriceTypeFree      PriceType = "free"
	PriceTypeHalfPrice PriceType = "half-price"
	PriceTypeFullPrice PriceType = "full-price"
)

// IsValid reports whether pt is one of the defined price types.
func (pt PriceType) IsValid() bool {
	switch pt {
	case PriceTypeFree, PriceTypeHalfPrice, PriceTypeFullPrice:
		return true
	}
	return false
}

// MarshalJSON returns the lowercase wire value.
func (pt PriceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(pt))
}

// UnmarshalJSON accepts both the lowercase wire value and the uppercase enum name.
func (pt *PriceType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("price type should be a string, got %s", data)
	}
	parsed, err := ParsePriceType(s)
	if err != nil {
		return err
	}
	*pt = parsed
	return nil
}

// ParsePriceType parses a string into a PriceType, accepting both casings.
func ParsePriceType(s string) (PriceType, error) {
	switch strings.ToLower(strings.ReplaceAll(s, "_", "-")) {
	case "free":
		return PriceTypeFree, nil
	case "half-price":
		return PriceTypeHalfPrice, nil
	case "full-price":
		return PriceTypeFullPrice, nil
	}
	return "", fmt.Errorf("invalid price type: %s", s)
}

// AllPriceTypes returns the 3 known price types (used to build cache keys).
func AllPriceTypes() []PriceType {
	return []PriceType{PriceTypeFree, PriceTypeHalfPrice, PriceTypeFullPrice}
}
