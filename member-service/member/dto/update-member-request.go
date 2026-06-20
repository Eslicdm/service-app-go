package dto

import (
	"service-app-go/member-service/core/entity"
	"time"
)

type UpdateMemberRequest struct {
	Name        *string             `json:"name" binding:"omitempty,min=1"`
	Email       *string             `json:"email" binding:"omitempty,email"`
	BirthDate   *time.Time          `json:"birthDate" time_format:"2006-01-02"`
	Photo       *string             `json:"photo"`
	ServiceType *entity.ServiceType `json:"serviceType"`
}
