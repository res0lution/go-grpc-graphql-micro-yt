package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"your_project/model"
)

type UserService struct {
	userRepo UserRepository
	jiraRepo JiraRepository
	logger   *logrus.Logger
}

// =========================
// CREATE USER FROM CLAIMS
// =========================

func (s UserService) createUserFromClaims(
	ctx context.Context,
	claims model.IDTokenClaims,
) (*model.User, bool, error) {

	identityID := claims.IdentityID
	if identityID == "" {
		identityID = claims.Sub
	}

	user := &model.User{
		ID:             uuid.New().String(),
		IdentityID:     identityID,
		Sub:            claims.Sub,
		Login:          claims.Login,
		Email:          claims.Email,
		EmployeeNumber: claims.EmployeeNumber,
		WinAccountName: claims.WinAccountName,
		GivenName:      claims.GivenName,
		FamilyName:     claims.FamilyName,
		Name:           claims.Name,
		Groups:         claims.Groups,
		IsActive:       true,
		LastLoginAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"identity_id": user.IdentityID,
		"login":       user.Login,
		"email":       user.Email,
	}).Info("New user created")

	return user, true, nil
}

// =========================
// UPDATE USER FROM CLAIMS
// =========================

func (s UserService) updateUserFromClaims(
	ctx context.Context,
	claims model.IDTokenClaims,
) (*model.User, bool, error) {

	identityID := claims.IdentityID
	if identityID == "" {
		identityID = claims.Sub
	}

	user, err := s.userRepo.GetByIdentityID(ctx, identityID)
	if err != nil {
		return nil, false, fmt.Errorf("user not found: %w", err)
	}

	user.Email = claims.Email
	user.EmployeeNumber = claims.EmployeeNumber
	user.WinAccountName = claims.WinAccountName
	user.GivenName = claims.GivenName
	user.FamilyName = claims.FamilyName
	user.Name = claims.Name
	user.Groups = claims.Groups
	user.LastLoginAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, false, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"identity_id": user.IdentityID,
		"login":       user.Login,
	}).Debug("User updated from claims")

	return user, false, nil
}

// =========================
// GET BY ID
// =========================

func (s UserService) GetByID(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// =========================
// GET BY IDENTITY ID
// =========================

func (s UserService) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
	return s.userRepo.GetByIdentityID(ctx, identityID)
}

// =========================
// GET ALL ACTIVE USERS
// =========================

func (s UserService) GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	return s.userRepo.GetAllActive(ctx, limit, offset)
}

// =========================
// GET USER TICKETS
// =========================

func (s *UserService) GetUserTickets(
	ctx context.Context,
	userID string,
	page, pageSize int,
) ([]model.TicketResponse, int, error) {

	logger := s.logger.WithFields(logrus.Fields{
		"operation": "GetUserTickets",
		"user_id":   userID,
		"page":      page,
		"pageSize":  pageSize,
	})

	logger.Debug("Getting user tickets")

	tickets, total, err := s.jiraRepo.GetTicketsByUserID(ctx, userID, page, pageSize)
	if err != nil {
		logger.WithError(err).Error("Failed to get user tickets")
		return nil, 0, fmt.Errorf("failed to get user tickets: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"tickets_found": len(tickets),
		"total":         total,
	}).Debug("User tickets received")

	return tickets, total, nil
}

// =========================
// GET SINGLE USER TICKET
// =========================

func (s *UserService) GetUserTicket(
	ctx context.Context,
	userID, ticketID string,
) (*model.TicketResponse, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "GetUserTicket",
		"user_id":   userID,
		"ticket_id": ticketID,
	})

	logger.Debug("Getting user ticket")

	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("User not found")
		return nil, fmt.Errorf("user not found: %w", err)
	}

	ticket, err := s.jiraRepo.GetTicketByUser(ctx, ticketID, userID)
	if err != nil {
		if err.Error() == "ticket not found" {
			logger.Warn("Ticket not found or access denied")
			return nil, fmt.Errorf("ticket not found")
		}

		logger.WithError(err).Error("Failed to get user ticket")
		return nil, fmt.Errorf("failed to get user ticket: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"ticket_key":    ticket.JiraKey,
		"ticket_status": ticket.Status,
	}).Debug("User ticket received")

	return ticket, nil
}

// =========================
// VALIDATION
// =========================

func (s UserService) ValidateGetUserTicketsRequest(req model.GetUserTicketsRequest) error {
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

func (s UserService) ValidateGetUserTicketRequest(req model.GetUserTicketRequest) error {
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
