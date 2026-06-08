package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"detection-service/internal/classifier"
	"detection-service/internal/models"
)

type ProtectHandler struct {
	service *classifier.Service
}

func NewProtectHandler(service *classifier.Service) *ProtectHandler {
	return &ProtectHandler{service: service}
}

func (h *ProtectHandler) Protect(c *gin.Context) {
	req, ok := bindDetectRequest(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid request: prompt and policy_id are required")
		return
	}

	result, err := h.service.Protect(c.Request.Context(), req.Prompt, req.PolicyID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	if result.Topics == nil {
		result.Topics = []models.Topic{}
	}

	c.JSON(http.StatusOK, models.DetectResponse{DetectedTopics: result.Topics})
}
