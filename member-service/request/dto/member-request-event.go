package dto

import (
	"encoding/json"
	"fmt"
	"strings"

	"service-app-go/member-service/core/entity"
)

// MemberRequestEvent mirrors the Spring member-service
// com.eslirodrigues.member.request.dto.MemberRequestEvent. It is the JSON shape
// produced by member-request-service to Kafka (member.requests.topic).
type MemberRequestEvent struct {
	Email       string              `json:"email"`
	ServiceType entity.ServiceType  `json:"serviceType"`
}

// UnmarshalJSON ensures the serviceType accepts both the lowercase wire value
// and the uppercase enum name (the producer may send either).
func (e *MemberRequestEvent) UnmarshalJSON(data []byte) error {
	type alias MemberRequestEvent
	var raw struct {
		Email       string `json:"email"`
		ServiceType string `json:"serviceType"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Email = raw.Email
	st, ok := parseServiceType(raw.ServiceType)
	if !ok {
		return fmt.Errorf("invalid serviceType: %s", raw.ServiceType)
	}
	e.ServiceType = st
	return nil
}

func parseServiceType(s string) (entity.ServiceType, bool) {
	switch strings.ToLower(strings.ReplaceAll(s, "_", "-")) {
	case "free":
		return entity.ServiceTypeFree, true
	case "half-price":
		return entity.ServiceTypeHalfPrice, true
	case "full-price":
		return entity.ServiceTypeFullPrice, true
	}
	return "", false
}
