package handlers

import (
	"net/http"
	"travelmate/internal/domain"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type CollaborationHandler struct {
	Service *services.CollaborationService
}

func NewCollaborationHandler(service *services.CollaborationService) *CollaborationHandler {
	return &CollaborationHandler{Service: service}
}

// InviteUserRequest defines the payload for inviting a user
type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=editor viewer"`
}

// InviteCollaborator handles inviting a user to a trip
func (h *CollaborationHandler) InviteCollaborator(c *gin.Context) {
	tripID := c.Param("id")
	inviterID := c.GetString("userID") // From AuthMiddleware

	var req InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: "Invalid request payload: " + err.Error()})
		return
	}

	collaborator, err := h.Service.InviteUser(c.Request.Context(), tripID, inviterID, req.Email, req.Role)
	if err != nil {
		errStr := err.Error()
		statusCode := http.StatusInternalServerError

		switch errStr {
		case "forbidden: only owners and editors can invite":
			statusCode = http.StatusForbidden
		case "user not found with email: " + req.Email:
			statusCode = http.StatusNotFound
			c.JSON(statusCode, domain.APIError{Code: "not_found", Message: "User not found"})
			return
		case "cannot invite yourself", "user is already a collaborator":
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, domain.APIError{Code: "error", Message: errStr})
		return
	}

	c.JSON(http.StatusOK, collaborator)
}

// GetCollaborators handles fetching the list of collaborators for a trip
func (h *CollaborationHandler) GetCollaborators(c *gin.Context) {
	tripID := c.Param("id")
	requestorID := c.GetString("userID")

	collaborators, err := h.Service.GetCollaborators(c.Request.Context(), tripID, requestorID)
	if err != nil {
		errStr := err.Error()
		statusCode := http.StatusInternalServerError
		if errStr == "forbidden: no access to trip" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, domain.APIError{Code: "error", Message: errStr})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collaborators": collaborators,
	})
}

// RemoveCollaborator handles removing a collaborator (or leaving a trip)
func (h *CollaborationHandler) RemoveCollaborator(c *gin.Context) {
	tripID := c.Param("id")
	targetUserID := c.Param("userId")
	requestorID := c.GetString("userID")

	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "bad_request", Message: "User ID is required"})
		return
	}

	err := h.Service.RemoveCollaborator(c.Request.Context(), tripID, requestorID, targetUserID)
	if err != nil {
		errStr := err.Error()
		statusCode := http.StatusInternalServerError
		if errStr == "forbidden: only owner can remove other collaborators" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, domain.APIError{Code: "error", Message: errStr})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Collaborator removed successfully",
	})
}

// UpdateCollaboratorRole handles updating a collaborator's role
func (h *CollaborationHandler) UpdateCollaboratorRole(c *gin.Context) {
	tripID := c.Param("id")
	targetUserID := c.Param("userId")
	requestorID := c.GetString("userID")

	var req struct {
		Role string `json:"role" binding:"required,oneof=editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: "Invalid request payload: " + err.Error()})
		return
	}

	if err := h.Service.UpdateRole(c.Request.Context(), tripID, requestorID, targetUserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}
