package config

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig returns a Gin CORS middleware mirroring the Spring gateway
// SecurityConfig: allow http://localhost:4200 (Angular), GET/POST/PUT/DELETE/
// OPTIONS, credentials true.
func CORSConfig(allowedOrigin string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	})
}
