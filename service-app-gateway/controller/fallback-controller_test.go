package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestFallbackController_Fallback_Returns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	fc := NewFallbackController()
	r.GET("/fallback/:service", fc.Fallback)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fallback/member-service", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "member-service", body["service"])
	assert.Contains(t, body["error"], "temporarily unavailable")
	assert.Equal(t, "Please try again later or contact support", body["message"])
	assert.NotNil(t, body["timestamp"])
}

func TestFallbackController_Fallback_DifferentServiceNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	fc := NewFallbackController()
	r.GET("/fallback/:service", fc.Fallback)

	services := []string{"member-service", "pricing-service", "member-request-service"}
	for _, svc := range services {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/fallback/"+svc, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var body map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, svc, body["service"])
	}
}
