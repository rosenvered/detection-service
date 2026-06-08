package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"detection-service/internal/classifier"
	"detection-service/internal/models"
)

type DetectHandler struct {
	service *classifier.Service
}

func NewDetectHandler(service *classifier.Service) *DetectHandler {
	return &DetectHandler{service: service}
}

func (h *DetectHandler) Detect(c *gin.Context) {
	req, ok := bindDetectRequest(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid request: prompt and policy_id are required")
		return
	}

	result, err := h.service.Detect(c.Request.Context(), req.Prompt, req.PolicyID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	if result.Topics == nil {
		result.Topics = []models.Topic{}
	}

	c.JSON(http.StatusOK, models.DetectResponse{DetectedTopics: result.Topics})
}
