package config

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SecurityConfig holds JWKS configuration for JWT validation, mirroring the
// Spring gateway SecurityConfig (OAuth2 resource server, JWK set URI).
type SecurityConfig struct {
	IssuerURI string
	jwksURL   string
	keyCache  map[string]*rsa.PublicKey
	mutex     sync.RWMutex
}

func NewSecurityConfig(issuerURI string) *SecurityConfig {
	return &SecurityConfig{
		IssuerURI: issuerURI,
		jwksURL:   strings.TrimRight(issuerURI, "/") + "/protocol/openid-connect/certs",
		keyCache:  make(map[string]*rsa.PublicKey),
	}
}

// AuthMiddleware validates JWT tokens via JWKS. Public endpoints (Spring
// permitAll): /fallback/**, actuator health/info, swagger, POST
// /api/v1/member-requests/**, GET /api/v1/prices, OPTIONS /**.
func (s *SecurityConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.isPublicEndpoint(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required or malformed"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.getPublicKey(token)
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid token: %v", err)})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims format"})
			return
		}
		if claims["iss"] != s.IssuerURI {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token issuer"})
			return
		}

		c.Set("claims", claims)
		c.Set("user_id", subClaim(claims))
		c.Next()
	}
}

// isPublicEndpoint mirrors the Spring gateway SecurityConfig public paths.
func (s *SecurityConfig) isPublicEndpoint(method, path string) bool {
	// OPTIONS /** is public (CORS preflight).
	if method == http.MethodOptions {
		return true
	}

	publicPrefixes := []string{
		"/fallback/",
		"/v3/api-docs",
		"/swagger-ui",
		"/actuator/health",
	}
	publicExactMatches := []string{
		"/swagger-ui.html",
		"/actuator/info",
	}
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, match := range publicExactMatches {
		if path == match {
			return true
		}
	}

	// POST /api/v1/member-requests/** is public (Spring permitAll).
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/member-requests") {
		return true
	}
	// GET /api/v1/prices is public (Spring permitAll).
	if method == http.MethodGet && path == "/api/v1/prices" {
		return true
	}

	return false
}

func subClaim(claims jwt.MapClaims) string {
	if v, ok := claims["sub"].(string); ok {
		return v
	}
	return ""
}

func (s *SecurityConfig) getPublicKey(token *jwt.Token) (interface{}, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("kid header not found")
	}

	s.mutex.RLock()
	key, exists := s.keyCache[kid]
	s.mutex.RUnlock()
	if exists {
		return key, nil
	}

	if err := s.refreshKeys(); err != nil {
		return nil, err
	}

	s.mutex.RLock()
	key, exists = s.keyCache[kid]
	s.mutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("key not found for kid: %s", kid)
	}
	return key, nil
}

func (s *SecurityConfig) refreshKeys() error {
	resp, err := http.Get(s.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response: %v", err)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %v", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.keyCache = make(map[string]*rsa.PublicKey)

	for _, k := range jwks.Keys {
		if k.N == "" || k.E == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		var e int
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		s.keyCache[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}
	return nil
}
