package controller

import (
	"net/http"
	"strconv"

	"service-app-go/member-service/member/dto"
	"service-app-go/member-service/member/service"

	"github.com/gin-gonic/gin"
)

type MemberController struct {
	memberService *service.MemberService
}

func NewMemberController(memberService *service.MemberService) *MemberController {
	return &MemberController{memberService: memberService}
}

// GetAllMembersByManagerID GET /api/v1/members
func (ctrl *MemberController) GetAllMembersByManagerID(c *gin.Context) {
	// Assuming the AuthMiddleware extracts the subject (manager ID) from the JWT and sets it in the context
	managerID := c.GetString("manager_id")

	members, err := ctrl.memberService.GetAllMembersByManagerID(managerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, members)
}

// GetMemberByID GET /api/v1/members/:memberId
func (ctrl *MemberController) GetMemberByID(c *gin.Context) {
	managerID := c.GetString("manager_id")

	idParam := c.Param("memberId")
	memberID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	member, err := ctrl.memberService.GetMemberByID(managerID, uint(memberID))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, member)
}

// CreateMember POST /api/v1/members
func (ctrl *MemberController) CreateMember(c *gin.Context) {
	managerID := c.GetString("manager_id")

	var request dto.CreateMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := ctrl.memberService.CreateMember(managerID, request)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, member)
}

// UpdateMember PUT /api/v1/members/:memberId
func (ctrl *MemberController) UpdateMember(c *gin.Context) {
	idParam := c.Param("memberId")
	memberID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var request dto.UpdateMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := ctrl.memberService.UpdateMember(uint(memberID), request)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, member)
}

// DeleteMember DELETE /api/v1/members/:memberId
func (ctrl *MemberController) DeleteMember(c *gin.Context) {
	idParam := c.Param("memberId")
	memberID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	err = ctrl.memberService.DeleteMember(uint(memberID))
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
