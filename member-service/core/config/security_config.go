package config

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SecurityConfig struct {
	IssuerURI string
}

func NewSecurityConfig(issuerURI string) *SecurityConfig {
	return &SecurityConfig{
		IssuerURI: issuerURI,
	}
}

func (s *SecurityConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}

func (s *SecurityConfig) isPublicEndpoint(path string) bool {
	if strings.HasPrefix(path, "/v3/api-docs") ||
		strings.HasPrefix(path, "/swagger-ui") ||
		path == "/swagger-ui.html" ||
		strings.HasPrefix(path, "/actuator/health") ||
		path == "/actuator/info" {
		return true
	}
	return false
}

func (s *SecurityConfig) ExtractRoles(claims map[string]interface{}) []string {
	realmAccess, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return []string{}
	}

	rolesRaw, ok := realmAccess["roles"]
	if !ok {
		return []string{}
	}

	rolesList, ok := rolesRaw.([]interface{})
	if !ok {
		return []string{}
	}

	var authorities []string
	for _, role := range rolesList {
		if r, ok := role.(string); ok {
			authorities = append(authorities, "ROLE_"+r)
		}
	}
	return authorities
}
