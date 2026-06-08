package classifier

import (
	"context"

	"detection-service/internal/models"
)

// StubClassifier is a test double for TopicClassifier.
type StubClassifier struct {
	ClassifyAllResp []models.Topic
	ClassifyAllErr  error
	ClassifyOneResp []models.Topic
	ClassifyOneErr  error

	ClassifyAllCalled bool
	ClassifyOneCalled bool
}

func (s *StubClassifier) ClassifyAll(ctx context.Context, prompt string, enabled []models.Topic) ([]models.Topic, error) {
	s.ClassifyAllCalled = true
	if s.ClassifyAllErr != nil {
		return nil, s.ClassifyAllErr
	}
	return s.ClassifyAllResp, nil
}

func (s *StubClassifier) ClassifyOne(ctx context.Context, prompt string, enabled []models.Topic) ([]models.Topic, error) {
	s.ClassifyOneCalled = true
	if s.ClassifyOneErr != nil {
		return nil, s.ClassifyOneErr
	}
	return s.ClassifyOneResp, nil
}
