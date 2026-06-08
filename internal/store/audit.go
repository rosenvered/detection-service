package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

type AuditQueryFilter struct {
	PolicyID string
	Endpoint string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

func (s *AuditStore) Query(filter AuditQueryFilter) (models.AuditQueryResult, error) {
	query := `SELECT id, timestamp, endpoint, prompt, policy_id, detected_topics, method, latency_ms FROM audit_log`
	countQuery := `SELECT COUNT(*) FROM audit_log`

	var conditions []string
	var args []any

	if filter.PolicyID != "" {
		conditions = append(conditions, "policy_id = ?")
		args = append(args, filter.PolicyID)
	}
	if filter.Endpoint != "" {
		conditions = append(conditions, "endpoint = ?")
		args = append(args, filter.Endpoint)
	}
	if filter.From != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.From.UTC().Format(time.RFC3339))
	}
	if filter.To != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.To.UTC().Format(time.RFC3339))
	}

	if len(conditions) > 0 {
		where := " WHERE " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), filter.Limit, filter.Offset)

	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return models.AuditQueryResult{}, fmt.Errorf("count audit records: %w", err)
	}

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return models.AuditQueryResult{}, fmt.Errorf("query audit records: %w", err)
	}
	defer rows.Close()

	records, err := scanAuditRows(rows)
	if err != nil {
		return models.AuditQueryResult{}, err
	}

	return models.AuditQueryResult{Records: records, Total: total}, nil
}

func scanAuditRows(rows *sql.Rows) ([]models.AuditRecord, error) {
	var records []models.AuditRecord
	for rows.Next() {
		record, err := scanAuditRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit rows: %w", err)
	}
	if records == nil {
		records = []models.AuditRecord{}
	}
	return records, nil
}

func scanAuditRow(scanner interface {
	Scan(dest ...any) error
}) (models.AuditRecord, error) {
	var (
		record     models.AuditRecord
		timestamp  string
		topicsJSON string
	)

	if err := scanner.Scan(
		&record.ID,
		&timestamp,
		&record.Endpoint,
		&record.Prompt,
		&record.PolicyID,
		&topicsJSON,
		&record.Method,
		&record.LatencyMs,
	); err != nil {
		return models.AuditRecord{}, fmt.Errorf("scan audit row: %w", err)
	}

	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return models.AuditRecord{}, fmt.Errorf("parse timestamp: %w", err)
	}
	record.Timestamp = parsedTime

	if err := json.Unmarshal([]byte(topicsJSON), &record.DetectedTopics); err != nil {
		return models.AuditRecord{}, fmt.Errorf("unmarshal detected topics: %w", err)
	}

	return record, nil
}

