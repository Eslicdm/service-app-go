package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/core/exception"
)

// MockPriceRepository is a mock implementation of the PriceRepository interface.
type MockPriceRepository struct {
	mock.Mock
}

func (m *MockPriceRepository) Save(ctx context.Context, price entity.Price) (*entity.Price, error) {
	args := m.Called(ctx, price)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Price), args.Error(1)
}

func (m *MockPriceRepository) FindByID(ctx context.Context, id string) (*entity.Price, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Price), args.Error(1)
}

func (m *MockPriceRepository) FindAll(ctx context.Context) ([]entity.Price, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Price), args.Error(1)
}

func (m *MockPriceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestPriceService_CreatePrice(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo)
	ctx := context.Background()

	testPrice := entity.Price{
		PriceType:   entity.PriceTypeFullPrice,
		Value:       100.0,
		Description: "Test Price",
	}

	// Test case 1: Successful creation
	t.Run("success", func(t *testing.T) {
		expectedPrice := testPrice
		expectedPrice.ID = uuid.New().String() // ID will be generated
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).Return(&expectedPrice, nil).Once()

		createdPrice, err := priceService.CreatePrice(ctx, testPrice)
		assert.NoError(t, err)
		assert.NotNil(t, createdPrice)
		assert.NotEmpty(t, createdPrice.ID)
		assert.Equal(t, expectedPrice.Value, createdPrice.Value)
		mockRepo.AssertExpectations(t)
	})

	// Test case 2: Invalid PriceType
	t.Run("invalid price type", func(t *testing.T) {
		invalidPrice := testPrice
		invalidPrice.PriceType = "INVALID"
		createdPrice, err := priceService.CreatePrice(ctx, invalidPrice)
		assert.Error(t, err)
		assert.Nil(t, createdPrice)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "invalid price type", err.Error())
	})

	// Test case 3: Non-positive value
	t.Run("non-positive value", func(t *testing.T) {
		invalidPrice := testPrice
		invalidPrice.Value = 0
		createdPrice, err := priceService.CreatePrice(ctx, invalidPrice)
		assert.Error(t, err)
		assert.Nil(t, createdPrice)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "price value must be positive", err.Error())
	})

	// Test case 4: Empty description
	t.Run("empty description", func(t *testing.T) {
		invalidPrice := testPrice
		invalidPrice.Description = ""
		createdPrice, err := priceService.CreatePrice(ctx, invalidPrice)
		assert.Error(t, err)
		assert.Nil(t, createdPrice)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "price description cannot be empty", err.Error())
	})

	// Test case 5: Repository error
	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).Return(nil, errors.New("db error")).Once()
		createdPrice, err := priceService.CreatePrice(ctx, testPrice)
		assert.Error(t, err)
		assert.Nil(t, createdPrice)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestPriceService_GetPriceByID(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo)
	ctx := context.Background()
	priceID := uuid.New().String()

	// Test case 1: Price found
	t.Run("success", func(t *testing.T) {
		expectedPrice := &entity.Price{ID: priceID, PriceType: entity.PriceTypeFullPrice, Value: 50.0}
		mockRepo.On("FindByID", ctx, priceID).Return(expectedPrice, nil).Once()

		foundPrice, err := priceService.GetPriceByID(ctx, priceID)
		assert.NoError(t, err)
		assert.NotNil(t, foundPrice)
		assert.Equal(t, expectedPrice.ID, foundPrice.ID)
		mockRepo.AssertExpectations(t)
	})

	// Test case 2: Price not found
	t.Run("not found", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, nil).Once() // Simulate no document found
		foundPrice, err := priceService.GetPriceByID(ctx, priceID)
		assert.Error(t, err)
		assert.Nil(t, foundPrice)
		assert.IsType(t, &exception.PriceNotFoundError{}, err)
		assert.Equal(t, fmt.Sprintf("price with ID %s not found", priceID), err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: Repository error
	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, errors.New("db error")).Once()
		foundPrice, err := priceService.GetPriceByID(ctx, priceID)
		assert.Error(t, err)
		assert.Nil(t, foundPrice)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestPriceService_GetAllPrices(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo)
	ctx := context.Background()

	// Test case 1: Prices found
	t.Run("success", func(t *testing.T) {
		expectedPrices := []entity.Price{
			{ID: uuid.New().String(), PriceType: entity.PriceTypeFullPrice, Value: 100.0},
			{ID: uuid.New().String(), PriceType: entity.PriceTypeHalfPrice, Value: 50.0},
		}
		mockRepo.On("FindAll", ctx).Return(expectedPrices, nil).Once()

		prices, err := priceService.GetAllPrices(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, prices)
		assert.Len(t, prices, 2)
		assert.Equal(t, expectedPrices, prices)
		mockRepo.AssertExpectations(t)
	})

	// Test case 2: No prices found
	t.Run("no prices", func(t *testing.T) {
		mockRepo.On("FindAll", ctx).Return([]entity.Price{}, nil).Once()

		prices, err := priceService.GetAllPrices(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, prices)
		assert.Len(t, prices, 0)
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: Repository error
	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("FindAll", ctx).Return(nil, errors.New("db error")).Once()

		prices, err := priceService.GetAllPrices(ctx)
		assert.Error(t, err)
		assert.Nil(t, prices)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestPriceService_UpdatePrice(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo)
	ctx := context.Background()
	priceID := uuid.New().String()

	existingPrice := &entity.Price{
		ID:          priceID,
		PriceType:   entity.PriceTypeFullPrice,
		Value:       100.0,
		Description: "Original Description",
		CreatedAt:   time.Now().Add(-time.Hour),
	}

	updatedPriceInput := entity.Price{
		PriceType:   entity.PriceTypeHalfPrice,
		Value:       50.0,
		Description: "Updated Description",
	}

	// Test case 1: Successful update
	t.Run("success", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(existingPrice, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).Return(func(ctx context.Context, p entity.Price) *entity.Price {
			p.UpdatedAt = time.Now() // Simulate update time
			return &p
		}, nil).Once()

		result, err := priceService.UpdatePrice(ctx, priceID, updatedPriceInput)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, priceID, result.ID)
		assert.Equal(t, updatedPriceInput.Value, result.Value)
		assert.Equal(t, updatedPriceInput.Description, result.Description)
		assert.True(t, result.UpdatedAt.After(existingPrice.CreatedAt))
		mockRepo.AssertExpectations(t)
	})

	// Test case 2: Price not found
	t.Run("not found", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, nil).Once() // Simulate no document found
		result, err := priceService.UpdatePrice(ctx, priceID, updatedPriceInput)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.IsType(t, &exception.PriceNotFoundError{}, err)
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: Invalid PriceType
	t.Run("invalid price type", func(t *testing.T) {
		invalidUpdate := updatedPriceInput
		invalidUpdate.PriceType = "INVALID"
		mockRepo.On("FindByID", ctx, priceID).Return(existingPrice, nil).Once()

		result, err := priceService.UpdatePrice(ctx, priceID, invalidUpdate)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "invalid price type", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 4: Non-positive value
	t.Run("non-positive value", func(t *testing.T) {
		invalidUpdate := updatedPriceInput
		invalidUpdate.Value = 0
		mockRepo.On("FindByID", ctx, priceID).Return(existingPrice, nil).Once()

		result, err := priceService.UpdatePrice(ctx, priceID, invalidUpdate)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "price value must be positive", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 5: Empty description
	t.Run("empty description", func(t *testing.T) {
		invalidUpdate := updatedPriceInput
		invalidUpdate.Description = ""
		mockRepo.On("FindByID", ctx, priceID).Return(existingPrice, nil).Once()

		result, err := priceService.UpdatePrice(ctx, priceID, invalidUpdate)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.IsType(t, &exception.InvalidInputError{}, err)
		assert.Equal(t, "price description cannot be empty", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 6: Repository FindByID error
	t.Run("repository find error", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, errors.New("db find error")).Once()
		result, err := priceService.UpdatePrice(ctx, priceID, updatedPriceInput)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "db find error", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 7: Repository Save error
	t.Run("repository save error", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(existingPrice, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).Return(nil, errors.New("db save error")).Once()
		result, err := priceService.UpdatePrice(ctx, priceID, updatedPriceInput)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "db save error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestPriceService_DeletePrice(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo)
	ctx := context.Background()
	priceID := uuid.New().String()

	// Test case 1: Successful deletion
	t.Run("success", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(&entity.Price{ID: priceID}, nil).Once()
		mockRepo.On("Delete", ctx, priceID).Return(nil).Once()

		err := priceService.DeletePrice(ctx, priceID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// Test case 2: Price not found
	t.Run("not found", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, nil).Once() // Simulate no document found
		err := priceService.DeletePrice(ctx, priceID)
		assert.Error(t, err)
		assert.IsType(t, &exception.PriceNotFoundError{}, err)
		assert.Equal(t, fmt.Sprintf("price with ID %s not found", priceID), err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: Repository FindByID error
	t.Run("repository find error", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(nil, errors.New("db find error")).Once()
		err := priceService.DeletePrice(ctx, priceID)
		assert.Error(t, err)
		assert.Equal(t, "db find error", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// Test case 4: Repository Delete error
	t.Run("repository delete error", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, priceID).Return(&entity.Price{ID: priceID}, nil).Once()
		mockRepo.On("Delete", ctx, priceID).Return(errors.New("db delete error")).Once()
		err := priceService.DeletePrice(ctx, priceID)
		assert.Error(t, err)
		assert.Equal(t, "db delete error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}
