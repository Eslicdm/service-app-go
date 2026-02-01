package exception

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DuplicateEmailError struct {
	Message string
}

func (e *DuplicateEmailError) Error() string {
	return e.Message
}

type EntityNotFoundError struct {
	Message string
}

func (e *EntityNotFoundError) Error() string {
	return e.Message
}

type AccessDeniedError struct {
	Message string
}

func (e *AccessDeniedError) Error() string {
	return e.Message
}

func GlobalErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			var status int
			var title string
			var detail string

			switch e := err.(type) {
			case *EntityNotFoundError:
				status = http.StatusNotFound
				title = "Member Not Found"
				detail = e.Message
			case *DuplicateEmailError:
				status = http.StatusConflict
				title = "Resource Conflict"
				detail = e.Message
			case *AccessDeniedError:
				status = http.StatusForbidden
				title = "Access Denied"
				detail = e.Message
			default:
				return
			}

			c.JSON(status, gin.H{
				"type":     "about:blank",
				"title":    title,
				"status":   status,
				"detail":   detail,
				"instance": c.Request.RequestURI,
			})
			c.Abort()
		}
	}
}
