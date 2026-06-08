package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"detection-service/internal/models"
)

func bindDetectRequest(c *gin.Context) (models.DetectRequest, bool) {
	var req models.DetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, false
	}

	req.Prompt = strings.TrimSpace(req.Prompt)
	req.PolicyID = strings.TrimSpace(req.PolicyID)
	if req.Prompt == "" || req.PolicyID == "" {
		return req, false
	}
	return req, true
}
