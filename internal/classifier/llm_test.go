package classifier

import (
	"testing"

	"detection-service/internal/models"
)

func TestParseTopics_JSON(t *testing.T) {
	topics, err := parseTopics(`{"topics":["healthcare","finance"]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 2 || topics[0] != models.TopicHealthcare || topics[1] != models.TopicFinance {
		t.Fatalf("unexpected topics: %v", topics)
	}
}

func TestParseTopics_JSONWithProse(t *testing.T) {
	topics, err := parseTopics("Here are the topics:\n```json\n{\"topics\":[\"legal\"]}\n```")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 1 || topics[0] != models.TopicLegal {
		t.Fatalf("unexpected topics: %v", topics)
	}
}

func TestParseTopics_RegexFallback(t *testing.T) {
	topics, err := parseTopics("This prompt relates to healthcare and finance.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %v", topics)
	}
}

func TestParseTopics_InvalidResponse(t *testing.T) {
	_, err := parseTopics("I am not sure about this prompt.")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFilterToEnabled(t *testing.T) {
	enabled := []models.Topic{models.TopicHealthcare, models.TopicFinance}
	detected := []models.Topic{
		models.TopicHealthcare,
		models.TopicLegal,
		models.TopicHealthcare,
		models.TopicFinance,
	}

	filtered := FilterToEnabled(detected, enabled)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 topics, got %v", filtered)
	}
	if filtered[0] != models.TopicHealthcare || filtered[1] != models.TopicFinance {
		t.Fatalf("unexpected order or values: %v", filtered)
	}
}
