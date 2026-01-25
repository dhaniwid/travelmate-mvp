package handlers

import (
	"net/http"
	"travelmate/internal/domain"
	"travelmate/internal/services"

	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	Service *services.TripService
}

func NewFeedbackHandler(s *services.TripService) *FeedbackHandler {
	return &FeedbackHandler{Service: s}
}

func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	id := c.Param("id")
	var req domain.Feedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Code: "validation_error", Message: err.Error()})
		return
	}

	err := h.Service.SubmitFeedback(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == "trip not found" {
			c.JSON(http.StatusNotFound, domain.APIError{Code: "not_found", Message: "Trip not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: "internal_error", Message: err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}
