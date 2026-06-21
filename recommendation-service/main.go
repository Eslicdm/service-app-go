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
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"google.golang.org/api/option"

	"service-app-go/recommendation-service/recommendation/config"
	"service-app-go/recommendation-service/recommendation/controller"
	"service-app-go/recommendation-service/recommendation/service"
)

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Weaviate ---
	weaviateHost := envOrDefault("WEAVIATE_HOST", "localhost:8091")
	weaviateScheme := envOrDefault("WEAVIATE_SCHEME", "http")
	weaviateClient, err := weaviate.NewClient(weaviate.Config{
		Host:   weaviateHost,
		Scheme: weaviateScheme,
	})
	if err != nil {
		log.Fatalf("Failed to create Weaviate client: %v", err)
	}

	// --- Gemini ---
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY not set")
	}
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer geminiClient.Close()

	// --- RAG service ---
	markdownReader := config.NewMarkdownReader()
	ragService, err := service.NewRagService(weaviateClient, geminiClient, markdownReader)
	if err != nil {
		log.Fatalf("Failed to create RAG service: %v", err)
	}

	// Load knowledge base into Weaviate if empty (Spring @PostConstruct).
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	defer initCancel()
	if err := ragService.Init(initCtx); err != nil {
		log.Printf("WARN: RAG init failed: %v (will continue, queries may lack context)", err)
	}

	ragController := controller.NewRagController(ragService)

	// --- HTTP server ---
	r := gin.Default()

	r.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	r.POST("/rag/ask", ragController.AskQuestion)

	port := envOrDefault("PORT", ":8085")
	go func() {
		fmt.Printf("Recommendation service starting on port %s\n", port)
		if err := r.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down recommendation service gracefully...")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
