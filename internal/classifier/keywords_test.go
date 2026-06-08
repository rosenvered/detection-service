package classifier

import (
	"testing"

	"detection-service/internal/models"
)

func TestKeywordMatcher_FirstMatch(t *testing.T) {
	m := NewKeywordMatcher()
	enabled := models.AllTopics

	topic, ok := m.FirstMatch("Please summarize this medical record", enabled)
	if !ok || topic != models.TopicHealthcare {
		t.Fatalf("expected healthcare, got %v ok=%v", topic, ok)
	}

	_, ok = m.FirstMatch("What is the weather today?", enabled)
	if ok {
		t.Fatal("expected no match for unrelated prompt")
	}
}

func TestKeywordMatcher_FirstMatchRespectsEnabled(t *testing.T) {
	m := NewKeywordMatcher()
	enabled := []models.Topic{models.TopicFinance}

	_, ok := m.FirstMatch("Please summarize this medical record", enabled)
	if ok {
		t.Fatal("healthcare keyword should not match when healthcare is disabled")
	}
}

func TestKeywordMatcher_AllMatches(t *testing.T) {
	m := NewKeywordMatcher()
	enabled := models.AllTopics

	matches := m.AllMatches("How much will my treatment cost and will insurance cover it?", enabled)
	if len(matches) < 2 {
		t.Fatalf("expected at least healthcare and finance, got %v", matches)
	}
}
