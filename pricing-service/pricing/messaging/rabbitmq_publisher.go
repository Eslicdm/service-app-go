package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"service-app-go/pricing-service/core/entity"
)

const (
	rabbitmqUsernameEnv    = "RABBITMQ_USERNAME"
	rabbitmqPasswordEnv    = "RABBITMQ_PW"
	defaultRabbitmqHost    = "localhost"
	defaultRabbitmqPort    = "5672"
	pricingExchange        = "pricing.exchange"
	priceUpdatedRoutingKey = "price.updated.key"
)

// PricePublisher publishes price update events to RabbitMQ so that downstream
// services (member-service price cache) can refresh their cache. This mirrors
// the Spring pricing-service's RabbitTemplate.convertAndSend(exchange, routingKey, price).
type PricePublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPricePublisher connects to RabbitMQ and declares the pricing exchange.
func NewPricePublisher() (*PricePublisher, error) {
	username := os.Getenv(rabbitmqUsernameEnv)
	password := os.Getenv(rabbitmqPasswordEnv)
	host := envOrDefault("RABBITMQ_HOST", defaultRabbitmqHost)
	port := envOrDefault("RABBITMQ_PORT", defaultRabbitmqPort)

	if username == "" || password == "" {
		return nil, fmt.Errorf("RabbitMQ credentials not set: %s, %s", rabbitmqUsernameEnv, rabbitmqPasswordEnv)
	}

	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/", username, password, host, port)

	var conn *amqp.Connection
	var err error
	for i := 0; i < 5; i++ {
		conn, err = amqp.Dial(connStr)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ, retrying in %d seconds: %v", 1<<i, err)
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after multiple retries: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	if err := channel.ExchangeDeclare(
		pricingExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &PricePublisher{conn: conn, channel: channel}, nil
}

// Publish marshals the price to JSON and publishes it to the pricing exchange
// with the price.updated.key routing key.
func (p *PricePublisher) Publish(ctx context.Context, price entity.Price) error {
	body, err := json.Marshal(price)
	if err != nil {
		return fmt.Errorf("failed to marshal price: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		publishCtx,
		pricingExchange,
		priceUpdatedRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish price update: %w", err)
	}
	log.Printf("Published price update for type %s to %s/%s", price.PriceType, pricingExchange, priceUpdatedRoutingKey)
	return nil
}

// Close closes the RabbitMQ channel and connection.
func (p *PricePublisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	log.Println("RabbitMQ publisher connection closed.")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
