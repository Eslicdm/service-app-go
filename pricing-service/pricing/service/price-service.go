package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/pricing/dto"
	"service-app-go/pricing-service/pricing/messaging"
	"service-app-go/pricing-service/pricing/repository"
)

const (
	errPriceTypeNotFound = "price with type %s not found"
)

// PriceService defines the business logic for price operations, mirroring the
// Spring pricing-service: getAllPrices and updatePrice(byType) + publish event.
type PriceService interface {
	GetAllPrices(ctx context.Context) ([]entity.Price, error)
	UpdatePriceByType(ctx context.Context, priceType entity.PriceType, updateDTO dto.UpdatePriceDTO) (*entity.Price, error)
}

// priceService implements PriceService.
type priceService struct {
	repo      repository.PriceRepository
	publisher *messaging.PricePublisher
}

// NewPriceService creates a new PriceService with an optional RabbitMQ publisher.
func NewPriceService(repo repository.PriceRepository, publisher *messaging.PricePublisher) PriceService {
	return &priceService{
		repo:      repo,
		publisher: publisher,
	}
}

// GetAllPrices retrieves all prices.
func (s *priceService) GetAllPrices(ctx context.Context) ([]entity.Price, error) {
	return s.repo.FindAll(ctx)
}

// UpdatePriceByType upserts a price by its PriceType and publishes a price
// update event to RabbitMQ. Mirrors Spring's PriceService.updatePrice:
// find by type or create a new one (set createdAt), set value/description/updatedAt,
// save, then sendPriceUpdateNotification.
func (s *priceService) UpdatePriceByType(ctx context.Context, priceType entity.PriceType, updateDTO dto.UpdatePriceDTO) (*entity.Price, error) {
	if !priceType.IsValid() {
		return nil, &exception.InvalidInputError{Message: fmt.Sprintf("invalid price type: %s", priceType)}
	}
	if updateDTO.Value < 0 {
		return nil, &exception.InvalidInputError{Message: "price value cannot be negative"}
	}
	if updateDTO.Description == "" {
		return nil, &exception.InvalidInputError{Message: "price description cannot be empty"}
	}

	price, err := s.repo.FindByPriceType(ctx, priceType)
	if err != nil {
		return nil, fmt.Errorf("failed to find price by type: %w", err)
	}
	now := time.Now()
	if price == nil {
		price = &entity.Price{
			PriceType: priceType,
			CreatedAt: now,
		}
	}
	if price.ID == "" {
		price.ID = uuid.NewString()
	}
	price.Value = updateDTO.Value
	price.Description = updateDTO.Description
	price.UpdatedAt = now

	saved, err := s.repo.Save(ctx, *price)
	if err != nil {
		return nil, fmt.Errorf("failed to save price: %w", err)
	}

	if s.publisher != nil {
		if perr := s.publisher.Publish(ctx, *saved); perr != nil {
			fmt.Printf("WARN: failed to publish price update: %v\n", perr)
		}
	}
	return saved, nil
}
