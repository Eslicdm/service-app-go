package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"service-app-go/member-service/pricing/client"
	"service-app-go/member-service/pricing/dto"
)

const (
	priceUpdateKeyPrefix = "price-update:"
)

// PriceCacheService implements a cache-aside for prices in Redis, mirroring the
// Spring PriceCacheService. Keys: price-update:<priceType>. getAllPrices does a
// multiGet over the 3 known keys; on a miss it fetches from the pricing-service
// and populates the cache. cachePriceUpdate writes a single entry (used by the
// RabbitMQ listener for push-based cache refresh).
type PriceCacheService struct {
	redis  *redis.Client
	client *client.PricingServiceClient
}

// NewPriceCacheService creates a new PriceCacheService.
func NewPriceCacheService(redisClient *redis.Client, pricingClient *client.PricingServiceClient) *PriceCacheService {
	return &PriceCacheService{
		redis:  redisClient,
		client: pricingClient,
	}
}

func cacheKey(pt dto.PriceType) string {
	return priceUpdateKeyPrefix + string(pt)
}

// CachePriceUpdate stores a single price update event in Redis (no expiration,
// matching the Spring behavior which relies on push invalidation via RabbitMQ).
func (s *PriceCacheService) CachePriceUpdate(ctx context.Context, price dto.PriceUpdateEventDTO) error {
	body, err := json.Marshal(price)
	if err != nil {
		return fmt.Errorf("failed to marshal price update: %w", err)
	}
	if err := s.redis.Set(ctx, cacheKey(price.PriceType), body, 0).Err(); err != nil {
		return fmt.Errorf("failed to cache price update: %w", err)
	}
	log.Printf("Cached price update for %s", price.PriceType)
	return nil
}

// GetAllPrices returns the 3 price tiers from Redis, fetching from the
// pricing-service on a cache miss and back-filling the cache.
func (s *PriceCacheService) GetAllPrices(ctx context.Context) ([]dto.PriceUpdateEventDTO, error) {
	types := dto.AllPriceTypes()
	keys := make([]string, len(types))
	for i, t := range types {
		keys[i] = cacheKey(t)
	}

	results, err := s.redis.MGet(ctx, keys...).Result()
	if err != nil {
		log.Printf("Redis MGet error, falling back to pricing-service: %v", err)
		return s.fetchAndCache(ctx)
	}

	allCached := true
	prices := make([]dto.PriceUpdateEventDTO, 0, len(types))
	for _, r := range results {
		if r == nil {
			allCached = false
			break
		}
		str, ok := r.(string)
		if !ok {
			allCached = false
			break
		}
		var p dto.PriceUpdateEventDTO
		if err := json.Unmarshal([]byte(str), &p); err != nil {
			allCached = false
			break
		}
		prices = append(prices, p)
	}

	if allCached && len(prices) == len(types) {
		log.Println("Returning all prices from cache.")
		return prices, nil
	}

	log.Println("Cache miss for some prices, fetching from pricing-service.")
	return s.fetchAndCache(ctx)
}

func (s *PriceCacheService) fetchAndCache(ctx context.Context) ([]dto.PriceUpdateEventDTO, error) {
	fresh, err := s.client.GetAllPrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices from pricing-service: %w", err)
	}
	for _, p := range fresh {
		if cErr := s.CachePriceUpdate(ctx, p); cErr != nil {
			log.Printf("WARN: failed to back-fill cache for %s: %v", p.PriceType, cErr)
		}
	}
	return fresh, nil
}
