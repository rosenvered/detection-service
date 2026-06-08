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
	keywords *KeywordMatcher
	llm      TopicClassifier
	policies *store.PolicyStore
	audit    *store.AuditStore
}

func NewService(keywords *KeywordMatcher, llm TopicClassifier, policies *store.PolicyStore, audit *store.AuditStore) *Service {
	return &Service{
		keywords: keywords,
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

	keywordHits := s.keywords.AllMatches(prompt, policy.EnabledTopics)

	llmHits, err := s.llm.ClassifyAll(ctx, prompt, policy.EnabledTopics)
	if err != nil {
		return DetectResult{}, fmt.Errorf("%w: %v", ErrLLMClassification, err)
	}

	topics := unionTopics(keywordHits, llmHits)
	result := DetectResult{
		Topics:    topics,
		Method:    detectionMethod(keywordHits, llmHits),
		LatencyMs: time.Since(start).Milliseconds(),
	}

	if err := s.writeAudit("detect", prompt, policyID, result); err != nil {
		return DetectResult{}, err
	}

	return result, nil
}

func (s *Service) Protect(ctx context.Context, prompt, policyID string) (DetectResult, error) {
	start := time.Now()

	policy, err := s.policies.GetByID(policyID)
	if err != nil {
		return DetectResult{}, err
	}

	if topic, ok := s.keywords.FirstMatch(prompt, policy.EnabledTopics); ok {
		result := DetectResult{
			Topics:    []models.Topic{topic},
			Method:    "keyword",
			LatencyMs: time.Since(start).Milliseconds(),
		}
		if err := s.writeAudit("protect", prompt, policyID, result); err != nil {
			return DetectResult{}, err
		}
		return result, nil
	}

	topics, err := s.llm.ClassifyOne(ctx, prompt, policy.EnabledTopics)
	if err != nil {
		return DetectResult{}, fmt.Errorf("%w: %v", ErrLLMClassification, err)
	}

	result := DetectResult{
		Topics:    topics,
		Method:    "llm",
		LatencyMs: time.Since(start).Milliseconds(),
	}

	if err := s.writeAudit("protect", prompt, policyID, result); err != nil {
		return DetectResult{}, err
	}

	return result, nil
}

func (s *Service) writeAudit(endpoint, prompt, policyID string, result DetectResult) error {
	topics := result.Topics
	if topics == nil {
		topics = []models.Topic{}
	}

	if err := s.audit.Insert(models.AuditRecord{
		Endpoint:       endpoint,
		Prompt:         prompt,
		PolicyID:       policyID,
		DetectedTopics: topics,
		Method:         result.Method,
		LatencyMs:      result.LatencyMs,
	}); err != nil {
		return fmt.Errorf("write audit record: %w", err)
	}
	return nil
}

var ErrLLMClassification = errors.New("llm classification failed")
