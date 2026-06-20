package service

import (
	"context"

	"github.com/redis/go-redis/v9"

	"service-app-go/member-service/core/entity"
	"service-app-go/member-service/request/dto"
)

// MemberRequestService reads pending member requests from the Redis hash
// populated by the Kafka consumer, mirroring the Spring MemberRequestService.
type MemberRequestService struct {
	redis *redis.Client
}

// NewMemberRequestService creates a new MemberRequestService.
func NewMemberRequestService(redisClient *redis.Client) *MemberRequestService {
	return &MemberRequestService{redis: redisClient}
}

// GetNewMemberRequests returns all pending member requests stored in the Redis
// hash "member-requests" (field=email, value=serviceType).
func (s *MemberRequestService) GetNewMemberRequests(ctx context.Context) ([]dto.MemberRequestEvent, error) {
	entries, err := s.redis.HGetAll(ctx, memberRequestsHashKey).Result()
	if err != nil {
		return nil, err
	}

	requests := make([]dto.MemberRequestEvent, 0, len(entries))
	for email, serviceTypeStr := range entries {
		st, ok := entity.ParseServiceType(serviceTypeStr)
		if !ok {
			st = entity.ServiceTypeFree
		}
		requests = append(requests, dto.MemberRequestEvent{
			Email:       email,
			ServiceType: st,
		})
	}
	return requests, nil
}
