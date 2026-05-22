package handler

import (
	"net/http"

	"portal-core/internal/logger"
	"portal-core/internal/middleware"
	"portal-core/internal/model"
	"portal-core/internal/service"
	"portal-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	userService *service.UserService
	logger      *logrus.Entry
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger.L().WithField("component", "user_handler"),
	}
}

// -------------------- GetUserTickets --------------------

func (h *UserHandler) GetUserTickets(c *gin.Context) {
	var req model.GetUserTicketsRequest

	h.logger.Info("Get user tickets request")

	userInfo := middleware.MustGetUserInfo(c)

	userIDInterface, exists := c.Get("session_user_id")
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, utils.ErrProxy{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
			Details: "Authentication required",
		})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		h.logger.Error("Invalid user ID type in context")
		c.JSON(http.StatusInternalServerError, utils.ErrProxy{
			Code:    "INTERNAL_ERROR",
			Message: "Invalid user ID format",
			Details: "User ID must be a string",
		})
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind query parameters")
		c.JSON(http.StatusBadRequest, service.ErrValidation.
			WithMessage("Invalid query parameters").
			WithDetails(err.Error()))
		return
	}

	req.UserID = userID

	if err := h.userService.ValidateGetUserTicketsRequest(&req); err != nil {
		h.logger.WithError(err).Warn("Request validation failed")
		c.JSON(http.StatusBadRequest, service.ErrValidation.
			WithDetails(err.Error()))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"identity_id": userInfo.IdentityID,
		"login":       userInfo.Login,
		"page":        req.Page,
		"pageSize":    req.PageSize,
	}).Info("Fetching user tickets")

	tickets, total, err := h.userService.GetUserTickets(
		c.Request.Context(),
		userID,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user tickets")

		c.JSON(http.StatusInternalServerError, service.ErrDatabase.
			WithMessage("Failed to get user tickets").
			WithDetails(err.Error()))
		return
	}

	offset := (req.Page - 1) * req.PageSize
	if offset > total {
		c.JSON(http.StatusBadRequest, service.ErrDatabase.
			WithMessage("Page does not exist"))
		return
	}

	response := model.UserTicketsResponse{
		Message: "User tickets received successfully",
		Total:   total,
		Tickets: tickets,
	}

	h.logger.WithFields(logrus.Fields{
		"tickets_count": len(tickets),
	}).Info("User tickets received successfully")

	c.JSON(http.StatusOK, response)
}

// -------------------- GetUserTicket --------------------

func (h *UserHandler) GetUserTicket(c *gin.Context) {
	var req model.GetUserTicketRequest

	h.logger.Info("Get user ticket request")

	userInfo := middleware.MustGetUserInfo(c)

	userIDInterface, exists := c.Get("session_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.APIError{
			Code:    "UNAUTHORIZED",
			Message: "User ID not found",
			Details: "Authentication required",
		})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.APIError{
			Code:    "INTERNAL_ERROR",
			Message: "Invalid user ID format",
			Details: "User ID must be a string",
		})
		return
	}

	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind URI parameters")
		c.JSON(http.StatusBadRequest, model.APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid URI format",
			Details: err.Error(),
		})
		return
	}

	if err := h.userService.ValidateGetUserTicketRequest(&req); err != nil {
		h.logger.WithError(err).Warn("Request validation failed")
		c.JSON(http.StatusBadRequest, model.APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
			Details: err.Error(),
		})
		return
	}

	req.UserID = userID

	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"identity_id": userInfo.IdentityID,
		"login":       userInfo.Login,
		"ticket_id":   req.TicketID,
	}).Info("Fetching user ticket")

	ticket, err := h.userService.GetUserTicket(
		c.Request.Context(),
		userID,
		req.TicketID,
	)

	if err != nil {
		h.logger.WithError(err).Error("Failed to get user ticket")

		switch err.Error() {
		case "invalid user_id UUID", "invalid ticket_id UUID":
			c.JSON(http.StatusBadRequest, model.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid UUID format",
				Details: "ID must be a valid UUID",
			})
		case "ticket not found":
			c.JSON(http.StatusNotFound, model.APIError{
				Code:    "NOT_FOUND",
				Message: "Ticket not found",
				Details: "The specified ticket was not found or access denied",
			})
		default:
			c.JSON(http.StatusInternalServerError, model.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to get user ticket",
				Details: err.Error(),
			})
		}
		return
	}

	response := model.UserTicketResponse{
		Message: "User ticket received successfully",
		Ticket:  ticket,
	}

	c.JSON(http.StatusOK, response)
}
