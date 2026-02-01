package repository

import (
	"service-app-go/member-service/core/entity"

	"gorm.io/gorm"
)

type MemberRepository struct {
	db *gorm.DB
}

func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) FindAllByManagerID(managerID string) ([]entity.Member, error) {
	var members []entity.Member
	result := r.db.Where("manager_id = ?", managerID).Find(&members)
	return members, result.Error
}

func (r *MemberRepository) FindByEmail(email string) (*entity.Member, error) {
	var member entity.Member
	result := r.db.Where("email = ?", email).First(&member)
	if result.Error != nil {
		return nil, result.Error
	}
	return &member, nil
}

func (r *MemberRepository) FindByID(id uint) (*entity.Member, error) {
	var member entity.Member
	result := r.db.First(&member, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &member, nil
}

func (r *MemberRepository) Save(member *entity.Member) (*entity.Member, error) {
	result := r.db.Save(member)
	return member, result.Error
}

func (r *MemberRepository) ExistsByID(id uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Member{}).Where("id = ?", id).Count(&count)
	return count > 0, result.Error
}

func (r *MemberRepository) DeleteByID(id uint) error {
	return r.db.Delete(&entity.Member{}, id).Error
}
