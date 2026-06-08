package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"detection-service/internal/store"
)

const (
	defaultAuditLimit = 50
	maxAuditLimit     = 100
)

type AuditHandler struct {
	audit *store.AuditStore
}

func NewAuditHandler(audit *store.AuditStore) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) Query(c *gin.Context) {
	filter, err := parseAuditQuery(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.audit.Query(filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, result)
}

func parseAuditQuery(c *gin.Context) (store.AuditQueryFilter, error) {
	filter := store.AuditQueryFilter{
		PolicyID: c.Query("policy_id"),
		Endpoint: c.Query("endpoint"),
		Limit:    defaultAuditLimit,
		Offset:   0,
	}

	if filter.Endpoint != "" && filter.Endpoint != "detect" && filter.Endpoint != "protect" {
		return filter, errBadRequest("endpoint must be detect or protect")
	}

	if from := c.Query("from"); from != "" {
		parsed, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return filter, errBadRequest("from must be RFC3339 timestamp")
		}
		filter.From = &parsed
	}

	if to := c.Query("to"); to != "" {
		parsed, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return filter, errBadRequest("to must be RFC3339 timestamp")
		}
		filter.To = &parsed
	}

	if limit := c.Query("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 0 {
			return filter, errBadRequest("limit must be a non-negative integer")
		}
		if parsed > maxAuditLimit {
			parsed = maxAuditLimit
		}
		filter.Limit = parsed
	}

	if offset := c.Query("offset"); offset != "" {
		parsed, err := strconv.Atoi(offset)
		if err != nil || parsed < 0 {
			return filter, errBadRequest("offset must be a non-negative integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}

type badRequestError struct {
	message string
}

func errBadRequest(message string) error {
	return badRequestError{message: message}
}

func (e badRequestError) Error() string {
	return e.message
}
