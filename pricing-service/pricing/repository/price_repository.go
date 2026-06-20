package repository

import (
	"context"

	"service-app-go/pricing-service/core/entity"
)

// PriceRepository defines the interface for price data operations, mirroring the
// Spring pricing-service which only needs findAll and findByPriceType (+ upsert save).
type PriceRepository interface {
	Save(ctx context.Context, price entity.Price) (*entity.Price, error)
	FindByPriceType(ctx context.Context, priceType entity.PriceType) (*entity.Price, error)
	FindAll(ctx context.Context) ([]entity.Price, error)
}
