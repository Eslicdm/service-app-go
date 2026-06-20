package service

import (
	"errors"
	"fmt"

	"service-app-go/member-service/core/entity"
	"service-app-go/member-service/core/exception"
	"service-app-go/member-service/member/dto"
	"service-app-go/member-service/member/repository"

	"gorm.io/gorm"
)

type MemberService struct {
	repo *repository.MemberRepository
}

func NewMemberService(repo *repository.MemberRepository) *MemberService {
	return &MemberService{repo: repo}
}

func (s *MemberService) GetAllMembersByManagerID(managerID string) ([]entity.Member, error) {
	return s.repo.FindAllByManagerID(managerID)
}

func (s *MemberService) GetMemberByID(managerID string, memberID uint) (*entity.Member, error) {
	member, err := s.repo.FindByID(memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &exception.EntityNotFoundError{Message: fmt.Sprintf("Member not found with id: %d", memberID)}
		}
		return nil, err
	}

	if member.ManagerID != managerID {
		return nil, &exception.AccessDeniedError{Message: "Not authorized to access this member"}
	}

	return member, nil
}

func (s *MemberService) CreateMember(managerID string,
	request dto.CreateMemberRequest) (*entity.Member, error) {
	_, err := s.repo.FindByEmail(request.Email)
	if err == nil {
		return nil, &exception.DuplicateEmailError{Message: "Member creation failed due to a conflict"}
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	member := &entity.Member{
		Name:        request.Name,
		Email:       request.Email,
		BirthDate:   request.BirthDate,
		Photo:       request.Photo,
		ServiceType: request.ServiceType,
		ManagerID:   managerID,
	}

	return s.repo.Save(member)
}

func (s *MemberService) UpdateMember(memberID uint,
	request dto.UpdateMemberRequest) (*entity.Member, error) {
	member, err := s.repo.FindByID(memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &exception.EntityNotFoundError{Message: fmt.Sprintf("Member not found with id: %d", memberID)}
		}
		return nil, err
	}

	if request.Name != nil {
		member.Name = *request.Name
	}
	if request.Email != nil {
		member.Email = *request.Email
	}
	if request.BirthDate != nil {
		member.BirthDate = *request.BirthDate
	}
	if request.Photo != nil {
		member.Photo = *request.Photo
	}
	if request.ServiceType != nil {
		member.ServiceType = *request.ServiceType
	}

	return s.repo.Save(member)
}

func (s *MemberService) DeleteMember(memberID uint) error {
	exists, err := s.repo.ExistsByID(memberID)
	if err != nil {
		return err
	}
	if !exists {
		return &exception.EntityNotFoundError{Message: fmt.Sprintf("Member not found with id: %d", memberID)}
	}
	return s.repo.DeleteByID(memberID)
}
