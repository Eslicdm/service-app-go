package service

import (
	"context"
	"errors"
	"fmt"
	"strconv" // Added import for strconv
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/pricing/dto"
	"service-app-go/pricing-service/pricing/repository"
)

const (
	errPriceNotFound    = "price with ID %s not found"
	errPriceTypeNotFound = "price with type %s not found"
)

// PriceService defines the business logic for price operations.
type PriceService interface {
	CreatePrice(ctx context.Context, createDTO dto.CreatePriceDTO) (*entity.Price, error)
	GetPriceByID(ctx context.Context, id string) (*entity.Price, error)
	GetAllPrices(ctx context.Context) ([]entity.Price, error)
	UpdatePrice(ctx context.Context, id string, updateDTO dto.UpdatePriceDTO) (*entity.Price, error)
	UpdatePriceByType(ctx context.Context, priceType string, updateDTO dto.UpdatePriceDTO) (*entity.Price, error)
	DeletePrice(ctx context.Context, id string) error
}

// priceService implements the PriceService interface.
type priceService struct {
	repo repository.PriceRepository
}

// NewPriceService creates a new instance of PriceService.
func NewPriceService(repo repository.PriceRepository) PriceService {
	return &priceService{
		repo: repo,
	}
}

// CreatePrice creates a new price.
func (s *priceService) CreatePrice(ctx context.Context, createDTO dto.CreatePriceDTO) (*entity.Price, error) {
	if !createDTO.PriceType.IsValid() {
		return nil, &exception.InvalidInputError{Message: "invalid price type"}
	}
	if createDTO.Value <= 0 {
		return nil, &exception.InvalidInputError{Message: "price value must be positive"}
	}
	if createDTO.Description == "" {
		return nil, &exception.InvalidInputError{Message: "price description cannot be empty"}
	}

	// Convert float64 to string with high precision, then parse to Decimal128
	decimalValueStr := strconv.FormatFloat(createDTO.Value, 'f', -1, 64)
	decimalValue, err := primitive.ParseDecimal128(decimalValueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimal value from float64: %w", err)
	}

	price := entity.Price{
		ID:          uuid.New().String(),
		PriceType:   createDTO.PriceType,
		Value:       decimalValue,
		Description: createDTO.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.repo.Save(ctx, price)
}

// GetPriceByID retrieves a price by its ID.
func (s *priceService) GetPriceByID(ctx context.Context, id string) (*entity.Price, error) {
	price, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
		}
		return nil, err
	}
	if price == nil {
		return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
	}
	return price, nil
}

// GetAllPrices retrieves all prices.
func (s *priceService) GetAllPrices(ctx context.Context) ([]entity.Price, error) {
	return s.repo.FindAll(ctx)
}

// UpdatePrice updates an existing price by ID.
func (s *priceService) UpdatePrice(ctx context.Context, id string, updateDTO dto.UpdatePriceDTO) (*entity.Price, error) {
	existingPrice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
		}
		return nil, err
	}
	if existingPrice == nil {
		return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
	}

	if updateDTO.Value <= 0 {
		return nil, &exception.InvalidInputError{Message: "price value must be positive"}
	}
	if updateDTO.Description == "" {
		return nil, &exception.InvalidInputError{Message: "price description cannot be empty"}
	}

	// Convert float64 to string with high precision, then parse to Decimal128
	decimalValueStr := strconv.FormatFloat(updateDTO.Value, 'f', -1, 64)
	decimalValue, err := primitive.ParseDecimal128(decimalValueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimal value from float64: %w", err)
	}
	existingPrice.Value = decimalValue
	existingPrice.Description = updateDTO.Description
	existingPrice.UpdatedAt = time.Now()

	return s.repo.Save(ctx, *existingPrice)
}

// UpdatePriceByType updates an existing price by its PriceType.
func (s *priceService) UpdatePriceByType(ctx context.Context, priceTypeStr string, updateDTO dto.UpdatePriceDTO) (*entity.Price, error) {
	var priceType entity.PriceType
	if err := priceType.UnmarshalJSON([]byte(fmt.Sprintf(`"%s"`, priceTypeStr))); err != nil {
		return nil, &exception.InvalidInputError{Message: fmt.Sprintf("invalid price type: %s", priceTypeStr)}
	}

	existingPrice, err := s.repo.FindByPriceType(ctx, priceType)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceTypeNotFound, priceTypeStr)}
		}
		return nil, err
	}
	if existingPrice == nil {
		return nil, &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceTypeNotFound, priceTypeStr)}
	}

	if updateDTO.Value <= 0 {
		return nil, &exception.InvalidInputError{Message: "price value must be positive"}
	}
	if updateDTO.Description == "" {
		return nil, &exception.InvalidInputError{Message: "price description cannot be empty"}
	}

	// Convert float64 to string with high precision, then parse to Decimal128
	decimalValueStr := strconv.FormatFloat(updateDTO.Value, 'f', -1, 64)
	decimalValue, err := primitive.ParseDecimal128(decimalValueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decimal value from float64: %w", err)
	}
	existingPrice.Value = decimalValue
	existingPrice.Description = updateDTO.Description
	existingPrice.UpdatedAt = time.Now()

	return s.repo.Save(ctx, *existingPrice)
}

// DeletePrice deletes a price by its ID.
func (s *priceService) DeletePrice(ctx context.Context, id string) error {
	existingPrice, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
		}
		return err
	}
	if existingPrice == nil {
		return &exception.PriceNotFoundError{Message: fmt.Sprintf(errPriceNotFound, id)}
	}

	return s.repo.Delete(ctx, id)
}