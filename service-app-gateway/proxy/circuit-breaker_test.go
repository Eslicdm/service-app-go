package proxy

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

func newTestSettings(name string, timeout time.Duration) gobreaker.Settings {
	return gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}
}

func newTestBreaker(s gobreaker.Settings) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(s)
}

func TestCircuitBreakers_Execute_Success(t *testing.T) {
	cbs := NewCircuitBreakers([]string{"test-route"})

	ok, fallback := cbs.Execute("test-route", "test-service", func() error {
		return nil
	})

	assert.True(t, ok)
	assert.Nil(t, fallback)
}

func TestCircuitBreakers_Execute_Failure(t *testing.T) {
	cbs := NewCircuitBreakers([]string{"test-route"})

	ok, fallback := cbs.Execute("test-route", "test-service", func() error {
		return errors.New("downstream error")
	})

	assert.False(t, ok)
	assert.Nil(t, fallback, "non-open-state error should not return fallback body")
}

func TestCircuitBreakers_Execute_OpenState_ReturnsFallback(t *testing.T) {
	cbs := NewCircuitBreakers([]string{"test-route"})

	// Trip the breaker by exceeding ConsecutiveFailures > 5.
	for i := 0; i < 10; i++ {
		cbs.Execute("test-route", "test-service", func() error {
			return errors.New("fail")
		})
	}

	// Now the breaker should be open; Execute should return fallback.
	ok, fallback := cbs.Execute("test-route", "test-service", func() error {
		return nil
	})

	assert.False(t, ok)
	assert.NotNil(t, fallback, "open state should return fallback body")

	var body fallbackResponse
	err := json.Unmarshal(fallback, &body)
	assert.NoError(t, err)
	assert.Equal(t, "test-service", body.Service)
	assert.Contains(t, body.Error, "temporarily unavailable")
}

func TestCircuitBreakers_Execute_NoBreakerForRoute(t *testing.T) {
	cbs := NewCircuitBreakers([]string{"route-a"})

	// Route "route-b" has no breaker — should just execute.
	ok, fallback := cbs.Execute("route-b", "service-b", func() error {
		return nil
	})

	assert.True(t, ok)
	assert.Nil(t, fallback)
}

func TestFallbackBody_ContainsServiceName(t *testing.T) {
	body := fallbackBody("my-service")

	var resp fallbackResponse
	err := json.Unmarshal(body, &resp)
	assert.NoError(t, err)
	assert.Equal(t, "my-service", resp.Service)
	assert.Equal(t, "Please try again later or contact support", resp.Message)
	assert.False(t, resp.Timestamp.IsZero())
}

func TestCircuitBreakers_MultipleRoutes_Independent(t *testing.T) {
	cbs := NewCircuitBreakers([]string{"route-a", "route-b"})

	// Trip route-a only.
	for i := 0; i < 10; i++ {
		cbs.Execute("route-a", "svc-a", func() error { return errors.New("fail") })
	}

	// route-a should be open.
	okA, fallbackA := cbs.Execute("route-a", "svc-a", func() error { return nil })
	assert.False(t, okA)
	assert.NotNil(t, fallbackA)

	// route-b should still be closed.
	okB, fallbackB := cbs.Execute("route-b", "svc-b", func() error { return nil })
	assert.True(t, okB)
	assert.Nil(t, fallbackB)
}

func TestCircuitBreakers_OpenState_WithTimeout(t *testing.T) {
	// Use a breaker with a very short timeout so it transitions to half-open
	// quickly, then verify it still works after the timeout.
	settings := newTestSettings("fast-timeout", 10*time.Millisecond)
	cb := newTestBreaker(settings)

	// Trip the breaker.
	for i := 0; i < 10; i++ {
		cb.Execute(func() (interface{}, error) { return nil, errors.New("fail") })
	}

	// Wait for the timeout to expire.
	time.Sleep(20 * time.Millisecond)

	// Now the breaker should allow a half-open request.
	_, err := cb.Execute(func() (interface{}, error) { return nil, nil })
	assert.NoError(t, err)
}
