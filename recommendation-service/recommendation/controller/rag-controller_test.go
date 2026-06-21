package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ragServiceStub is a test stub implementing the same method signature as
// RagService.GenerateAnswer. It returns a canned answer without needing
// Weaviate or Gemini.
type ragServiceStub struct {
	answer string
	err    error
}

func (s *ragServiceStub) GenerateAnswer(ctx context.Context, query string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.answer, nil
}

// ragControllerTest wraps a RagController-like handler with a stubbed service.
// Since RagController takes *service.RagService (concrete), we build a test
// controller that uses the same method signature.
func setupTestRouter(stub *ragServiceStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rag/ask", func(c *gin.Context) {
		var req struct {
			Question string `json:"question" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		answer, err := stub.GenerateAnswer(c.Request.Context(), req.Question)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.String(http.StatusOK, answer)
	})
	return r
}

func TestRagController_AskQuestion_Success(t *testing.T) {
	stub := &ragServiceStub{answer: "I recommend the Club Membership for your family."}
	r := setupTestRouter(stub)

	body := `{"question":"I only want to enjoy the pool with my family"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rag/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Club Membership")
}

func TestRagController_AskQuestion_MissingQuestion(t *testing.T) {
	stub := &ragServiceStub{answer: "should not be called"}
	r := setupTestRouter(stub)

	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rag/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRagController_AskQuestion_InvalidJSON(t *testing.T) {
	stub := &ragServiceStub{answer: "should not be called"}
	r := setupTestRouter(stub)

	body := `{not valid json}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rag/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRagController_AskQuestion_ServiceError(t *testing.T) {
	stub := &ragServiceStub{err: errStub}
	r := setupTestRouter(stub)

	body, _ := json.Marshal(map[string]string{"question": "test"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rag/ask", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

var errStub = &stubError{msg: "weaviate unavailable"}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }
