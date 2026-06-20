package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"service-app-go/member-service/pricing/service"
)

// PriceController exposes the cached prices, mirroring the Spring
// PriceController: GET /api/v1/members/prices.
type PriceController struct {
	cache *service.PriceCacheService
}

// NewPriceController creates a new PriceController.
func NewPriceController(cache *service.PriceCacheService) *PriceController {
	return &PriceController{cache: cache}
}

// GetAllPrices returns the cached prices (cache-aside with pricing-service fallback).
// @Summary Get all prices (cached)
// @Description Returns the 3 pricing tiers from Redis (cache-aside) with pricing-service fallback
// @Tags prices
// @Produce json
// @Success 200 {array} dto.PriceUpdateEventDTO
// @Failure 500 {object} map[string]string
// @Router /members/prices [get]
func (c *PriceController) GetAllPrices(ctx *gin.Context) {
	prices, err := c.cache.GetAllPrices(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, prices)
}
