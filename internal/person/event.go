package person

import "time"

type DomainEvent interface {
	GetEventID() string
	GetEventType() string
	GetPersonID() string
	GetTenantID() string
	GetTimestamp() time.Time
}

type PersonCreatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	PersonID         string    `json:"person_id"`
	OrgID            string    `json:"org_id"`
	Name             string    `json:"name"`
	EmployeeNo       string    `json:"employee_no"`
	Roles            []RoleRef `json:"roles"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *PersonCreatedEvent) GetEventID() string      { return e.EventID }
func (e *PersonCreatedEvent) GetEventType() string    { return e.EventType }
func (e *PersonCreatedEvent) GetPersonID() string     { return e.PersonID }
func (e *PersonCreatedEvent) GetTenantID() string     { return e.TenantID }
func (e *PersonCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type PersonUpdatedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	PersonID         string    `json:"person_id"`
	OrgID            string    `json:"org_id"`
	NewName          string    `json:"new_name"`
	NewEmployeeNo    string    `json:"new_employee_no"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *PersonUpdatedEvent) GetEventID() string      { return e.EventID }
func (e *PersonUpdatedEvent) GetEventType() string    { return e.EventType }
func (e *PersonUpdatedEvent) GetPersonID() string     { return e.PersonID }
func (e *PersonUpdatedEvent) GetTenantID() string     { return e.TenantID }
func (e *PersonUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type RoleAssignedEvent struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	PersonID         string    `json:"person_id"`
	RoleID           string    `json:"role_id"`
	Version          int64     `json:"version"`
	SourceEvidenceID string    `json:"source_evidence_id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
}

func (e *RoleAssignedEvent) GetEventID() string      { return e.EventID }
func (e *RoleAssignedEvent) GetEventType() string    { return e.EventType }
func (e *RoleAssignedEvent) GetPersonID() string     { return e.PersonID }
func (e *RoleAssignedEvent) GetTenantID() string     { return e.TenantID }
func (e *RoleAssignedEvent) GetTimestamp() time.Time { return e.Timestamp }
