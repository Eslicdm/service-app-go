package repository

import (
	"context"

	"service-app-go/pricing-service/core/entity"
)

// PriceRepository defines the interface for price data operations.
type PriceRepository interface {
	Save(ctx context.Context, price entity.Price) (*entity.Price, error)
	FindByID(ctx context.Context, id string) (*entity.Price, error)
	FindByPriceType(ctx context.Context, priceType entity.PriceType) (*entity.Price, error)
	FindAll(ctx context.Context) ([]entity.Price, error)
	Delete(ctx context.Context, id string) error
}
