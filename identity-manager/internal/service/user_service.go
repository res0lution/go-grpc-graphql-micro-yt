package service

import (
	"context"

	"identity-manager/internal/model"
	"identity-manager/internal/repository"
)

type DefaultUserService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewUserService(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) *DefaultUserService {
	return &DefaultUserService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *DefaultUserService) GetCurrentUser(ctx context.Context, sessionID string) (*model.User, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.userRepo.GetByID(ctx, session.UserID)
}

func (s *DefaultUserService) GetByID(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *DefaultUserService) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
	return s.userRepo.GetByIdentityID(ctx, identityID)
}

func (s *DefaultUserService) GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	return s.userRepo.GetAllActive(ctx, limit, offset)
}
