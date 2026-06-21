package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"service-app-go/recommendation-service/recommendation/dto"
	"service-app-go/recommendation-service/recommendation/service"
)

// RagController handles RAG chatbot requests, mirroring the Spring RagController:
// POST /rag/ask {question} -> plain string answer.
type RagController struct {
	ragService *service.RagService
}

// NewRagController creates a new RagController.
func NewRagController(ragService *service.RagService) *RagController {
	return &RagController{ragService: ragService}
}

// AskQuestion handles POST /rag/ask.
// @Summary Ask the RAG assistant
// @Description Ask a question about membership plans; the assistant answers from the knowledge base
// @Tags rag
// @Accept json
// @Produce plain
// @Param request body dto.QuestionRequest true "Question"
// @Success 200 {string} string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /rag/ask [post]
func (ctrl *RagController) AskQuestion(c *gin.Context) {
	var req dto.QuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer, err := ctrl.ragService.GenerateAnswer(c.Request.Context(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.String(http.StatusOK, answer)
}
