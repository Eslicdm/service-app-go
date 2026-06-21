package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	weavientmodels "github.com/weaviate/weaviate/entities/models"

	"service-app-go/recommendation-service/recommendation/config"
)

const (
	defaultClassName      = "PriceTier"
	embeddingBatchSize    = 10
	similaritySearchTopK  = 50
	initProbeQuery        = "Garden Pass"
	initProbeTopK         = 1
)

// RagService implements the RAG pipeline: load knowledge base into Weaviate
// at startup (if empty), then answer questions by retrieving relevant context
// and calling Google Gemini. Mirrors the Spring RagService.
type RagService struct {
	weaviateClient *weaviate.Client
	geminiClient   *genai.Client
	markdownReader *config.MarkdownReader
	className      string
	chatModel      string
	embeddingModel string
	promptTemplate string
}

// NewRagService creates a new RagService. The Gemini client and Weaviate client
// must already be connected. Init() should be called after construction.
func NewRagService(weaviateClient *weaviate.Client, geminiClient *genai.Client, reader *config.MarkdownReader) (*RagService, error) {
	promptTemplate, err := reader.LoadPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt template: %w", err)
	}

	return &RagService{
		weaviateClient: weaviateClient,
		geminiClient:   geminiClient,
		markdownReader: reader,
		className:      envOrDefault("WEAVIATE_CLASS", defaultClassName),
		chatModel:      envOrDefault("GOOGLE_CHAT_MODEL", "gemini-2.0-flash-live"),
		embeddingModel: envOrDefault("GOOGLE_EMBEDDING_MODEL", "text-embedding-004"),
		promptTemplate: promptTemplate,
	}, nil
}

// Init loads the knowledge base into Weaviate if it is empty. Mirrors the
// Spring RagService @PostConstruct init().
func (s *RagService) Init(ctx context.Context) error {
	if err := s.ensureSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}

	existing, err := s.searchDocs(ctx, initProbeQuery, initProbeTopK)
	if err != nil {
		log.Printf("VectorStore check failed (will attempt load): %v", err)
	}
	if len(existing) > 0 {
		log.Println("VectorStore already contains documents. Skipping data loading.")
		return nil
	}

	docs, err := s.markdownReader.LoadMarkdown()
	if err != nil {
		return fmt.Errorf("failed to load markdown: %w", err)
	}
	if len(docs) == 0 {
		log.Println("No documents found to load.")
		return nil
	}

	if err := s.loadDocuments(ctx, docs); err != nil {
		return fmt.Errorf("failed to load documents into VectorStore: %w", err)
	}
	log.Printf("Loaded %d documents into VectorStore", len(docs))
	return nil
}

// GenerateAnswer retrieves relevant context from Weaviate and calls Gemini to
// generate an answer. Mirrors the Spring RagService.generateAnswer().
func (s *RagService) GenerateAnswer(ctx context.Context, query string) (string, error) {
	docs, err := s.searchDocs(ctx, query, similaritySearchTopK)
	if err != nil {
		log.Printf(" similarity search error: %v", err)
	}

	context := strings.Join(docs, "\n\n")
	log.Printf("Retrieved %d context chunks", len(docs))

	prompt := strings.ReplaceAll(s.promptTemplate, "{context}", context)
	prompt = strings.ReplaceAll(prompt, "{question}", query)

	model := s.geminiClient.GenerativeModel(s.chatModel)
	model.SetTemperature(0.5)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "I'm sorry, I don't have information about that.", nil
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			sb.WriteString(string(text))
		}
	}
	return sb.String(), nil
}

// ensureSchema creates the Weaviate collection if it does not exist.
func (s *RagService) ensureSchema(ctx context.Context) error {
	exists, err := s.weaviateClient.Schema().ClassGetter().WithClassName(s.className).Do(ctx)
	if err == nil && exists != nil {
		return nil
	}

	class := &weavientmodels.Class{
		Class:      s.className,
		Vectorizer: "none",
		Properties: []*weavientmodels.Property{
			{Name: "text", DataType: []string{"text"}},
		},
	}
	if err := s.weaviateClient.Schema().ClassCreator().WithClass(class).Do(ctx); err != nil {
		return fmt.Errorf("failed to create class %s: %w", s.className, err)
	}
	log.Printf("Created Weaviate class %s", s.className)
	return nil
}

// loadDocuments embeds each document via Gemini and stores it in Weaviate.
func (s *RagService) loadDocuments(ctx context.Context, docs []config.Document) error {
	for _, doc := range docs {
		vector, err := s.embed(ctx, doc.Text)
		if err != nil {
			return fmt.Errorf("failed to embed document: %w", err)
		}

		_, err = s.weaviateClient.Data().Creator().
			WithClassName(s.className).
			WithID(uuid.NewString()).
			WithProperties(map[string]interface{}{"text": doc.Text}).
			WithVector(vector).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to store document in Weaviate: %w", err)
		}
	}
	return nil
}

// searchDocs performs a nearVector similarity search and returns the text
// fields of the matched objects.
func (s *RagService) searchDocs(ctx context.Context, query string, topK int) ([]string, error) {
	queryVector, err := s.embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	result, err := s.weaviateClient.GraphQL().Get().
		WithClassName(s.className).
		WithFields(graphql.Field{Name: "text"}).
		WithNearVector((&graphql.NearVectorArgumentBuilder{}).WithVector(queryVector)).
		WithLimit(topK).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("weaviate search failed: %w", err)
	}

	return parseGraphQLTexts(result, s.className)
}

// embed calls the Gemini embedding model and returns a float32 vector.
func (s *RagService) embed(ctx context.Context, text string) ([]float32, error) {
	em := s.geminiClient.EmbeddingModel(s.embeddingModel)
	resp, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Embedding == nil {
		return nil, fmt.Errorf("empty embedding response")
	}
	return resp.Embedding.Values, nil
}

// parseGraphQLTexts extracts the "text" field from a Weaviate GraphQL response.
// The response Data is a map[string]models.JSONObject; we navigate to Get.<className>[].text.
func parseGraphQLTexts(result *weavientmodels.GraphQLResponse, className string) ([]string, error) {
	if result == nil || result.Data == nil {
		return nil, nil
	}
	getData, ok := result.Data["Get"]
	if !ok {
		return nil, nil
	}
	// JSONObject is map[string]interface{}; navigate to the class array.
	classMap, ok := getData.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	objects, ok := classMap[className].([]interface{})
	if !ok {
		return nil, nil
	}

	texts := make([]string, 0, len(objects))
	for _, obj := range objects {
		m, ok := obj.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := m["text"].(string); ok {
			texts = append(texts, text)
		}
	}
	return texts, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
