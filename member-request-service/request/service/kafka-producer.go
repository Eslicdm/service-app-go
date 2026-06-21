package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"

	"service-app-go/member-request-service/request/dto"
)

const (
	defaultKafkaBrokers = "localhost:9092"
	defaultTopic        = "member.requests.topic"
)

// KafkaProducer publishes member request events to Kafka, mirroring the Spring
// MemberRequestProducer (kafkaTemplate.send(topic, email, dto)). Uses
// segmentio/kafka-go Writer with JSON values and no type headers
// (Spring spring.json.add.type.headers=false).
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka writer for the member.requests.topic topic.
func NewKafkaProducer() *KafkaProducer {
	brokers := strings.Split(envOrDefault("KAFKA_BROKERS", defaultKafkaBrokers), ",")
	topic := envOrDefault("KAFKA_TOPIC", defaultTopic)

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	return &KafkaProducer{writer: w}
}

// Send marshals the DTO to JSON and writes it to Kafka with key=email.
func (p *KafkaProducer) Send(ctx context.Context, request dto.MemberRequestDTO) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal member request: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(request.Email),
		Value: body,
	})
	if err != nil {
		return fmt.Errorf("failed to produce kafka message: %w", err)
	}
	log.Printf("Sent member request for %s to topic %s", request.Email, p.writer.Topic)
	return nil
}

// Close closes the Kafka writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
