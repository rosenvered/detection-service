package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"detection-service/internal/models"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrPolicyExists   = errors.New("policy already exists")
)

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
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrPolicyExists
		}
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

	return parsePolicyRow(policy.ID, topicsJSON, createdAt, updatedAt)
}

func (s *PolicyStore) List() ([]models.Policy, error) {
	rows, err := s.db.Query(`SELECT id, enabled_topics, created_at, updated_at FROM policies ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var (
			policy     models.Policy
			topicsJSON string
			createdAt  string
			updatedAt  string
		)
		if err := rows.Scan(&policy.ID, &topicsJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan policy row: %w", err)
		}
		policy, err := parsePolicyRow(policy.ID, topicsJSON, createdAt, updatedAt)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate policies: %w", err)
	}
	if policies == nil {
		policies = []models.Policy{}
	}
	return policies, nil
}

func (s *PolicyStore) Update(id string, enabledTopics []models.Topic) (models.Policy, error) {
	if _, err := s.GetByID(id); err != nil {
		return models.Policy{}, err
	}

	topicsJSON, err := json.Marshal(enabledTopics)
	if err != nil {
		return models.Policy{}, fmt.Errorf("marshal enabled topics: %w", err)
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(
		`UPDATE policies SET enabled_topics = ?, updated_at = ? WHERE id = ?`,
		string(topicsJSON),
		now.Format(time.RFC3339),
		id,
	)
	if err != nil {
		return models.Policy{}, fmt.Errorf("update policy: %w", err)
	}

	return s.GetByID(id)
}

func (s *PolicyStore) Delete(id string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	_, err := s.db.Exec(`DELETE FROM policies WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	return nil
}

func parsePolicyRow(id, topicsJSON, createdAt, updatedAt string) (models.Policy, error) {
	var policy models.Policy
	policy.ID = id

	if err := json.Unmarshal([]byte(topicsJSON), &policy.EnabledTopics); err != nil {
		return models.Policy{}, fmt.Errorf("unmarshal enabled topics: %w", err)
	}

	var err error
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
