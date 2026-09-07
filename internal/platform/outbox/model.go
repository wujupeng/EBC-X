package outbox

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
	StatusDLQ       Status = "dlq"
)

type Event struct {
	EventID       string         `json:"event_id"`
	AggregateType string         `json:"aggregate_type"`
	AggregateID   string         `json:"aggregate_id"`
	EventType     string         `json:"event_type"`
	Payload       map[string]any `json:"payload"`
	TenantID      string         `json:"tenant_id"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	CausationID   string         `json:"causation_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	RetryCount    int            `json:"retry_count"`
	Status        Status         `json:"status"`
	NextRetryAt   *time.Time     `json:"next_retry_at,omitempty"`
}

type Mode string

const (
	ModeNormal   Mode = "normal"
	ModeDegraded Mode = "degraded"
	ModeRecovery Mode = "recovery"
	ModeRebuild  Mode = "rebuild"
)

type ProjectionResult struct {
	Mode            Mode
	ProjectionLagMs int64
	EventLag        int64
	ConsumerLag     int64
	DLQCount        int
	FailedCount     int
}
