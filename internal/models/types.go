package models

import "time"

const DefaultPolicyID = "pol_a1b2c3"

type Topic string

const (
	TopicHealthcare Topic = "healthcare"
	TopicFinance    Topic = "finance"
	TopicLegal      Topic = "legal"
	TopicHR         Topic = "hr"
)

var AllTopics = []Topic{
	TopicHealthcare,
	TopicFinance,
	TopicLegal,
	TopicHR,
}

type Policy struct {
	ID            string    `json:"id"`
	EnabledTopics []Topic   `json:"enabled_topics"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DetectRequest struct {
	Prompt   string `json:"prompt" binding:"required"`
	PolicyID string `json:"policy_id" binding:"required"`
}

type DetectResponse struct {
	DetectedTopics []Topic `json:"detected_topics"`
}

type CreatePolicyRequest struct {
	ID            string  `json:"id"`
	EnabledTopics []Topic `json:"enabled_topics" binding:"required"`
}

type UpdatePolicyRequest struct {
	EnabledTopics []Topic `json:"enabled_topics" binding:"required"`
}

type PolicyListResponse struct {
	Policies []Policy `json:"policies"`
}

type AuditRecord struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Endpoint       string    `json:"endpoint"`
	Prompt         string    `json:"prompt"`
	PolicyID       string    `json:"policy_id"`
	DetectedTopics []Topic   `json:"detected_topics"`
	Method         string    `json:"method"`
	LatencyMs      int64     `json:"latency_ms"`
}

type AuditQueryResult struct {
	Records []AuditRecord `json:"records"`
	Total   int           `json:"total"`
}
