package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"service-app-go/member-service/request/service"
)

// MemberRequestController exposes pending member requests, mirroring the Spring
// MemberRequestController: GET /api/v1/members/requests.
type MemberRequestController struct {
	requestService *service.MemberRequestService
}

// NewMemberRequestController creates a new MemberRequestController.
func NewMemberRequestController(requestService *service.MemberRequestService) *MemberRequestController {
	return &MemberRequestController{requestService: requestService}
}

// GetNewMemberRequests returns the pending member requests from Redis.
// @Summary Get pending member requests
// @Description Returns member requests ingested via Kafka and stored in Redis
// @Tags member-requests
// @Produce json
// @Success 200 {array} dto.MemberRequestEvent
// @Failure 500 {object} map[string]string
// @Router /members/requests [get]
func (c *MemberRequestController) GetNewMemberRequests(ctx *gin.Context) {
	requests, err := c.requestService.GetNewMemberRequests(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, requests)
}
