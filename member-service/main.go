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
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	coreconfig "service-app-go/member-service/core/config"
	"service-app-go/member-service/core/entity"
	"service-app-go/member-service/core/exception"
	"service-app-go/member-service/core/observability"
	"service-app-go/member-service/member/controller"
	"service-app-go/member-service/member/repository"
	"service-app-go/member-service/member/service"
	priceController "service-app-go/member-service/pricing/controller"
	priceClient "service-app-go/member-service/pricing/client"
	priceconfig "service-app-go/member-service/pricing/config"
	priceService "service-app-go/member-service/pricing/service"
	requestController "service-app-go/member-service/request/controller"
	requestService "service-app-go/member-service/request/service"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- OpenTelemetry ---
	otelEndpoint := envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	otelShutdown, err := observability.SetupOTel(ctx, "member-service", otelEndpoint)
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

	// --- PostgreSQL (GORM) ---
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		envOrDefault("MEMBER_DB_HOST", "localhost"),
		os.Getenv("MEMBER_DB_USERNAME"),
		os.Getenv("MEMBER_DB_PASSWORD"),
		os.Getenv("MEMBER_DB_NAME"),
		envOrDefault("MEMBER_DB_PORT", "5435"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&entity.Member{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// --- Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("WARN: Redis ping failed: %v (price cache/requests will be degraded)", err)
	}

	// --- Dependency injection: member sub-domain ---
	memberRepo := repository.NewMemberRepository(db)
	memberService := service.NewMemberService(memberRepo)
	memberController := controller.NewMemberController(memberService)

	// --- Dependency injection: pricing sub-domain (cache + consumer + REST client) ---
	pricingClient := priceClient.NewPricingServiceClient(envOrDefault("PRICING_SERVICE_URL", "http://localhost:8082/api/v1"))
	priceCache := priceService.NewPriceCacheService(redisClient, pricingClient)

	var priceListener *priceService.PriceUpdateListener
	rmq, err := priceconfig.NewRabbitMQConfig()
	if err != nil {
		log.Printf("WARN: RabbitMQ not available: %v (price cache will rely on REST fallback)", err)
	} else {
		defer rmq.Close()
		priceListener = priceService.NewPriceUpdateListener(rmq, priceCache)
		go func() {
			if lerr := priceListener.StartConsuming(ctx); lerr != nil {
				log.Printf("Price update listener stopped: %v", lerr)
			}
		}()
	}
	priceCtrl := priceController.NewPriceController(priceCache)

	// --- Dependency injection: request sub-domain (Kafka consumer) ---
	requestSvc := requestService.NewMemberRequestService(redisClient)
	requestCtrl := requestController.NewMemberRequestController(requestSvc)
	requestConsumer := requestService.NewMemberRequestConsumer(redisClient, memberRepo)
	go requestConsumer.StartConsuming(ctx)
	defer func() {
		if cerr := requestConsumer.Close(); cerr != nil {
			log.Printf("Error closing kafka reader: %v", cerr)
		}
	}()

	// --- Security ---
	issuer := envOrDefault("KEYCLOAK_REALM_URL", "http://keycloak:8080/realms/service-app-realm")
	securityConfig := coreconfig.NewSecurityConfig(issuer)
	authMiddleware := securityConfig.AuthMiddleware()

	// --- HTTP server ---
	r := gin.Default()
	r.Use(observability.GinMiddleware("member-service"))
	r.Use(exception.GlobalErrorHandler())

	r.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	api := r.Group("/api/v1/members")
	api.Use(authMiddleware)
	{
		// Member CRUD: manager/admin only (Spring @PreAuthorize).
		members := api.Group("")
		members.Use(securityConfig.RequireRole("manager", "admin"))
		{
			members.GET("", memberController.GetAllMembersByManagerID)
			members.GET("/:memberId", memberController.GetMemberByID)
			members.POST("", memberController.CreateMember)
			members.PUT("/:memberId", memberController.UpdateMember)
			members.DELETE("/:memberId", memberController.DeleteMember)
		}
		// Prices (cached) and pending requests: authenticated.
		api.GET("/prices", priceCtrl.GetAllPrices)
		api.GET("/requests", requestCtrl.GetNewMemberRequests)
	}

	port := envOrDefault("PORT", ":8081")
	go func() {
		fmt.Printf("Member service starting on port %s\n", port)
		if err := r.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down member service gracefully...")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
