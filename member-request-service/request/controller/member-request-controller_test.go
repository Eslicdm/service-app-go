package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"service-app-go/member-request-service/request/dto"
)

type MockRequestService struct {
	mock.Mock
}

func (m *MockRequestService) ProcessSubmission(ctx context.Context, request dto.MemberRequestDTO) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func setupRouter(svc *MockRequestService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := NewMemberRequestController(svc)
	r.POST("/api/v1/member-requests", ctrl.SubmitRequest)
	return r
}

func TestMemberRequestController_SubmitRequest_Success(t *testing.T) {
	mockSvc := new(MockRequestService)
	r := setupRouter(mockSvc)

	req := dto.MemberRequestDTO{Email: "test@example.com", ServiceType: dto.ServiceTypeFree}
	mockSvc.On("ProcessSubmission", mock.Anything, req).Return(nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/member-requests", strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusAccepted, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestMemberRequestController_SubmitRequest_InvalidEmail(t *testing.T) {
	mockSvc := new(MockRequestService)
	r := setupRouter(mockSvc)

	body := `{"email":"not-an-email","serviceType":"free"}`
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/member-requests", strings.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "ProcessSubmission")
}

func TestMemberRequestController_SubmitRequest_MissingFields(t *testing.T) {
	mockSvc := new(MockRequestService)
	r := setupRouter(mockSvc)

	body := `{"email":"test@example.com"}`
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/member-requests", strings.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "ProcessSubmission")
}
