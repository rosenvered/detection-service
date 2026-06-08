package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"detection-service/internal/models"
)

var ErrPolicyNotFound = errors.New("policy not found")

type PolicyStore struct {
	db *sql.DB
}

func NewPolicyStore(db *sql.DB) *PolicyStore {
	return &PolicyStore{db: db}
}

func (s *PolicyStore) Create(policy models.Policy) error {
	topicsJSON, err := json.Marshal(policy.EnabledTopics)
	if err != nil {
		return fmt.Errorf("marshal enabled topics: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO policies (id, enabled_topics, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		policy.ID,
		string(topicsJSON),
		policy.CreatedAt.UTC().Format(time.RFC3339),
		policy.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert policy: %w", err)
	}
	return nil
}

func (s *PolicyStore) GetByID(id string) (models.Policy, error) {
	var (
		topicsJSON string
		createdAt  string
		updatedAt  string
		policy     models.Policy
	)

	err := s.db.QueryRow(
		`SELECT id, enabled_topics, created_at, updated_at FROM policies WHERE id = ?`,
		id,
	).Scan(&policy.ID, &topicsJSON, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Policy{}, ErrPolicyNotFound
	}
	if err != nil {
		return models.Policy{}, fmt.Errorf("query policy: %w", err)
	}

	if err := json.Unmarshal([]byte(topicsJSON), &policy.EnabledTopics); err != nil {
		return models.Policy{}, fmt.Errorf("unmarshal enabled topics: %w", err)
	}

	policy.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.Policy{}, fmt.Errorf("parse created_at: %w", err)
	}
	policy.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.Policy{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return policy, nil
}

func (s *PolicyStore) SeedDefault() error {
	_, err := s.GetByID(models.DefaultPolicyID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrPolicyNotFound) {
		return err
	}

	now := time.Now().UTC()
	return s.Create(models.Policy{
		ID:            models.DefaultPolicyID,
		EnabledTopics: models.AllTopics,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
}
