package config

import (
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// PricingExchange is the topic exchange declared by the pricing-service.
	PricingExchange = "pricing.exchange"
	// PriceUpdatedQueue is the member-service queue bound to the exchange.
	PriceUpdatedQueue = "queue.price-updated.member-service"
	// PriceUpdatedRoutingKey is the routing key used by the pricing-service publisher.
	PriceUpdatedRoutingKey = "price.updated.key"
)

// RabbitMQConfig holds the AMQP connection and channel used to consume price
// update events. Mirrors the Spring RabbitMQConfig (topic exchange + queue + binding).
type RabbitMQConfig struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQConfig connects to RabbitMQ and declares the exchange, the
// member-service queue and the binding.
func NewRabbitMQConfig() (*RabbitMQConfig, error) {
	username := os.Getenv("RABBITMQ_USERNAME")
	password := os.Getenv("RABBITMQ_PW")
	host := envOrDefault("RABBITMQ_HOST", "localhost")
	port := envOrDefault("RABBITMQ_PORT", "5672")

	if username == "" || password == "" {
		return nil, fmt.Errorf("RabbitMQ credentials not set: RABBITMQ_USERNAME, RABBITMQ_PW")
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
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	if err := channel.ExchangeDeclare(PricingExchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}
	if _, err := channel.QueueDeclare(PriceUpdatedQueue, true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}
	if err := channel.QueueBind(PriceUpdatedQueue, PriceUpdatedRoutingKey, PricingExchange, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &RabbitMQConfig{conn: conn, channel: channel}, nil
}

// Channel returns the AMQP channel (used by the consumer).
func (c *RabbitMQConfig) Channel() *amqp.Channel {
	return c.channel
}

// Close closes the channel and connection.
func (c *RabbitMQConfig) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("RabbitMQ consumer connection closed.")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
