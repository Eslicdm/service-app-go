package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"service-app-go/member-service/pricing/dto"
)

// PricingServiceClient calls the pricing-service to fetch the current prices.
// Mirrors the Spring PricingServiceClient (RestClient GET /prices).
type PricingServiceClient struct {
	baseURL string
	http    *http.Client
}

// NewPricingServiceClient creates a client targeting {baseURL}/prices.
// baseURL is e.g. "http://pricing-service:8082/api/v1".
func NewPricingServiceClient(baseURL string) *PricingServiceClient {
	return &PricingServiceClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// GetAllPrices calls GET {baseURL}/prices and decodes the response.
func (c *PricingServiceClient) GetAllPrices(ctx context.Context) ([]dto.PriceUpdateEventDTO, error) {
	url := c.baseURL + "/prices"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build prices request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call pricing-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing-service returned status %d", resp.StatusCode)
	}

	var prices []dto.PriceUpdateEventDTO
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("failed to decode prices response: %w", err)
	}
	return prices, nil
}
