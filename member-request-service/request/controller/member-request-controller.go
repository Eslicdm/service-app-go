package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"service-app-go/member-request-service/request/dto"
	"service-app-go/member-request-service/request/service"
)

// MemberRequestController handles prospect member request submissions,
// mirroring the Spring MemberRequestController: POST /api/v1/member-requests -> 202.
type MemberRequestController struct {
	requestService *service.MemberRequestService
}

// NewMemberRequestController creates a new MemberRequestController.
func NewMemberRequestController(requestService *service.MemberRequestService) *MemberRequestController {
	return &MemberRequestController{requestService: requestService}
}

// SubmitRequest handles POST /api/v1/member-requests.
// @Summary Submit a member request
// @Description Receive a prospect's membership request, dedup via Redis, produce to Kafka
// @Tags member-requests
// @Accept json
// @Produce json
// @Param request body dto.MemberRequestDTO true "Member request"
// @Success 202 "Accepted"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /member-requests [post]
func (ctrl *MemberRequestController) SubmitRequest(c *gin.Context) {
	var request dto.MemberRequestDTO
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.requestService.ProcessSubmission(c.Request.Context(), request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusAccepted)
}
