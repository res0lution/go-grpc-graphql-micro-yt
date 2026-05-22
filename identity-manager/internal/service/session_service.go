package service

import (
	"context"

	"identity-manager/internal/model"
	"identity-manager/internal/repository"
)

type DefaultSessionService struct {
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
	auth        AuthService
}

func NewSessionService(
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
	auth AuthService,
) *DefaultSessionService {
	return &DefaultSessionService{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		auth:        auth,
	}
}

func (s *DefaultSessionService) GetCurrentSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return s.sessionRepo.GetByID(ctx, sessionID)
}

func (s *DefaultSessionService) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if s.auth != nil {
		return s.auth.RefreshSession(ctx, sessionID)
	}
	return s.sessionRepo.GetByID(ctx, sessionID)
}

func (s *DefaultSessionService) RevokeSession(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

func (s *DefaultSessionService) ResolveIdentity(ctx context.Context, sessionID string) (*model.ResolvedIdentity, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	return &model.ResolvedIdentity{
		Identity: model.IdentityContext{
			SessionID:  session.ID,
			UserID:     session.UserID,
			IdentityID: user.IdentityID,
			Login:      user.Login,
			Groups:     user.Groups,
		},
		UserInfo: model.UserInfo{
			Sub:            user.Sub,
			IdentityID:     user.IdentityID,
			Login:          user.Login,
			Email:          user.Email,
			GivenName:      user.GivenName,
			FamilyName:     user.FamilyName,
			Name:           user.Name,
			Group:          user.Groups,
			WinAccountName: user.WinAccountName,
			EmployeeNumber: user.EmployeeNumber,
		},
	}, nil
}
