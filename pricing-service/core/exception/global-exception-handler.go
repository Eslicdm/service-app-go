package exception

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorResponse defines the structure for API error responses.
type ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

// Custom Error Types for Pricing Service
type PriceNotFoundError struct {
	Message string
}

func (e *PriceNotFoundError) Error() string {
	return e.Message
}

type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string {
	return e.Message
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// GlobalExceptionHandler is a Gin middleware that handles errors globally.
func GlobalExceptionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process request

		// Check if any errors occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err // Get the last error

			var status int
			var title string
			var detail string

			switch e := err.(type) {
			case *PriceNotFoundError:
				status = http.StatusNotFound
				title = "Price Not Found"
				detail = e.Message
			case *InvalidInputError:
				status = http.StatusBadRequest
				title = "Invalid Input"
				detail = e.Message
			case *ConflictError:
				status = http.StatusConflict
				title = "Resource Conflict"
				detail = e.Message
			default:
				// Check for Gin binding errors (e.g., JSON parsing errors)
				if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
					if strings.Contains(err.Error(), "json: cannot unmarshal") || strings.Contains(err.Error(), "invalid character") {
						status = http.StatusBadRequest
						title = "Invalid Request Payload"
						detail = "The request body is malformed or contains invalid data."
					} else {
						// For other unhandled errors, return a generic 500
						status = http.StatusInternalServerError
						title = "Internal Server Error"
						detail = "An unexpected error occurred."
					}
				} else {
					// For other unhandled errors, return a generic 500
					status = http.StatusInternalServerError
					title = "Internal Server Error"
					detail = "An unexpected error occurred."
				}
			}

			c.JSON(status, ErrorResponse{
				Type:     "about:blank",
				Title:    title,
				Status:   status,
				Detail:   detail,
				Instance: c.Request.RequestURI,
			})
			c.Abort() // Stop further processing of this request
		}
	}
}
