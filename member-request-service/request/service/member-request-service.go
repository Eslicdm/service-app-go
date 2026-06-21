package service

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"service-app-go/member-request-service/request/dto"
)

const (
	submissionCachePrefix = "submission:"
	submissionCacheTTL    = 5 * time.Minute
)

// Producer defines the interface for publishing member request events.
// This enables unit testing with a mock implementation.
type Producer interface {
	Send(ctx context.Context, request dto.MemberRequestDTO) error
}

// MemberRequestService processes prospect submissions, mirroring the Spring
// MemberRequestService.processSubmission: Redis SETNX dedup (5 min TTL),
// then produce to Kafka if new.
type MemberRequestService struct {
	redis    *redis.Client
	producer Producer
}

// NewMemberRequestService creates a new MemberRequestService.
func NewMemberRequestService(redisClient *redis.Client, producer Producer) *MemberRequestService {
	return &MemberRequestService{
		redis:    redisClient,
		producer: producer,
	}
}

// ProcessSubmission deduplicates by email (Redis SETNX with 5 min TTL) and
// produces a Kafka event on first submission. Duplicate submissions within
// the TTL window are silently ignored (matching Spring behavior).
func (s *MemberRequestService) ProcessSubmission(ctx context.Context, request dto.MemberRequestDTO) error {
	cacheKey := submissionCachePrefix + request.Email

	wasSet, err := s.redis.SetNX(ctx, cacheKey, "processed", submissionCacheTTL).Result()
	if err != nil {
		log.Printf("Redis SetNX error for %s: %v", request.Email, err)
		return err
	}

	if wasSet {
		log.Printf("New submission for email: %s. Sending to Kafka.", request.Email)
		return s.producer.Send(ctx, request)
	}

	log.Printf("Duplicate submission detected within 5 minutes for email: %s. Ignoring.", request.Email)
	return nil
}
