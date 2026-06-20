package exception

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorResponse defines the RFC 7807-style structure for API error responses.
type ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

// GlobalExceptionHandler is a Gin middleware that handles errors globally,
// mirroring the Spring @RestControllerAdvice pattern.
func GlobalExceptionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var status int
			var title string
			var detail string

			if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
				if strings.Contains(err.Error(), "json: cannot unmarshal") || strings.Contains(err.Error(), "invalid character") {
					status = http.StatusBadRequest
					title = "Invalid Request Payload"
					detail = "The request body is malformed or contains invalid data."
				} else {
					status = http.StatusInternalServerError
					title = "Internal Server Error"
					detail = "An unexpected error occurred."
				}
			} else {
				status = http.StatusInternalServerError
				title = "Internal Server Error"
				detail = "An unexpected error occurred."
			}

			c.JSON(status, ErrorResponse{
				Type:     "about:blank",
				Title:    title,
				Status:   status,
				Detail:   detail,
				Instance: c.Request.RequestURI,
			})
			c.Abort()
		}
	}
}
