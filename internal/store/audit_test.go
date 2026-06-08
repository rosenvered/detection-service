package store

import (
	"testing"
	"time"

	"detection-service/internal/models"
)

func seedAuditRecords(t *testing.T) (*AuditStore, time.Time) {
	t.Helper()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	audit := NewAuditStore(db)
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	records := []models.AuditRecord{
		{
			ID:             "aud_1",
			Timestamp:      now.Add(-3 * time.Hour),
			Endpoint:       "detect",
			Prompt:         "oldest detect",
			PolicyID:       "pol_a1b2c3",
			DetectedTopics: []models.Topic{models.TopicHealthcare},
			Method:         "llm",
			LatencyMs:      100,
		},
		{
			ID:             "aud_2",
			Timestamp:      now.Add(-2 * time.Hour),
			Endpoint:       "protect",
			Prompt:         "middle protect",
			PolicyID:       "pol_a1b2c3",
			DetectedTopics: []models.Topic{models.TopicFinance},
			Method:         "keyword",
			LatencyMs:      1,
		},
		{
			ID:             "aud_3",
			Timestamp:      now.Add(-1 * time.Hour),
			Endpoint:       "detect",
			Prompt:         "recent detect",
			PolicyID:       "pol_a1b2c3",
			DetectedTopics: []models.Topic{models.TopicLegal},
			Method:         "hybrid",
			LatencyMs:      50,
		},
		{
			ID:             "aud_4",
			Timestamp:      now,
			Endpoint:       "protect",
			Prompt:         "newest protect other policy",
			PolicyID:       "pol_other",
			DetectedTopics: []models.Topic{models.TopicHR},
			Method:         "keyword",
			LatencyMs:      2,
		},
	}

	for _, record := range records {
		if err := audit.Insert(record); err != nil {
			t.Fatalf("insert audit: %v", err)
		}
	}

	return audit, now
}

func TestAuditStore_QueryPolicyAndEndpointFilter(t *testing.T) {
	audit, _ := seedAuditRecords(t)

	result, err := audit.Query(AuditQueryFilter{
		PolicyID: "pol_a1b2c3",
		Endpoint: "protect",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("expected 1 protect record for pol_a1b2c3, got total=%d records=%d", result.Total, len(result.Records))
	}
	if result.Records[0].Prompt != "middle protect" {
		t.Fatalf("unexpected record: %+v", result.Records[0])
	}
}

func TestAuditStore_QueryLimit(t *testing.T) {
	audit, _ := seedAuditRecords(t)

	result, err := audit.Query(AuditQueryFilter{
		PolicyID: "pol_a1b2c3",
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records with limit=2, got %d", len(result.Records))
	}
	// Ordered by timestamp DESC: recent detect, then middle protect
	if result.Records[0].Prompt != "recent detect" {
		t.Fatalf("expected newest first, got %q", result.Records[0].Prompt)
	}
	if result.Records[1].Prompt != "middle protect" {
		t.Fatalf("expected second newest, got %q", result.Records[1].Prompt)
	}
}

func TestAuditStore_QueryOffset(t *testing.T) {
	audit, _ := seedAuditRecords(t)

	result, err := audit.Query(AuditQueryFilter{
		PolicyID: "pol_a1b2c3",
		Limit:    10,
		Offset:   1,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records with offset=1, got %d", len(result.Records))
	}
	if result.Records[0].Prompt != "middle protect" {
		t.Fatalf("expected offset to skip newest, got %q", result.Records[0].Prompt)
	}
	if result.Records[1].Prompt != "oldest detect" {
		t.Fatalf("expected oldest last, got %q", result.Records[1].Prompt)
	}
}

func TestAuditStore_QueryLimitAndOffset(t *testing.T) {
	audit, _ := seedAuditRecords(t)

	result, err := audit.Query(AuditQueryFilter{
		Limit:  1,
		Offset: 2,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 4 {
		t.Fatalf("expected total=4, got %d", result.Total)
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	// All records DESC: aud_4, aud_3, aud_2, aud_1 — offset 2 yields aud_2
	if result.Records[0].Prompt != "middle protect" {
		t.Fatalf("expected middle protect, got %q", result.Records[0].Prompt)
	}
}

func TestAuditStore_QueryFromFilter(t *testing.T) {
	audit, now := seedAuditRecords(t)

	from := now.Add(-90 * time.Minute) // excludes aud_1 (-3h) and aud_2 (-2h), includes aud_3 (-1h) and aud_4 (now)
	result, err := audit.Query(AuditQueryFilter{
		From:  &from,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	if result.Records[0].Prompt != "newest protect other policy" {
		t.Fatalf("expected newest first, got %q", result.Records[0].Prompt)
	}
	if result.Records[1].Prompt != "recent detect" {
		t.Fatalf("expected recent detect second, got %q", result.Records[1].Prompt)
	}
}

func TestAuditStore_QueryToFilter(t *testing.T) {
	audit, now := seedAuditRecords(t)

	to := now.Add(-90 * time.Minute) // includes aud_1 (-3h) and aud_2 (-2h), excludes aud_3 and aud_4
	result, err := audit.Query(AuditQueryFilter{
		To:    &to,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	prompts := []string{result.Records[0].Prompt, result.Records[1].Prompt}
	if prompts[0] != "middle protect" || prompts[1] != "oldest detect" {
		t.Fatalf("unexpected records: %v", prompts)
	}
}

func TestAuditStore_QueryFromAndToFilter(t *testing.T) {
	audit, now := seedAuditRecords(t)

	from := now.Add(-170 * time.Minute) // after aud_1 (09:00), includes aud_2 (10:00)
	to := now.Add(-70 * time.Minute)    // before aud_3 (11:00), includes aud_2 (10:00)
	result, err := audit.Query(AuditQueryFilter{
		From:  &from,
		To:    &to,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("expected exactly aud_2 in window, got total=%d records=%d", result.Total, len(result.Records))
	}
	if result.Records[0].Prompt != "middle protect" {
		t.Fatalf("expected middle protect, got %q", result.Records[0].Prompt)
	}
}

func TestAuditStore_QueryNoMatches(t *testing.T) {
	audit, now := seedAuditRecords(t)

	from := now.Add(1 * time.Hour)
	result, err := audit.Query(AuditQueryFilter{
		From:  &from,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if result.Total != 0 || len(result.Records) != 0 {
		t.Fatalf("expected empty result, got total=%d records=%d", result.Total, len(result.Records))
	}
}
