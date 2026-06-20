package entity

type ServiceType string

const (
	ServiceTypeFree      ServiceType = "free"
	ServiceTypeHalfPrice ServiceType = "half-price"
	ServiceTypeFullPrice ServiceType = "full-price"
)

func (s ServiceType) String() string {
	return string(s)
}

func (s ServiceType) IsValid() bool {
	switch s {
	case ServiceTypeFree, ServiceTypeHalfPrice, ServiceTypeFullPrice:
		return true
	}
	return false
}

// ParseServiceType parses a string into a ServiceType, accepting both the
// lowercase wire value ("free"/"half-price") and the uppercase enum name
// ("FREE"/"HALF_PRICE"). Matches the Spring ServiceType.@JsonCreator behavior.
func ParseServiceType(s string) (ServiceType, bool) {
	switch s {
	case "free", "FREE":
		return ServiceTypeFree, true
	case "half-price", "HALF_PRICE", "HALF-PRICE":
		return ServiceTypeHalfPrice, true
	case "full-price", "FULL_PRICE", "FULL-PRICE":
		return ServiceTypeFullPrice, true
	}
	return "", false
}
