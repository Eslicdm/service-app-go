package dto

import (
	"service-app-go/member-service/core/entity"
	"time"
)

type CreateMemberRequest struct {
	Name        string             `json:"name" binding:"required"`
	Email       string             `json:"email" binding:"required,email"`
	BirthDate   time.Time          `json:"birthDate" binding:"required" time_format:"2006-01-02"`
	Photo       string             `json:"photo"`
	ServiceType entity.ServiceType `json:"serviceType" binding:"required"`
}
