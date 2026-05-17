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
	"service-app-go/pricing-service/pricing/controller"
	"service-app-go/pricing-service/pricing/messaging"
	"service-app-go/pricing-service/pricing/repository"
	"service-app-go/pricing-service/pricing/service"
)

const (
	dbUsernameEnv = "PRICING_DB_USERNAME"
	dbPasswordEnv = "PRICING_DB_PASSWORD"
	dbNameEnv     = "PRICING_DB_NAME"
	mongoHost     = "localhost" // This should match the service name in docker-compose
	mongoPort     = "27017"
	port          = ":8082" // Port for the Gin server, aligned with Java service
	errEnvNotSet  = "Environment variable %s not set"
)

func main() {
	err := godotenv.Load()

	// --- Set up graceful shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Environment Variable Loading ---
	dbUsername := os.Getenv(dbUsernameEnv)
	if dbUsername == "" {
		log.Fatalf(errEnvNotSet, dbUsernameEnv)
	}
	dbPassword := os.Getenv(dbPasswordEnv)
	if dbPassword == "" {
		log.Fatalf(errEnvNotSet, dbPasswordEnv)
	}
	dbName := os.Getenv(dbNameEnv)
	if dbName == "" {
		log.Fatalf(errEnvNotSet, dbNameEnv)
	}

	// --- MongoDB Connection ---
	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=admin", dbUsername, dbPassword, mongoHost, mongoPort, dbName)
	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	client, err := mongo.Connect(mongoCtx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(mongoCtx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
		fmt.Println("Disconnected from MongoDB")
	}()

	err = client.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	fmt.Println("Successfully connected to MongoDB!")

	// --- Dependency Injection ---
	priceRepo := repository.NewMongoPriceRepository(client, dbName)
	priceService := service.NewPriceService(priceRepo)

	// --- RabbitMQ Consumer Setup ---
	rmqConsumer, err := messaging.NewRabbitMQConsumer(priceService)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ consumer: %v", err)
	}
	defer rmqConsumer.Close() // Ensure consumer connection is closed on exit

	// Start consuming messages in a goroutine
	go rmqConsumer.StartConsuming(ctx)

	// --- Gin Web Server Setup ---
	router := gin.Default()

	// Add global exception handler middleware
	router.Use(exception.GlobalExceptionHandler())

	// Initialize security config and JWT middleware
	securityConfig := config.NewSecurityConfig("http://keycloak:8080/realms/service-app-realm")
	authMiddleware := securityConfig.AuthMiddleware()

	priceController := controller.NewPriceController(priceService)

	pricesGroup := router.Group("/api/v1/prices")
	pricesGroup.Use(authMiddleware) // Apply AuthMiddleware to all price routes
	{
		pricesGroup.POST("", priceController.CreatePrice)
		pricesGroup.GET("/:id", priceController.GetPriceByID)
		pricesGroup.GET("", priceController.GetAllPrices)
		pricesGroup.PUT("/:id", priceController.UpdatePrice)
		pricesGroup.DELETE("/:id", priceController.DeletePrice)
	}

	// --- Start Gin server in a goroutine ---
	go func() {
		fmt.Printf("Pricing service starting on port %s\n", port)
		if err := router.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	// --- Wait for interrupt signal ---
	<-ctx.Done()
	log.Println("Shutting down pricing service gracefully...")
}
