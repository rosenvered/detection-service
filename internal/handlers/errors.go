package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"detection-service/internal/classifier"
	"detection-service/internal/store"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrPolicyNotFound):
		writeError(c, http.StatusNotFound, "policy not found")
	case errors.Is(err, classifier.ErrLLMClassification):
		writeError(c, http.StatusBadGateway, "classification service unavailable")
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
