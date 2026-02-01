package entity

import (
	"time"
)

type Member struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	Name        string      `json:"name"`
	Email       string      `gorm:"unique" json:"email"`
	BirthDate   time.Time   `gorm:"column:birth_date" json:"birthDate"`
	Photo       string      `json:"photo"`
	ServiceType ServiceType `gorm:"column:service_type" json:"serviceType"`
	ManagerID   string      `gorm:"column:manager_id" json:"managerId"`
}

func (Member) TableName() string {
	return "member"
}
