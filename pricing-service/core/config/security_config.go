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

// SecurityConfig holds security-related configurations, including the JWKS URL and a cache for public keys.
type SecurityConfig struct {
	IssuerURI string
	jwksURL   string
	keyCache  map[string]*rsa.PublicKey
	mutex     sync.RWMutex
}

// NewSecurityConfig creates and returns a new SecurityConfig.
func NewSecurityConfig(issuerURI string) *SecurityConfig {
	return &SecurityConfig{
		IssuerURI: issuerURI,
		jwksURL:   strings.TrimRight(issuerURI, "/") + "/protocol/openid-connect/certs",
		keyCache:  make(map[string]*rsa.PublicKey),
	}
}

// AuthMiddleware is a Gin middleware for JWT authentication using JWKS.
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
			// Validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return s.getPublicKey(token)
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid token: %v", err)})
			return
		}

		// Check issuer claim
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
		c.Set("manager_id", subClaim(claims))
		c.Set("roles", s.ExtractRoles(claims))
		c.Next()
	}
}

// RequireRole returns a Gin middleware that allows access only when the
// authenticated JWT carries one of the allowed Keycloak realm roles. It
// replaces Spring's @PreAuthorize("hasRole('manager') or hasRole('admin')").
func (s *SecurityConfig) RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet["ROLE_"+a] = struct{}{}
	}
	return func(c *gin.Context) {
		raw, ok := c.Get("roles")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: no roles"})
			return
		}
		roles, _ := raw.([]string)
		for _, r := range roles {
			if _, ok := allowedSet[r]; ok {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient role"})
	}
}

// subClaim extracts the JWT "sub" claim as a string (the Keycloak user id,
// used as managerId in the Spring reference).
func subClaim(claims map[string]interface{}) string {
	if v, ok := claims["sub"].(string); ok {
		return v
	}
	return ""
}

// isPublicEndpoint checks if the given request is a public endpoint that does
// not require authentication. Mirrors the Spring SecurityConfig: GET /api/v1/prices
// is public, swagger/actuator health are public.
func (s *SecurityConfig) isPublicEndpoint(method, path string) bool {
	publicPrefixes := []string{
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

	// GET /api/v1/prices is public (Spring: permitAll for GET prices).
	if method == http.MethodGet && path == "/api/v1/prices" {
		return true
	}

	return false
}

// ExtractRoles extracts roles from JWT claims.
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

// getPublicKey retrieves the RSA public key from cache or by refreshing JWKS.
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

	// Key not in cache, try to refresh
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

// refreshKeys fetches the JWKS from the configured URL and updates the key cache.
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

	// Clear existing keys before adding new ones to handle key rotation
	s.keyCache = make(map[string]*rsa.PublicKey)

	for _, k := range jwks.Keys {
		if k.N == "" || k.E == "" {
			continue
		}

		// Decode Modulus (N)
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)

		// Decode Exponent (E)
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
