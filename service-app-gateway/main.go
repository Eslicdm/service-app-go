package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"service-app-go/service-app-gateway/controller"
	"service-app-go/service-app-gateway/core/config"
	"service-app-go/service-app-gateway/core/observability"
	"service-app-go/service-app-gateway/proxy"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- OpenTelemetry ---
	otelEndpoint := envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	otelShutdown, err := observability.SetupOTel(ctx, "service-app-gateway", otelEndpoint)
	if err != nil {
		log.Printf("WARN: OTel setup failed: %v (traces/metrics disabled)", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				log.Printf("OTel shutdown error: %v", err)
			}
		}()
	}

	// --- Security ---
	issuer := envOrDefault("KEYCLOAK_REALM_URL", "http://keycloak:8080/realms/service-app-realm")
	securityConfig := config.NewSecurityConfig(issuer)
	authMiddleware := securityConfig.AuthMiddleware()

	// --- Rate limiter (per-user token bucket: 10/s, burst 20) ---
	rateLimiter := proxy.NewRateLimiter()

	// --- Routes (path prefix -> downstream service) ---
	routes := []proxy.Route{
		{
			ID:          "member-request-service-api",
			PathPrefix:  "/api/v1/member-requests",
			TargetURL:   envOrDefault("MEMBER_REQUEST_SERVICE_URL", "http://localhost:8084"),
			ServiceName: "member-request-service",
		},
		{
			ID:          "member-service-api",
			PathPrefix:  "/api/v1/members",
			TargetURL:   envOrDefault("MEMBER_SERVICE_URL", "http://localhost:8081"),
			ServiceName: "member-service",
		},
		{
			ID:          "pricing-service-api",
			PathPrefix:  "/api/v1/prices",
			TargetURL:   envOrDefault("PRICING_SERVICE_URL", "http://localhost:8082"),
			ServiceName: "pricing-service",
		},
	}

	gateway := proxy.NewGateway(routes, rateLimiter)
	fallbackController := controller.NewFallbackController()

	// --- HTTP server ---
	r := gin.Default()
	r.Use(observability.GinMiddleware("service-app-gateway"))

	// CORS (Angular dev server).
	corsOrigin := envOrDefault("CORS_ALLOWED_ORIGIN", "http://localhost:4200")
	r.Use(config.CORSConfig(corsOrigin))

	// Health check (public).
	r.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
	r.GET("/actuator/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"app": "service-app-gateway", "version": "1.0.0"})
	})

	// Fallback endpoints (public).
	fallback := r.Group("/fallback")
	{
		fallback.GET("/:service", fallbackController.Fallback)
		fallback.POST("/:service", fallbackController.Fallback)
	}

	// API routes: auth + rate limit + proxy.
	api := r.Group("/")
	api.Use(authMiddleware)
	api.Use(rateLimiter.Middleware())
	gateway.RegisterRoutes(api)

	port := envOrDefault("PORT", ":8090")
	go func() {
		fmt.Printf("Service App Gateway starting on port %s\n", port)
		if err := r.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	// Wait for interrupt signal.
	<-ctx.Done()
	log.Println("Shutting down gateway gracefully...")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
