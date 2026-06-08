package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"detection-service/internal/classifier"
	"detection-service/internal/models"
	"detection-service/internal/store"
)

type PolicyHandler struct {
	policies *store.PolicyStore
}

func NewPolicyHandler(policies *store.PolicyStore) *PolicyHandler {
	return &PolicyHandler{policies: policies}
}

func (h *PolicyHandler) Create(c *gin.Context) {
	var req models.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: enabled_topics is required")
		return
	}
	if err := classifier.ValidateTopics(req.EnabledTopics); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	id := req.ID
	if id == "" {
		id = "pol_" + uuid.New().String()[:8]
	}

	now := time.Now().UTC()
	policy := models.Policy{
		ID:            id,
		EnabledTopics: req.EnabledTopics,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.policies.Create(policy); err != nil {
		handlePolicyError(c, err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

func (h *PolicyHandler) List(c *gin.Context) {
	policies, err := h.policies.List()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, models.PolicyListResponse{Policies: policies})
}

func (h *PolicyHandler) Get(c *gin.Context) {
	policy, err := h.policies.GetByID(c.Param("id"))
	if err != nil {
		handlePolicyError(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

func (h *PolicyHandler) Update(c *gin.Context) {
	var req models.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: enabled_topics is required")
		return
	}
	if err := classifier.ValidateTopics(req.EnabledTopics); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	policy, err := h.policies.Update(c.Param("id"), req.EnabledTopics)
	if err != nil {
		handlePolicyError(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

func (h *PolicyHandler) Delete(c *gin.Context) {
	if err := h.policies.Delete(c.Param("id")); err != nil {
		handlePolicyError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func handlePolicyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrPolicyNotFound):
		writeError(c, http.StatusNotFound, "policy not found")
	case errors.Is(err, store.ErrPolicyExists):
		writeError(c, http.StatusConflict, "policy already exists")
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
