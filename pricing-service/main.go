package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"service-app-go/pricing-service/core/config"
	"service-app-go/pricing-service/core/exception"
	"service-app-go/pricing-service/core/observability"
	"service-app-go/pricing-service/pricing/controller"
	"service-app-go/pricing-service/pricing/messaging"
	"service-app-go/pricing-service/pricing/repository"
	"service-app-go/pricing-service/pricing/service"
)

const (
	dbUsernameEnv = "PRICING_DB_USERNAME"
	dbPasswordEnv = "PRICING_DB_PASSWORD"
	dbNameEnv     = "PRICING_DB_NAME"
	defaultMongo  = "localhost"
	mongoPort     = "27017"
	errEnvNotSet  = "Environment variable %s not set"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- OpenTelemetry ---
	otelEndpoint := envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	otelShutdown, err := observability.SetupOTel(ctx, "pricing-service", otelEndpoint)
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

	dbUsername := os.Getenv(dbUsernameEnv)
	dbPassword := os.Getenv(dbPasswordEnv)
	dbName := os.Getenv(dbNameEnv)
	if dbUsername == "" {
		log.Fatalf(errEnvNotSet, dbUsernameEnv)
	}
	if dbPassword == "" {
		log.Fatalf(errEnvNotSet, dbPasswordEnv)
	}
	if dbName == "" {
		log.Fatalf(errEnvNotSet, dbNameEnv)
	}
	mongoHost := envOrDefault("MONGO_HOST", defaultMongo)

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=admin",
		dbUsername, dbPassword, mongoHost, mongoPort, dbName)
	clientOpts := options.Client().ApplyURI(mongoURI)
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	client, err := mongo.Connect(mongoCtx, clientOpts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if derr := client.Disconnect(context.Background()); derr != nil {
			log.Printf("Failed to disconnect from MongoDB: %v", derr)
		}
	}()

	if err := client.Ping(mongoCtx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	fmt.Println("Successfully connected to MongoDB!")

	priceRepo := repository.NewMongoPriceRepository(client, dbName)

	publisher, err := messaging.NewPricePublisher()
	if err != nil {
		log.Printf("WARN: RabbitMQ publisher not available: %v (price updates will not be published)", err)
		publisher = nil
	}
	defer func() {
		if publisher != nil {
			publisher.Close()
		}
	}()

	priceService := service.NewPriceService(priceRepo, publisher)
	priceController := controller.NewPriceController(priceService)

	issuer := envOrDefault("KEYCLOAK_REALM_URL", "http://keycloak:8080/realms/service-app-realm")
	securityConfig := config.NewSecurityConfig(issuer)
	authMiddleware := securityConfig.AuthMiddleware()

	router := gin.Default()
	router.Use(observability.GinMiddleware("pricing-service"))
	router.Use(exception.GlobalExceptionHandler())

	router.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	prices := router.Group("/api/v1/prices")
	prices.Use(authMiddleware)
	{
		// GET /api/v1/prices is public (allowed in isPublicEndpoint by method+path).
		prices.GET("", priceController.GetAllPrices)
		// PUT /api/v1/prices/:priceType requires manager/admin.
		prices.PUT("/:priceType", securityConfig.RequireRole("manager", "admin"), priceController.UpdatePrice)
	}

	port := envOrDefault("PORT", ":8082")
	go func() {
		fmt.Printf("Pricing service starting on port %s\n", port)
		if err := router.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down pricing service gracefully...")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
