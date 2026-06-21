package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter()

	called := 0
	handler := func(c *gin.Context) { called++; c.Status(200) }

	// Burst is 20, so 20 requests should all pass.
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", "user-1")
		rl.Middleware()(c)
		if !c.IsAborted() {
			handler(c)
		}
	}
	assert.Equal(t, 20, called, "all 20 burst requests should be allowed")
}

func TestRateLimiter_RejectsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter()

	allowed := 0
	rejected := 0

	for i := 0; i < 25; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", "user-2")
		rl.Middleware()(c)
		if c.IsAborted() {
			rejected++
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		} else {
			allowed++
		}
	}
	assert.Equal(t, 20, allowed, "burst of 20 should be allowed")
	assert.Equal(t, 5, rejected, "5 over the burst should be rejected")
}

func TestRateLimiter_DifferentUsersIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter()

	user1Allowed := 0
	user2Allowed := 0

	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", "user-a")
		rl.Middleware()(c)
		if !c.IsAborted() {
			user1Allowed++
		}
	}
	// User 2 should still have full burst.
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("user_id", "user-b")
		rl.Middleware()(c)
		if !c.IsAborted() {
			user2Allowed++
		}
	}
	assert.Equal(t, 20, user1Allowed)
	assert.Equal(t, 20, user2Allowed, "different users should have independent limits")
}

func TestRateLimiter_AnonymousWhenNoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// No user_id set in context.
	rl.Middleware()(c)
	assert.False(t, c.IsAborted(), "anonymous should be allowed on first request")
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter()

	var mu sync.Mutex
	allowed := 0
	rejected := 0

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Set("user_id", "concurrent-user")
			rl.Middleware()(c)
			mu.Lock()
			if c.IsAborted() {
				rejected++
			} else {
				allowed++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, 30, allowed+rejected, "all goroutines should complete")
	assert.Equal(t, 20, allowed, "burst should be 20")
	assert.Equal(t, 10, rejected, "10 over burst should be rejected")
}
