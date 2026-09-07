package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	ErrEventBusUnavailable = errors.New("eventbus unavailable — outbox accumulating (fault isolation)")
	ErrAlreadyConsumed     = errors.New("event already consumed (idempotency)")
)

type Publisher struct {
	db        *sql.DB
	mu        sync.Mutex
	mode      Mode
	dlq       []Event
	published map[string]bool
}

func NewPublisher(db *sql.DB) *Publisher {
	return &Publisher{
		db:        db,
		mode:      ModeNormal,
		published: make(map[string]bool),
	}
}

func (p *Publisher) Write(ctx context.Context, tx *sql.Tx, aggType, aggID, eventType, tenantID, corrID, causID string, payload []byte) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO outbox.events (aggregate_type, aggregate_id, event_type, payload, tenant_id, correlation_id, causation_id)
		 VALUES ($1, $2, $3, $4::jsonb, $5::uuid, $6, $7)`,
		aggType, aggID, eventType, payload, tenantID, corrID, causID)
	return err
}

func (p *Publisher) PublishPending(ctx context.Context, bus EventBus) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rows, err := p.db.QueryContext(ctx,
		`SELECT event_id, aggregate_type, aggregate_id, event_type, payload::text, tenant_id::text,
		        correlation_id, causation_id, retry_count
		 FROM outbox.events WHERE status = 'pending'
		 ORDER BY created_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var payloadStr string
		rows.Scan(&e.EventID, &e.AggregateType, &e.AggregateID, &e.EventType, &payloadStr,
			&e.TenantID, &e.CorrelationID, &e.CausationID, &e.RetryCount)
		events = append(events, e)
	}

	published := 0
	for _, e := range events {
		if err := bus.Publish(ctx, e); err != nil {
			if errors.Is(err, ErrEventBusUnavailable) {
				p.mode = ModeDegraded
				return published, ErrEventBusUnavailable
			}
			p.retryOrFail(ctx, e)
			continue
		}
		p.db.ExecContext(ctx, "UPDATE outbox.events SET status='published', published_at=now() WHERE event_id=$1", e.EventID)
		p.published[e.EventID] = true
		published++
	}
	if p.mode == ModeDegraded && published > 0 {
		p.mode = ModeRecovery
	}
	return published, nil
}

func (p *Publisher) retryOrFail(ctx context.Context, e Event) {
	maxRetries := 5
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	if e.RetryCount >= maxRetries {
		p.db.ExecContext(ctx, "UPDATE outbox.events SET status='dlq' WHERE event_id=$1", e.EventID)
		p.dlq = append(p.dlq, e)
		log.Printf("DLQ: event %s moved to dead letter queue after %d retries", e.EventID, e.RetryCount)
		return
	}
	nextRetry := time.Now().Add(backoffs[min(e.RetryCount, len(backoffs)-1)])
	p.db.ExecContext(ctx, "UPDATE outbox.events SET retry_count=retry_count+1, next_retry_at=$2 WHERE event_id=$1", e.EventID, nextRetry)
}

func (p *Publisher) Mode() Mode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mode
}

func (p *Publisher) DLQ() []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dlq
}

func (p *Publisher) SetMode(m Mode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = m
}

type EventBus interface {
	Publish(ctx context.Context, e Event) error
}

type MemoryBus struct {
	mu        sync.Mutex
	events    []Event
	available bool
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{available: true}
}

func (b *MemoryBus) Publish(ctx context.Context, e Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.available {
		return ErrEventBusUnavailable
	}
	b.events = append(b.events, e)
	return nil
}

func (b *MemoryBus) SetAvailable(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.available = v
}

func (b *MemoryBus) Events() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.events
}

type IdempotentConsumer struct {
	consumed map[string]bool
	mu       sync.Mutex
}

func NewIdempotentConsumer() *IdempotentConsumer {
	return &IdempotentConsumer{consumed: make(map[string]bool)}
}

func (c *IdempotentConsumer) Consume(e Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.consumed[e.EventID] {
		return ErrAlreadyConsumed
	}
	c.consumed[e.EventID] = true
	return nil
}

func (c *IdempotentConsumer) ConsumedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.consumed)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Publisher) Stats() ProjectionResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	return ProjectionResult{
		Mode:        p.mode,
		DLQCount:    len(p.dlq),
		FailedCount: 0,
	}
}

var _ = fmt.Sprintf
