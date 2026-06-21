package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"service-app-go/member-service/core/entity"
	"service-app-go/member-service/core/exception"
	"service-app-go/member-service/member/dto"
)

type MockMemberRepository struct {
	mock.Mock
}

func (m *MockMemberRepository) FindAllByManagerID(managerID string) ([]entity.Member, error) {
	args := m.Called(managerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Member), args.Error(1)
}

func (m *MockMemberRepository) FindByEmail(email string) (*entity.Member, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Member), args.Error(1)
}

func (m *MockMemberRepository) FindByID(id uint) (*entity.Member, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Member), args.Error(1)
}

func (m *MockMemberRepository) Save(member *entity.Member) (*entity.Member, error) {
	args := m.Called(member)
	var result *entity.Member
	if v := args.Get(0); v != nil {
		switch vv := v.(type) {
		case *entity.Member:
			result = vv
		case func(*entity.Member) *entity.Member:
			result = vv(member)
		}
	}
	return result, args.Error(1)
}

func (m *MockMemberRepository) ExistsByID(id uint) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func (m *MockMemberRepository) DeleteByID(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestMemberService_GetAllMembersByManagerID(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)
	managerID := "manager-123"

	t.Run("success", func(t *testing.T) {
		expected := []entity.Member{
			{ID: 1, Name: "John", Email: "john@test.com", ManagerID: managerID},
			{ID: 2, Name: "Jane", Email: "jane@test.com", ManagerID: managerID},
		}
		mockRepo.On("FindAllByManagerID", managerID).Return(expected, nil).Once()

		members, err := svc.GetAllMembersByManagerID(managerID)
		assert.NoError(t, err)
		assert.Len(t, members, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("FindAllByManagerID", managerID).Return(nil, errors.New("db error")).Once()

		members, err := svc.GetAllMembersByManagerID(managerID)
		assert.Error(t, err)
		assert.Nil(t, members)
		mockRepo.AssertExpectations(t)
	})
}

func TestMemberService_GetMemberByID(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)
	managerID := "manager-123"
	memberID := uint(1)

	t.Run("success", func(t *testing.T) {
		member := &entity.Member{ID: memberID, Name: "John", ManagerID: managerID}
		mockRepo.On("FindByID", memberID).Return(member, nil).Once()

		result, err := svc.GetMemberByID(managerID, memberID)
		assert.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.On("FindByID", memberID).Return(nil, gorm.ErrRecordNotFound).Once()

		result, err := svc.GetMemberByID(managerID, memberID)
		assert.Nil(t, result)
		assert.IsType(t, &exception.EntityNotFoundError{}, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("access denied - different manager", func(t *testing.T) {
		member := &entity.Member{ID: memberID, Name: "John", ManagerID: "other-manager"}
		mockRepo.On("FindByID", memberID).Return(member, nil).Once()

		result, err := svc.GetMemberByID(managerID, memberID)
		assert.Nil(t, result)
		assert.IsType(t, &exception.AccessDeniedError{}, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestMemberService_CreateMember(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)
	managerID := "manager-123"
	birthDate := time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC)
	req := dto.CreateMemberRequest{
		Name:        "John Doe",
		Email:       "john.doe@test.com",
		BirthDate:   birthDate,
		ServiceType: entity.ServiceTypeFree,
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("FindByEmail", req.Email).Return(nil, gorm.ErrRecordNotFound).Once()
		mockRepo.On("Save", mock.AnythingOfType("*entity.Member")).Return(func(m *entity.Member) *entity.Member {
			m.ID = 1
			return m
		}, nil).Once()

		result, err := svc.CreateMember(managerID, req)
		assert.NoError(t, err)
		assert.Equal(t, "John Doe", result.Name)
		assert.Equal(t, managerID, result.ManagerID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("duplicate email", func(t *testing.T) {
		existing := &entity.Member{ID: 1, Email: req.Email}
		mockRepo.On("FindByEmail", req.Email).Return(existing, nil).Once()

		result, err := svc.CreateMember(managerID, req)
		assert.Nil(t, result)
		assert.IsType(t, &exception.DuplicateEmailError{}, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestMemberService_UpdateMember(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)
	memberID := uint(1)
	existing := &entity.Member{ID: memberID, Name: "Old Name", Email: "old@test.com"}

	t.Run("partial update - name only", func(t *testing.T) {
		mockRepo.On("FindByID", memberID).Return(existing, nil).Once()
		newName := "New Name"
		mockRepo.On("Save", mock.AnythingOfType("*entity.Member")).Return(func(m *entity.Member) *entity.Member {
			return m
		}, nil).Once()

		result, err := svc.UpdateMember(memberID, dto.UpdateMemberRequest{Name: &newName})
		assert.NoError(t, err)
		assert.Equal(t, "New Name", result.Name)
		assert.Equal(t, "old@test.com", result.Email) // unchanged
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.On("FindByID", memberID).Return(nil, gorm.ErrRecordNotFound).Once()

		result, err := svc.UpdateMember(memberID, dto.UpdateMemberRequest{})
		assert.Nil(t, result)
		assert.IsType(t, &exception.EntityNotFoundError{}, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestMemberService_DeleteMember(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)
	memberID := uint(1)

	t.Run("success", func(t *testing.T) {
		mockRepo.On("ExistsByID", memberID).Return(true, nil).Once()
		mockRepo.On("DeleteByID", memberID).Return(nil).Once()

		err := svc.DeleteMember(memberID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.On("ExistsByID", memberID).Return(false, nil).Once()

		err := svc.DeleteMember(memberID)
		assert.IsType(t, &exception.EntityNotFoundError{}, err)
		mockRepo.AssertExpectations(t)
	})
}
