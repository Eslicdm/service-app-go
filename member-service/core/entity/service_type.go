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
