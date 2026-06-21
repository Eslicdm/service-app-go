package dto

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MemberRequestDTO mirrors the Spring member-request-service MemberRequestDTO
// record: email (@NotBlank @Email) + serviceType (@NotNull ServiceType).
// The nested ServiceType enum serializes to lowercase ("free"/"half-price"/
// "full-price") and accepts both casings on input, matching Spring's
// @JsonValue / @JsonCreator.
type MemberRequestDTO struct {
	Email       string      `json:"email" binding:"required,email"`
	ServiceType ServiceType `json:"serviceType" binding:"required"`
}

// ServiceType is the nested enum for the 3 membership tiers. JSON wire format
// is lowercase (Spring @JsonValue); both casings accepted on input.
type ServiceType string

const (
	ServiceTypeFree      ServiceType = "free"
	ServiceTypeHalfPrice ServiceType = "half-price"
	ServiceTypeFullPrice ServiceType = "full-price"
)

func (st ServiceType) IsValid() bool {
	switch st {
	case ServiceTypeFree, ServiceTypeHalfPrice, ServiceTypeFullPrice:
		return true
	}
	return false
}

// MarshalJSON returns the lowercase wire value.
func (st ServiceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(st))
}

// UnmarshalJSON accepts both the lowercase wire value and the uppercase enum name.
func (st *ServiceType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("service type should be a string, got %s", data)
	}
	parsed, err := ParseServiceType(s)
	if err != nil {
		return err
	}
	*st = parsed
	return nil
}

// ParseServiceType parses a string into a ServiceType, accepting both casings.
func ParseServiceType(s string) (ServiceType, error) {
	switch strings.ToLower(strings.ReplaceAll(s, "_", "-")) {
	case "free":
		return ServiceTypeFree, nil
	case "half-price":
		return ServiceTypeHalfPrice, nil
	case "full-price":
		return ServiceTypeFullPrice, nil
	}
	return "", fmt.Errorf("invalid service type: %s", s)
}
