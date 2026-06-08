package classifier

import (
	"context"

	"detection-service/internal/models"
)

// TopicClassifier classifies prompts using an LLM.
type TopicClassifier interface {
	ClassifyAll(ctx context.Context, prompt string, enabled []models.Topic) ([]models.Topic, error)
	ClassifyOne(ctx context.Context, prompt string, enabled []models.Topic) ([]models.Topic, error)
}
