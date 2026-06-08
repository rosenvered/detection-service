package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"detection-service/internal/models"
)

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Insert(record models.AuditRecord) error {
	if record.ID == "" {
		record.ID = "aud_" + uuid.New().String()
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}

	topicsJSON, err := json.Marshal(record.DetectedTopics)
	if err != nil {
		return fmt.Errorf("marshal detected topics: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO audit_log (id, timestamp, endpoint, prompt, policy_id, detected_topics, method, latency_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.Timestamp.UTC().Format(time.RFC3339),
		record.Endpoint,
		record.Prompt,
		record.PolicyID,
		string(topicsJSON),
		record.Method,
		record.LatencyMs,
	)
	if err != nil {
		return fmt.Errorf("insert audit record: %w", err)
	}
	return nil
}
