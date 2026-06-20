package repository

import (
	"errors"

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

// EmailExists reports whether a member with the given email already exists.
// Used by the Kafka request consumer to skip requests for existing members.
func (r *MemberRepository) EmailExists(email string) (bool, error) {
	_, err := r.FindByEmail(email)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
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
