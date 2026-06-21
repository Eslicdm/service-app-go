package proxy

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// fallbackResponse is the 503 body returned when a circuit breaker is open,
// mirroring the Spring FallbackController.
type fallbackResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

// CircuitBreakers holds a gobreaker.CircuitBreaker per route ID. When a breaker
// is open, the proxy returns a 503 fallback instead of forwarding the request.
type CircuitBreakers struct {
	breakers map[string]*gobreaker.CircuitBreaker
}

// NewCircuitBreakers creates circuit breakers for the given route IDs, mirroring
// the Spring gateway Resilience4J CircuitBreaker per route.
func NewCircuitBreakers(routeIDs []string) *CircuitBreakers {
	cb := &CircuitBreakers{
		breakers: make(map[string]*gobreaker.CircuitBreaker, len(routeIDs)),
	}
	settings := gobreaker.Settings{
		Name:        "gateway",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.ConsecutiveFailures > 5 || failureRatio > 0.6
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// Could log here; kept minimal to avoid pulling a logger dep.
		},
	}
	for _, id := range routeIDs {
		s := settings
		s.Name = id
		cb.breakers[id] = gobreaker.NewCircuitBreaker(s)
	}
	return cb
}

// Execute runs the given function through the circuit breaker for the route ID.
// If the breaker is open, it returns a 503 fallback response instead.
func (cbs *CircuitBreakers) Execute(routeID, serviceName string, fn func() error) (bool, []byte) {
	cb := cbs.breakers[routeID]
	if cb == nil {
		// No breaker for this route; just execute.
		if err := fn(); err != nil {
			return false, nil
		}
		return true, nil
	}

	_, err := cb.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return false, fallbackBody(serviceName)
		}
		return false, nil
	}
	return true, nil
}

func fallbackBody(serviceName string) []byte {
	body, _ := json.Marshal(fallbackResponse{
		Error:     serviceName + " is temporarily unavailable",
		Message:   "Please try again later or contact support",
		Timestamp: time.Now(),
		Service:   serviceName,
	})
	return body
}

// FallbackStatusCode is the HTTP status returned when a circuit is open.
const FallbackStatusCode = http.StatusServiceUnavailable
