package enterprise

import "time"

type DomainEvent interface {
	GetEventID() string
	GetEventType() string
	GetEnterpriseID() string
	GetTenantID() string
	GetTimestamp() time.Time
}

type EnterpriseCreatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	EnterpriseID     string    `json:"enterprise_id"`
	Name             string    `json:"name"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *EnterpriseCreatedEvent) GetEventID() string      { return e.EventID }
func (e *EnterpriseCreatedEvent) GetEventType() string    { return e.EventType }
func (e *EnterpriseCreatedEvent) GetEnterpriseID() string { return e.EnterpriseID }
func (e *EnterpriseCreatedEvent) GetTenantID() string     { return e.TenantID }
func (e *EnterpriseCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type EnterpriseUpdatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	EnterpriseID     string    `json:"enterprise_id"`
	Name             string    `json:"name"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *EnterpriseUpdatedEvent) GetEventID() string      { return e.EventID }
func (e *EnterpriseUpdatedEvent) GetEventType() string    { return e.EventType }
func (e *EnterpriseUpdatedEvent) GetEnterpriseID() string { return e.EnterpriseID }
func (e *EnterpriseUpdatedEvent) GetTenantID() string     { return e.TenantID }
func (e *EnterpriseUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
