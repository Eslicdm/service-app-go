package service

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"service-app-go/member-service/request/dto"
)

const (
	memberRequestsHashKey = "member-requests"
	defaultKafkaBrokers    = "localhost:9092"
	defaultTopic           = "member.requests.topic"
	defaultGroupID         = "member-service-group"
)

// MemberExistenceChecker reports whether a member with a given email exists.
// Implemented by member/repository.MemberRepository.EmailExists.
type MemberExistenceChecker interface {
	EmailExists(email string) (bool, error)
}

// MemberRequestConsumer consumes member request events from Kafka and stores
// new (non-existing) requests in a Redis hash, mirroring the Spring
// MemberRequestConsumer (@KafkaListener on member.requests.topic).
type MemberRequestConsumer struct {
	reader  *kafka.Reader
	redis   *redis.Client
	members MemberExistenceChecker
}

// NewMemberRequestConsumer creates a Kafka reader for the member.requests.topic
// topic, group member-service-group.
func NewMemberRequestConsumer(redisClient *redis.Client, members MemberExistenceChecker) *MemberRequestConsumer {
	brokers := strings.Split(envOrDefault("KAFKA_BROKERS", defaultKafkaBrokers), ",")
	topic := envOrDefault("KAFKA_TOPIC", defaultTopic)
	groupID := envOrDefault("KAFKA_GROUP_ID", defaultGroupID)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &MemberRequestConsumer{
		reader:  r,
		redis:   redisClient,
		members: members,
	}
}

// StartConsuming reads messages until ctx is cancelled.
func (c *MemberRequestConsumer) StartConsuming(ctx context.Context) {
	log.Printf(" [*] Waiting for member requests on topic %s (group %s)", c.reader.Config().Topic, c.reader.Config().GroupID)

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error reading kafka message: %v", err)
			continue
		}

		var event dto.MemberRequestEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error unmarshalling member request event: %v", err)
			continue
		}

		exists, err := c.members.EmailExists(event.Email)
		if err != nil {
			log.Printf("Error checking member existence for %s: %v", event.Email, err)
			continue
		}
		if exists {
			log.Printf("Email %s already exists as a member. Ignoring request.", event.Email)
			continue
		}

		if err := c.redis.HSet(ctx, memberRequestsHashKey, event.Email, string(event.ServiceType)).Err(); err != nil {
			log.Printf("Error storing member request for %s in Redis: %v", event.Email, err)
			continue
		}
		log.Printf("Stored new member request for %s in Redis.", event.Email)
	}
}

// Close closes the Kafka reader.
func (c *MemberRequestConsumer) Close() error {
	return c.reader.Close()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
