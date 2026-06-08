package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"detection-service/internal/classifier"
	"detection-service/internal/handlers"
	"detection-service/internal/models"
	"detection-service/internal/store"
)

func setupTestRouter(t *testing.T, llm classifier.TopicClassifier) (*gin.Engine, *store.AuditStore) {
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
	if llm == nil {
		llm = &classifier.StubClassifier{}
	}
	service := classifier.NewService(
		classifier.NewKeywordMatcher(),
		llm,
		policyStore,
		auditStore,
	)

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	engine.POST("/detect", handlers.NewDetectHandler(service).Detect)
	engine.POST("/protect", handlers.NewProtectHandler(service).Protect)
	engine.GET("/audit", handlers.NewAuditHandler(auditStore).Query)

	return engine, auditStore
}

func postJSON(t *testing.T, engine *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)
	return rec
}

func TestProtectKeywordFastPath(t *testing.T) {
	stub := &classifier.StubClassifier{}
	engine, auditStore := setupTestRouter(t, stub)

	rec := postJSON(t, engine, "/protect", models.DetectRequest{
		Prompt:   "Please summarize this medical record",
		PolicyID: models.DefaultPolicyID,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("protect status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp models.DetectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.DetectedTopics) != 1 || resp.DetectedTopics[0] != models.TopicHealthcare {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if stub.ClassifyOneCalled {
		t.Fatal("keyword fast-path should not call LLM")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{Endpoint: "protect", Limit: 10})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 1 || auditResult.Records[0].Method != "keyword" {
		t.Fatalf("expected keyword audit record, got %+v", auditResult)
	}
}

func TestProtectLLMFallback(t *testing.T) {
	stub := &classifier.StubClassifier{
		ClassifyOneResp: []models.Topic{models.TopicFinance},
	}
	engine, auditStore := setupTestRouter(t, stub)

	rec := postJSON(t, engine, "/protect", models.DetectRequest{
		Prompt:   "Tell me about Q3 projections",
		PolicyID: models.DefaultPolicyID,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("protect status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp models.DetectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.DetectedTopics) != 1 || resp.DetectedTopics[0] != models.TopicFinance {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !stub.ClassifyOneCalled {
		t.Fatal("expected LLM fallback to call ClassifyOne")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{Endpoint: "protect", Limit: 10})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 1 || auditResult.Records[0].Method != "llm" {
		t.Fatalf("expected llm audit record, got %+v", auditResult)
	}
}

func TestProtectLLMFallbackEmptyResult(t *testing.T) {
	stub := &classifier.StubClassifier{
		ClassifyOneResp: nil,
	}
	engine, _ := setupTestRouter(t, stub)

	rec := postJSON(t, engine, "/protect", models.DetectRequest{
		Prompt:   "Tell me about Q3 projections",
		PolicyID: models.DefaultPolicyID,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("protect status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp models.DetectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.DetectedTopics) != 0 {
		t.Fatalf("expected empty topics, got %+v", resp)
	}
}

func TestProtectInvalidPolicyID(t *testing.T) {
	stub := &classifier.StubClassifier{}
	engine, auditStore := setupTestRouter(t, stub)

	rec := postJSON(t, engine, "/protect", models.DetectRequest{
		Prompt:   "Please summarize this medical record",
		PolicyID: "pol_missing",
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.ClassifyOneCalled {
		t.Fatal("LLM should not be called for missing policy")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 0 {
		t.Fatalf("expected no audit records for failed request, got %d", auditResult.Total)
	}
}

func TestDetectInvalidPolicyID(t *testing.T) {
	stub := &classifier.StubClassifier{}
	engine, auditStore := setupTestRouter(t, stub)

	rec := postJSON(t, engine, "/detect", models.DetectRequest{
		Prompt:   "How much will my treatment cost?",
		PolicyID: "pol_missing",
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.ClassifyAllCalled {
		t.Fatal("LLM should not be called for missing policy")
	}

	auditResult, err := auditStore.Query(store.AuditQueryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditResult.Total != 0 {
		t.Fatalf("expected no audit records for failed request, got %d", auditResult.Total)
	}
}

func TestProtectRejectsEmptyPrompt(t *testing.T) {
	engine, _ := setupTestRouter(t, &classifier.StubClassifier{})

	rec := postJSON(t, engine, "/protect", map[string]string{
		"prompt":    "   ",
		"policy_id": models.DefaultPolicyID,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuditInvalidEndpointFilter(t *testing.T) {
	engine, _ := setupTestRouter(t, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit?endpoint=invalid", nil)
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDetectRejectsEmptyPrompt(t *testing.T) {
	engine, _ := setupTestRouter(t, nil)

	rec := postJSON(t, engine, "/detect", map[string]string{
		"prompt":    "   ",
		"policy_id": models.DefaultPolicyID,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
