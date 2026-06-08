package classifier

import (
	"fmt"
	"strings"

	"detection-service/internal/models"
)

func IsValidTopic(t models.Topic) bool {
	for _, valid := range models.AllTopics {
		if t == valid {
			return true
		}
	}
	return false
}

func ParseTopic(s string) (models.Topic, bool) {
	t := models.Topic(strings.ToLower(strings.TrimSpace(s)))
	return t, IsValidTopic(t)
}

func FilterToEnabled(detected []models.Topic, enabled []models.Topic) []models.Topic {
	enabledSet := make(map[models.Topic]struct{}, len(enabled))
	for _, t := range enabled {
		enabledSet[t] = struct{}{}
	}

	seen := make(map[models.Topic]struct{})
	var result []models.Topic
	for _, t := range detected {
		if _, ok := enabledSet[t]; !ok {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

func ValidateTopics(topics []models.Topic) error {
	for _, t := range topics {
		if !IsValidTopic(t) {
			return fmt.Errorf("invalid topic: %s", t)
		}
	}
	return nil
}
