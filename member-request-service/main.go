package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"service-app-go/member-request-service/core/config"
	"service-app-go/member-request-service/core/exception"
	"service-app-go/member-request-service/request/controller"
	"service-app-go/member-request-service/request/service"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("WARN: Redis ping failed: %v (dedup will fail)", err)
	}

	// --- Kafka producer ---
	kafkaProducer := service.NewKafkaProducer()
	defer func() {
		if cerr := kafkaProducer.Close(); cerr != nil {
			log.Printf("Error closing kafka producer: %v", cerr)
		}
	}()

	// --- Dependency injection ---
	requestService := service.NewMemberRequestService(redisClient, kafkaProducer)
	requestController := controller.NewMemberRequestController(requestService)

	// --- Security ---
	issuer := envOrDefault("KEYCLOAK_REALM_URL", "http://keycloak:8080/realms/service-app-realm")
	securityConfig := config.NewSecurityConfig(issuer)
	authMiddleware := securityConfig.AuthMiddleware()

	// --- HTTP server ---
	r := gin.Default()
	r.Use(exception.GlobalExceptionHandler())

	r.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	api := r.Group("/api/v1")
	api.Use(authMiddleware)
	{
		// POST /api/v1/member-requests is public (allowed in isPublicEndpoint by method+path).
		api.POST("/member-requests", requestController.SubmitRequest)
	}

	port := envOrDefault("PORT", ":8084")
	go func() {
		fmt.Printf("Member request service starting on port %s\n", port)
		if err := r.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down member request service gracefully...")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
