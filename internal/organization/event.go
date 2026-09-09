package organization

import "time"

type DomainEvent interface {
	GetEventID() string
	GetEventType() string
	GetOrgID() string
	GetTenantID() string
	GetTimestamp() time.Time
}

type OrganizationCreatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	OrgID            string    `json:"org_id"`
	EnterpriseID     string    `json:"enterprise_id"`
	ParentID         string    `json:"parent_id"`
	Name             string    `json:"name"`
	Code             string    `json:"code"`
	Level            int       `json:"level"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *OrganizationCreatedEvent) GetEventID() string      { return e.EventID }
func (e *OrganizationCreatedEvent) GetEventType() string    { return e.EventType }
func (e *OrganizationCreatedEvent) GetOrgID() string        { return e.OrgID }
func (e *OrganizationCreatedEvent) GetTenantID() string     { return e.TenantID }
func (e *OrganizationCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type OrganizationUpdatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	OrgID            string    `json:"org_id"`
	EnterpriseID     string    `json:"enterprise_id"`
	ParentID         string    `json:"parent_id"`
	NewName          string    `json:"new_name"`
	NewCode          string    `json:"new_code"`
	Level            int       `json:"level"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *OrganizationUpdatedEvent) GetEventID() string      { return e.EventID }
func (e *OrganizationUpdatedEvent) GetEventType() string    { return e.EventType }
func (e *OrganizationUpdatedEvent) GetOrgID() string        { return e.OrgID }
func (e *OrganizationUpdatedEvent) GetTenantID() string     { return e.TenantID }
func (e *OrganizationUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type OrganizationMovedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	OrgID            string    `json:"org_id"`
	EnterpriseID     string    `json:"enterprise_id"`
	OldParentID      string    `json:"old_parent_id"`
	NewParentID      string    `json:"new_parent_id"`
	OldLevel         int       `json:"old_level"`
	NewLevel         int       `json:"new_level"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *OrganizationMovedEvent) GetEventID() string      { return e.EventID }
func (e *OrganizationMovedEvent) GetEventType() string    { return e.EventType }
func (e *OrganizationMovedEvent) GetOrgID() string        { return e.OrgID }
func (e *OrganizationMovedEvent) GetTenantID() string     { return e.TenantID }
func (e *OrganizationMovedEvent) GetTimestamp() time.Time { return e.Timestamp }