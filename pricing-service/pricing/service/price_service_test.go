package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/pricing/dto"
)

// MockPriceRepository is a mock implementation of the PriceRepository interface.
type MockPriceRepository struct {
	mock.Mock
}

func (m *MockPriceRepository) Save(ctx context.Context, price entity.Price) (*entity.Price, error) {
	args := m.Called(ctx, price)
	var p *entity.Price
	if v := args.Get(0); v != nil {
		switch vv := v.(type) {
		case *entity.Price:
			p = vv
		case func() *entity.Price:
			p = vv()
		}
	}
	return p, args.Error(1)
}

func (m *MockPriceRepository) FindByPriceType(ctx context.Context, priceType entity.PriceType) (*entity.Price, error) {
	args := m.Called(ctx, priceType)
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

func TestPriceService_GetAllPrices(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expected := []entity.Price{
			{ID: "1", PriceType: entity.PriceTypeFree, Value: 0, Description: "Free"},
			{ID: "2", PriceType: entity.PriceTypeHalfPrice, Value: 49.99, Description: "Half"},
		}
		mockRepo.On("FindAll", ctx).Return(expected, nil).Once()

		prices, err := priceService.GetAllPrices(ctx)
		assert.NoError(t, err)
		assert.Len(t, prices, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("FindAll", ctx).Return(nil, errors.New("db error")).Once()
		prices, err := priceService.GetAllPrices(ctx)
		assert.Error(t, err)
		assert.Nil(t, prices)
		mockRepo.AssertExpectations(t)
	})
}

func TestPriceService_UpdatePriceByType(t *testing.T) {
	mockRepo := new(MockPriceRepository)
	priceService := NewPriceService(mockRepo, nil)
	ctx := context.Background()

	t.Run("upsert existing", func(t *testing.T) {
		existing := &entity.Price{ID: "2", PriceType: entity.PriceTypeHalfPrice, Value: 0, Description: "old"}
		mockRepo.On("FindByPriceType", ctx, entity.PriceTypeHalfPrice).Return(existing, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).Return(existing, nil).Once()

		result, err := priceService.UpdatePriceByType(ctx, entity.PriceTypeHalfPrice, dto.UpdatePriceDTO{Value: 49.99, Description: "Club Membership"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 49.99, result.Value)
		assert.Equal(t, "Club Membership", result.Description)
		mockRepo.AssertExpectations(t)
	})

	t.Run("create when not found", func(t *testing.T) {
		mockRepo.On("FindByPriceType", ctx, entity.PriceTypeFree).Return(nil, nil).Once()
		var captured entity.Price
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(entity.Price)
			}).
			Return(func() *entity.Price { return &captured }, nil).Once()

		result, err := priceService.UpdatePriceByType(ctx, entity.PriceTypeFree, dto.UpdatePriceDTO{Value: 0, Description: "Garden Pass"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0.0, result.Value)
		assert.False(t, result.CreatedAt.IsZero())
		assert.NotEmpty(t, captured.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("negative value rejected", func(t *testing.T) {
		result, err := priceService.UpdatePriceByType(ctx, entity.PriceTypeFullPrice, dto.UpdatePriceDTO{Value: -1, Description: "x"})
		assert.Nil(t, result)
		assert.IsType(t, &exception.InvalidInputError{}, err)
	})

	t.Run("empty description rejected", func(t *testing.T) {
		result, err := priceService.UpdatePriceByType(ctx, entity.PriceTypeFullPrice, dto.UpdatePriceDTO{Value: 10, Description: ""})
		assert.Nil(t, result)
		assert.IsType(t, &exception.InvalidInputError{}, err)
	})

	t.Run("free tier value 0 allowed", func(t *testing.T) {
		mockRepo.On("FindByPriceType", ctx, entity.PriceTypeFree).Return(nil, nil).Once()
		var captured entity.Price
		mockRepo.On("Save", ctx, mock.AnythingOfType("entity.Price")).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(entity.Price)
			}).
			Return(func() *entity.Price { return &captured }, nil).Once()
		result, err := priceService.UpdatePriceByType(ctx, entity.PriceTypeFree, dto.UpdatePriceDTO{Value: 0, Description: "Garden Pass"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})
}
