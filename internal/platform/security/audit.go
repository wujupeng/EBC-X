package security

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrAuditAppendOnly = errors.New("audit log is append-only — UPDATE/DELETE prohibited")
	ErrAuditNotFound   = errors.New("audit entry not found")
)

type AuditEntry struct {
	AuditID     string    `json:"auditId"`
	TenantID    string    `json:"tenantId"`
	UserID      string    `json:"userId"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resourceId"`
	Decision    string    `json:"decision"`
	Reason      string    `json:"reason"`
	IPAddress   string    `json:"ipAddress"`
	UserAgent   string    `json:"userAgent"`
	Timestamp   time.Time `json:"timestamp"`
	Metadata    string    `json:"metadata,omitempty"`
}

type AuditLogger interface {
	Append(ctx context.Context, entry AuditEntry) error
	Query(ctx context.Context, tenantID string, filters AuditQueryFilters) ([]AuditEntry, error)
	GetByID(ctx context.Context, tenantID, auditID string) (*AuditEntry, error)
}

type AuditQueryFilters struct {
	UserID    string
	Action    string
	Resource  string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type InMemoryAuditLogger struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

func NewInMemoryAuditLogger() *InMemoryAuditLogger {
	return &InMemoryAuditLogger{
		entries: make([]AuditEntry, 0),
	}
}

func (l *InMemoryAuditLogger) Append(ctx context.Context, entry AuditEntry) error {
	if entry.TenantID == "" {
		return errors.New("audit entry missing tenantId")
	}
	if entry.Action == "" {
		return errors.New("audit entry missing action")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, entry)
	return nil
}

func (l *InMemoryAuditLogger) Query(ctx context.Context, tenantID string, filters AuditQueryFilters) ([]AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]AuditEntry, 0)
	for _, e := range l.entries {
		if e.TenantID != tenantID {
			continue
		}
		if filters.UserID != "" && e.UserID != filters.UserID {
			continue
		}
		if filters.Action != "" && e.Action != filters.Action {
			continue
		}
		if filters.Resource != "" && e.Resource != filters.Resource {
			continue
		}
		if !filters.StartTime.IsZero() && e.Timestamp.Before(filters.StartTime) {
			continue
		}
		if !filters.EndTime.IsZero() && e.Timestamp.After(filters.EndTime) {
			continue
		}
		result = append(result, e)
	}

	if filters.Limit > 0 && len(result) > filters.Limit {
		result = result[:filters.Limit]
	}
	return result, nil
}

func (l *InMemoryAuditLogger) GetByID(ctx context.Context, tenantID, auditID string) (*AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, e := range l.entries {
		if e.AuditID == auditID && e.TenantID == tenantID {
			return &e, nil
		}
	}
	return nil, ErrAuditNotFound
}

func (l *InMemoryAuditLogger) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func AuditEntryFromClaims(claims *Claims, action, resource, resourceID, decision, reason, ipAddress, userAgent string) AuditEntry {
	return AuditEntry{
		TenantID:   claims.TenantID,
		UserID:     claims.UserID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Decision:   decision,
		Reason:     reason,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Timestamp:  time.Now(),
	}
}

func (e *AuditEntry) String() string {
	return fmt.Sprintf("AuditEntry{tenant=%s, user=%s, action=%s, resource=%s/%s, decision=%s}",
		e.TenantID, e.UserID, e.Action, e.Resource, e.ResourceID, e.Decision)
}