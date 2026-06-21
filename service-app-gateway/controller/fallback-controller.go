package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// FallbackController handles /fallback/:service endpoints, mirroring the Spring
// FallbackController: returns 503 SERVICE_UNAVAILABLE with a JSON body.
type FallbackController struct{}

// NewFallbackController creates a new FallbackController.
func NewFallbackController() *FallbackController {
	return &FallbackController{}
}

// Fallback handles GET/POST /fallback/:service.
// @Summary Fallback for unavailable service
// @Description Returns 503 when a downstream service is unavailable (circuit breaker open)
// @Tags fallback
// @Produce json
// @Param service path string true "Service name"
// @Success 503 {object} map[string]interface{}
// @Router /fallback/{service} [get]
// @Router /fallback/{service} [post]
func (fc *FallbackController) Fallback(c *gin.Context) {
	serviceName := c.Param("service")
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":     serviceName + " is temporarily unavailable",
		"message":   "Please try again later or contact support",
		"timestamp": time.Now(),
		"service":   serviceName,
	})
}
