package classifier

import (
	"strings"

	"detection-service/internal/models"
)

var topicKeywords = map[models.Topic][]string{
	models.TopicHealthcare: {
		"medical", "medicine", "patient", "diagnosis", "treatment", "hospital",
		"doctor", "physician", "clinic", "prescription", "symptom", "surgery",
		"healthcare", "hipaa", "medical record", "health insurance", "insurance cover",
	},
	models.TopicFinance: {
		"finance", "financial", "payment", "invoice", "budget", "revenue", "profit",
		"investment", "banking", "bank", "loan", "mortgage", "salary", "compensation",
		"401k", "stock", "dividend", "tax", "accounting", "cost", "price", "expense",
	},
	models.TopicLegal: {
		"legal", "lawyer", "attorney", "lawsuit", "litigation", "contract", "agreement",
		"compliance", "regulation", "liability", "intellectual property", "patent",
		"trademark", "copyright", "court", "subpoena", "nda", "terms of service",
	},
	models.TopicHR: {
		"hr", "human resources", "hiring", "recruitment", "onboarding", "termination",
		"firing", "layoff", "performance review", "employee", "workplace", "payroll",
		"benefits", "pto", "disciplinary", "harassment", "diversity", "promotion",
	},
}

// topicPriority defines scan order for FirstMatch (fail-fast on /protect).
var topicPriority = []models.Topic{
	models.TopicHealthcare,
	models.TopicFinance,
	models.TopicLegal,
	models.TopicHR,
}

type KeywordMatcher struct{}

func NewKeywordMatcher() *KeywordMatcher {
	return &KeywordMatcher{}
}

func (m *KeywordMatcher) FirstMatch(prompt string, enabled []models.Topic) (models.Topic, bool) {
	enabledSet := enabledSet(enabled)
	lower := strings.ToLower(prompt)

	for _, topic := range topicPriority {
		if _, ok := enabledSet[topic]; !ok {
			continue
		}
		for _, keyword := range topicKeywords[topic] {
			if strings.Contains(lower, keyword) {
				return topic, true
			}
		}
	}
	return "", false
}

func (m *KeywordMatcher) AllMatches(prompt string, enabled []models.Topic) []models.Topic {
	enabledSet := enabledSet(enabled)
	lower := strings.ToLower(prompt)

	var matches []models.Topic
	for _, topic := range topicPriority {
		if _, ok := enabledSet[topic]; !ok {
			continue
		}
		for _, keyword := range topicKeywords[topic] {
			if strings.Contains(lower, keyword) {
				matches = append(matches, topic)
				break
			}
		}
	}
	return matches
}

func enabledSet(enabled []models.Topic) map[models.Topic]struct{} {
	set := make(map[models.Topic]struct{}, len(enabled))
	for _, t := range enabled {
		set[t] = struct{}{}
	}
	return set
}

func unionTopics(a, b []models.Topic) []models.Topic {
	seen := make(map[models.Topic]struct{})
	var result []models.Topic
	for _, list := range [][]models.Topic{a, b} {
		for _, t := range list {
			if _, dup := seen[t]; dup {
				continue
			}
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}

func detectionMethod(keywordHits, llmHits []models.Topic) string {
	hasKeyword := len(keywordHits) > 0
	hasLLM := len(llmHits) > 0
	switch {
	case hasKeyword && hasLLM:
		return "hybrid"
	case hasKeyword:
		return "keyword"
	default:
		return "llm"
	}
}
