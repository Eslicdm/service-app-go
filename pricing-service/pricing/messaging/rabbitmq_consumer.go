package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv" // Added import for strconv
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"service-app-go/pricing-service/core/entity"
	"service-app-go/pricing-service/pricing/dto" // Import the dto package
	"service-app-go/pricing-service/pricing/service"
)

const (
	rabbitmqUsernameEnv    = "RABBITMQ_USERNAME"
	rabbitmqPasswordEnv    = "RABBITMQ_PW"
	rabbitmqHost           = "localhost" // This should match the service name in docker-compose
	rabbitmqPort           = "5672"
	pricingExchange        = "pricing.exchange"
	priceUpdatedRoutingKey = "price.updated.key"
	priceUpdateQueue       = "price.update.queue" // A dedicated queue for this service
)

// RabbitMQConsumer listens for messages from RabbitMQ.
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	service service.PriceService
}

// NewRabbitMQConsumer creates a new RabbitMQConsumer.
func NewRabbitMQConsumer(priceService service.PriceService) (*RabbitMQConsumer, error) {
	username := os.Getenv(rabbitmqUsernameEnv)
	password := os.Getenv(rabbitmqPasswordEnv)

	if username == "" || password == "" {
		return nil, fmt.Errorf("RabbitMQ credentials not set: %s, %s", rabbitmqUsernameEnv, rabbitmqPasswordEnv)
	}

	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/", username, password, rabbitmqHost, rabbitmqPort)

	var conn *amqp.Connection
	var err error
	// Retry connection with a backoff
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
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the exchange
	err = channel.ExchangeDeclare(
		pricingExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare an exchange: %w", err)
	}

	// Declare the queue
	queue, err := channel.QueueDeclare(
		priceUpdateQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Bind the queue to the exchange with the routing key
	err = channel.QueueBind(
		queue.Name,
		priceUpdatedRoutingKey,
		pricingExchange,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue to exchange: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: channel,
		service: priceService,
	}, nil
}

// StartConsuming starts consuming messages from the RabbitMQ queue.
func (c *RabbitMQConsumer) StartConsuming(ctx context.Context) {
	msgs, err := c.channel.Consume(
		priceUpdateQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Printf(" [*] Waiting for messages in %s. To exit press CTRL+C", priceUpdateQueue)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping RabbitMQ consumer...")
			return
		case d := <-msgs:
			log.Printf(" [x] Received a message: %s", d.Body)
			var price entity.Price // Unmarshal into entity.Price as it contains the ID
			if err := json.Unmarshal(d.Body, &price); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				d.Nack(false, false) // Nack the message, don't requeue
				continue
			}

			// Convert primitive.Decimal128 to float64
			valueFloat, err := strconv.ParseFloat(price.Value.String(), 64)
			if err != nil {
				log.Printf("Error converting Decimal128 to float64 for price ID %s: %v", price.ID, err)
				d.Nack(false, false) // Nack the message, don't requeue
				continue
			}

			// Create an UpdatePriceDTO from the unmarshalled entity.Price
			updateDTO := dto.UpdatePriceDTO{
				Value:       valueFloat,
				Description: price.Description,
			}

			// Attempt to update the price using the service
			_, err = c.service.UpdatePrice(ctx, price.ID, updateDTO)
			if err != nil {
				log.Printf("Error updating price from RabbitMQ message (ID: %s): %v", price.ID, err)
				d.Nack(false, true) // Nack and requeue for retry
				continue
			}

			log.Printf(" [x] Price (ID: %s) updated successfully from RabbitMQ.", price.ID)
			d.Ack(false) // Acknowledge the message
		}
	}
}

// Close closes the RabbitMQ connection and channel.
func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("RabbitMQ connection closed.")
}
