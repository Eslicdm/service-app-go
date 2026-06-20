package service

import (
	"context"
	"encoding/json"
	"log"

	"service-app-go/member-service/pricing/config"
	"service-app-go/member-service/pricing/dto"
)

// PriceUpdateListener consumes price update events from RabbitMQ and refreshes
// the Redis cache. Mirrors the Spring PriceUpdateListener (@RabbitListener on
// queue.price-updated.member-service).
type PriceUpdateListener struct {
	rmq   *config.RabbitMQConfig
	cache *PriceCacheService
}

// NewPriceUpdateListener creates a new listener backed by the given RabbitMQ config.
func NewPriceUpdateListener(rmq *config.RabbitMQConfig, cache *PriceCacheService) *PriceUpdateListener {
	return &PriceUpdateListener{rmq: rmq, cache: cache}
}

// StartConsuming starts consuming messages until ctx is cancelled.
func (l *PriceUpdateListener) StartConsuming(ctx context.Context) error {
	msgs, err := l.rmq.Channel().Consume(
		config.PriceUpdatedQueue,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf(" [*] Waiting for price updates on %s", config.PriceUpdatedQueue)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping price update listener...")
			return nil
		case d, ok := <-msgs:
			if !ok {
				log.Println("RabbitMQ channel closed, stopping price update listener.")
				return nil
			}
			var price dto.PriceUpdateEventDTO
			if err := json.Unmarshal(d.Body, &price); err != nil {
				log.Printf("Error unmarshalling price update: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			if err := l.cache.CachePriceUpdate(ctx, price); err != nil {
				log.Printf("Error caching price update: %v", err)
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
