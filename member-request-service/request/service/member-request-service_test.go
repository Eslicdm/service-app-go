package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"service-app-go/member-request-service/request/dto"
)

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) Send(ctx context.Context, request dto.MemberRequestDTO) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func setupMiniRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr
}

func TestMemberRequestService_ProcessSubmission_NewRequest(t *testing.T) {
	client, _ := setupMiniRedis(t)
	mockProducer := new(MockKafkaProducer)
	svc := NewMemberRequestService(client, mockProducer)

	req := dto.MemberRequestDTO{
		Email:       "new.prospect@example.com",
		ServiceType: dto.ServiceTypeFullPrice,
	}

	mockProducer.On("Send", mock.Anything, req).Return(nil).Once()

	err := svc.ProcessSubmission(context.Background(), req)
	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestMemberRequestService_ProcessSubmission_DuplicateRequest(t *testing.T) {
	client, _ := setupMiniRedis(t)
	mockProducer := new(MockKafkaProducer)
	svc := NewMemberRequestService(client, mockProducer)

	req := dto.MemberRequestDTO{
		Email:       "dup.prospect@example.com",
		ServiceType: dto.ServiceTypeFree,
	}

	// First submission should succeed and produce.
	mockProducer.On("Send", mock.Anything, req).Return(nil).Once()
	err := svc.ProcessSubmission(context.Background(), req)
	assert.NoError(t, err)

	// Second submission within 5 min should be silently ignored (no Kafka produce).
	err = svc.ProcessSubmission(context.Background(), req)
	assert.NoError(t, err)

	// Producer should only have been called once (for the first submission).
	mockProducer.AssertExpectations(t)
}

func TestMemberRequestService_ProcessSubmission_DifferentEmails(t *testing.T) {
	client, _ := setupMiniRedis(t)
	mockProducer := new(MockKafkaProducer)
	svc := NewMemberRequestService(client, mockProducer)

	req1 := dto.MemberRequestDTO{Email: "user1@example.com", ServiceType: dto.ServiceTypeFree}
	req2 := dto.MemberRequestDTO{Email: "user2@example.com", ServiceType: dto.ServiceTypeHalfPrice}

	mockProducer.On("Send", mock.Anything, req1).Return(nil).Once()
	mockProducer.On("Send", mock.Anything, req2).Return(nil).Once()

	err := svc.ProcessSubmission(context.Background(), req1)
	assert.NoError(t, err)

	err = svc.ProcessSubmission(context.Background(), req2)
	assert.NoError(t, err)

	mockProducer.AssertExpectations(t)
}
