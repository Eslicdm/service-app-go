package dto

import "service-app-go/pricing-service/core/entity"

type CreatePriceDTO struct {
	PriceType   entity.PriceType `json:"priceType" binding:"required,oneof=free half-price full-price"`
	Value       float64          `json:"value" binding:"required,min=0.0"`
	Description string           `json:"description" binding:"required"`
}
