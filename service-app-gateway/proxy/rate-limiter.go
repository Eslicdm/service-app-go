package proxy

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	rateLimitPerSecond = 10  // tokens replenished per second (Spring: replenishRate=10)
	rateLimitBurst     = 20  // bucket capacity (Spring: burstCapacity=20)
)

// RateLimiter implements a per-user token bucket rate limiter. The key is the
// JWT "sub" claim (or "anonymous" if unauthenticated), mirroring the Spring
// gateway KeyResolver + RedisRateLimiter(10, 20, 1).
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a RateLimiter with the Spring gateway defaults.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rateLimitPerSecond),
		burst:    rateLimitBurst,
	}
}

// Middleware returns a Gin middleware that checks the rate limit for the
// current user (JWT sub or "anonymous"). Returns 429 Too Many Requests on
// limit exceeded.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("user_id")
		if key == "" {
			key = "anonymous"
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, ok := rl.limiters[key]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[key] = l
	}
	return l
}
