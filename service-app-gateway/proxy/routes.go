package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// Route defines a single gateway route: a path prefix proxied to a target URL.
type Route struct {
	ID          string // e.g. "member-service-api"
	PathPrefix  string // e.g. "/api/v1/members"
	TargetURL   string // e.g. "http://member-service:8081"
	ServiceName string // e.g. "member-service" (for fallback)
}

// Gateway holds the routes, circuit breakers, and rate limiter.
type Gateway struct {
	routes  []Route
	breakers *CircuitBreakers
	limiter *RateLimiter
}

// NewGateway creates a gateway with the given routes and resilience components.
func NewGateway(routes []Route, limiter *RateLimiter) *Gateway {
	ids := make([]string, len(routes))
	for i, r := range routes {
		ids[i] = r.ID
	}
	return &Gateway{
		routes:   routes,
		breakers: NewCircuitBreakers(ids),
		limiter:  limiter,
	}
}

// RegisterRoutes registers all proxy routes on the given Gin router (engine
// or group), wrapped with circuit breaker. The auth + rate-limiter middleware
// must be applied by the caller via the group.
func (g *Gateway) RegisterRoutes(r gin.IRoutes) {
	for _, route := range g.routes {
		g.registerRoute(r, route)
	}
}

func (g *Gateway) registerRoute(r gin.IRoutes, route Route) {
	target, err := url.Parse(route.TargetURL)
	if err != nil {
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve the original Host and path so the downstream service
		// sees the full /api/v1/... path (Spring StripPrefix is not used
		// for the API routes, only for springdoc which we don't proxy here).
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		// If the downstream is unreachable, return the fallback.
		w.WriteHeader(FallbackStatusCode)
		_, _ = w.Write(fallbackBody(route.ServiceName))
	}

	// Match the path prefix and proxy.
	r.Any(strings.TrimSuffix(route.PathPrefix, "/")+"/*rest", func(c *gin.Context) {
		ok, fallback := g.breakers.Execute(route.ID, route.ServiceName, func() error {
			proxy.ServeHTTP(c.Writer, c.Request)
			return nil
		})
		if !ok && fallback != nil {
			c.Data(FallbackStatusCode, "application/json", fallback)
		}
	})

	// Also handle the exact prefix (no trailing slash) for routes like
	// GET /api/v1/prices and POST /api/v1/member-requests.
	r.Any(route.PathPrefix, func(c *gin.Context) {
		ok, fallback := g.breakers.Execute(route.ID, route.ServiceName, func() error {
			proxy.ServeHTTP(c.Writer, c.Request)
			return nil
		})
		if !ok && fallback != nil {
			c.Data(FallbackStatusCode, "application/json", fallback)
		}
	})
}
