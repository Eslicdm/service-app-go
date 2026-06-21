package dto

// QuestionRequest mirrors the Spring QuestionRequest record: a single
// "question" field. Used as the body for POST /rag/ask.
type QuestionRequest struct {
	Question string `json:"question" binding:"required"`
}
