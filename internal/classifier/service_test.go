package classifier

import (
	"context"
	"errors"
	"testing"

	"detection-service/internal/models"
	"detection-service/internal/store"
)

func setupTestService(t *testing.T, llm TopicClassifier) (*Service, *store.AuditStore) {
	t.Helper()

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	policyStore := store.NewPolicyStore(db)
	if err := policyStore.SeedDefault(); err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	auditStore := store.NewAuditStore(db)
	service := NewService(NewKeywordMatcher(), llm, policyStore, auditStore)
	return service, auditStore
}

func TestService_Protect_KeywordFastPath(t *testing.T) {
	stub := &StubClassifier{}
	service, auditStore := setupTestService(t, stub)

	result, err := service.Protect(context.Background(), "Please summarize this medical record", models.DefaultPolicyID)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	if len(result.Topics) != 1 || result.Topics[0] != models.TopicHealthcare {
		t.Fatalf("unexpected topics: %v", result.Topics)
	}
	if result.Method != "keyword" {
		t.Fatalf("expected keyword method, got %q", result.Method)
	}
	if stub.ClassifyOneCalled {
		t.Fatal("expected LLM ClassifyOne not to be called on keyword hit")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{
		Endpoint: "protect",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 1 || auditResult.Records[0].Method != "keyword" {
		t.Fatalf("unexpected audit: %+v", auditResult)
	}
}

func TestService_Protect_LLMFallback(t *testing.T) {
	stub := &StubClassifier{
		ClassifyOneResp: []models.Topic{models.TopicFinance},
	}
	service, auditStore := setupTestService(t, stub)

	// No keyword triggers — should fall back to LLM
	result, err := service.Protect(context.Background(), "Tell me about Q3 projections", models.DefaultPolicyID)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	if len(result.Topics) != 1 || result.Topics[0] != models.TopicFinance {
		t.Fatalf("unexpected topics: %v", result.Topics)
	}
	if result.Method != "llm" {
		t.Fatalf("expected llm method, got %q", result.Method)
	}
	if !stub.ClassifyOneCalled {
		t.Fatal("expected LLM ClassifyOne to be called")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{
		Endpoint: "protect",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 1 || auditResult.Records[0].Method != "llm" {
		t.Fatalf("unexpected audit: %+v", auditResult)
	}
}

func TestService_Protect_LLMFallbackNoMatch(t *testing.T) {
	stub := &StubClassifier{
		ClassifyOneResp: nil,
	}
	service, _ := setupTestService(t, stub)

	result, err := service.Protect(context.Background(), "Tell me about Q3 projections", models.DefaultPolicyID)
	if err != nil {
		t.Fatalf("protect: %v", err)
	}
	if result.Topics != nil {
		t.Fatalf("expected nil topics, got %v", result.Topics)
	}
	if result.Method != "llm" {
		t.Fatalf("expected llm method, got %q", result.Method)
	}
}

func TestService_Protect_InvalidPolicy(t *testing.T) {
	stub := &StubClassifier{}
	service, _ := setupTestService(t, stub)

	_, err := service.Protect(context.Background(), "medical record", "pol_missing")
	if !errors.Is(err, store.ErrPolicyNotFound) {
		t.Fatalf("expected ErrPolicyNotFound, got %v", err)
	}
	if stub.ClassifyOneCalled {
		t.Fatal("LLM should not be called for missing policy")
	}
}

func TestService_Detect_InvalidPolicy(t *testing.T) {
	stub := &StubClassifier{}
	service, _ := setupTestService(t, stub)

	_, err := service.Detect(context.Background(), "medical record", "pol_missing")
	if !errors.Is(err, store.ErrPolicyNotFound) {
		t.Fatalf("expected ErrPolicyNotFound, got %v", err)
	}
	if stub.ClassifyAllCalled {
		t.Fatal("LLM should not be called for missing policy")
	}
}
