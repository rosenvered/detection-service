package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"

	"detection-service/internal/models"
)

const (
	defaultAPIKey = "aim-haka-7b7018e15bac5cfad7220f562ecc94a6fb116fe3626c4456"
	modelName     = "gpt-4.1"
)

var topicPattern = regexp.MustCompile(`\b(healthcare|finance|legal|hr)\b`)

type LLMClassifier struct {
	client *openai.Client
}

func NewLLMClassifier() *LLMClassifier {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = defaultAPIKey
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.aim.security/fw/v1/proxy/openai"

	return &LLMClassifier{client: openai.NewClientWithConfig(config)}
}

func (c *LLMClassifier) ClassifyAll(ctx context.Context, prompt string, enabled []models.Topic) ([]models.Topic, error) {
	content, err := c.complete(ctx, buildClassifyAllPrompt(prompt, enabled), false)
	if err != nil {
		return nil, err
	}

	topics, err := parseTopics(content)
	if err != nil {
		content, retryErr := c.complete(ctx, buildStrictRetryPrompt(prompt, enabled), true)
		if retryErr != nil {
			return nil, fmt.Errorf("parse llm response: %w", err)
		}
		topics, err = parseTopics(content)
		if err != nil {
			return nil, fmt.Errorf("parse llm response after retry: %w", err)
		}
	}

	return FilterToEnabled(topics, enabled), nil
}

func (c *LLMClassifier) complete(ctx context.Context, userPrompt string, strict bool) (string, error) {
	systemPrompt := systemClassifyPrompt
	if strict {
		systemPrompt = systemStrictPrompt
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0,
	})
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

const systemClassifyPrompt = `You classify user prompts for sensitive business topics.
Supported topics:
- healthcare: medical care, patients, diagnoses, treatments, insurance coverage for care
- finance: money, payments, invoices, investments, banking, budgets, compensation amounts
- legal: contracts, lawsuits, compliance, regulations, liability, intellectual property
- hr: hiring, firing, performance reviews, employee relations, workplace policies

Respond with JSON only: {"topics":["topic1","topic2"]}
Use only topic names from the list above. Return an empty array if none apply.`

const systemStrictPrompt = `Return JSON only with no markdown and no explanation.
Format exactly: {"topics":["topic1"]}
Topic names must be one of: healthcare, finance, legal, hr`

func buildClassifyAllPrompt(prompt string, enabled []models.Topic) string {
	return fmt.Sprintf(
		"Classify this prompt for all applicable topics.\nOnly consider these enabled topics: %s\n\nPrompt:\n%s",
		formatTopicList(enabled),
		prompt,
	)
}

func buildStrictRetryPrompt(prompt string, enabled []models.Topic) string {
	return fmt.Sprintf(
		`Classify this prompt. Enabled topics: %s
Return JSON only: {"topics":["..."]}

Prompt:
%s`,
		formatTopicList(enabled),
		prompt,
	)
}

func formatTopicList(topics []models.Topic) string {
	names := make([]string, len(topics))
	for i, t := range topics {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}

func parseTopics(content string) ([]models.Topic, error) {
	jsonPayload := extractJSONObject(content)

	var parsed struct {
		Topics []string `json:"topics"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &parsed); err == nil {
		return stringsToTopics(parsed.Topics), nil
	}

	matches := topicPattern.FindAllString(strings.ToLower(content), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not parse topics from response: %q", content)
	}

	return stringsToTopics(matches), nil
}

func extractJSONObject(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return content
}

func stringsToTopics(values []string) []models.Topic {
	seen := make(map[models.Topic]struct{})
	var topics []models.Topic
	for _, value := range values {
		topic, ok := ParseTopic(value)
		if !ok {
			continue
		}
		if _, dup := seen[topic]; dup {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}
