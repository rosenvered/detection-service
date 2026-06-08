package classifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"detection-service/internal/models"
	"detection-service/internal/store"
)

type Service struct {
	llm      *LLMClassifier
	policies *store.PolicyStore
	audit    *store.AuditStore
}

func NewService(llm *LLMClassifier, policies *store.PolicyStore, audit *store.AuditStore) *Service {
	return &Service{
		llm:      llm,
		policies: policies,
		audit:    audit,
	}
}

type DetectResult struct {
	Topics    []models.Topic
	Method    string
	LatencyMs int64
}

func (s *Service) Detect(ctx context.Context, prompt, policyID string) (DetectResult, error) {
	start := time.Now()

	policy, err := s.policies.GetByID(policyID)
	if err != nil {
		return DetectResult{}, err
	}

	topics, err := s.llm.ClassifyAll(ctx, prompt, policy.EnabledTopics)
	if err != nil {
		return DetectResult{}, fmt.Errorf("%w: %v", ErrLLMClassification, err)
	}

	result := DetectResult{
		Topics:    topics,
		Method:    "llm",
		LatencyMs: time.Since(start).Milliseconds(),
	}

	auditErr := s.audit.Insert(models.AuditRecord{
		Endpoint:       "detect",
		Prompt:         prompt,
		PolicyID:       policyID,
		DetectedTopics: result.Topics,
		Method:         result.Method,
		LatencyMs:      result.LatencyMs,
	})
	if auditErr != nil {
		return DetectResult{}, fmt.Errorf("write audit record: %w", auditErr)
	}

	return result, nil
}

var ErrLLMClassification = errors.New("llm classification failed")
