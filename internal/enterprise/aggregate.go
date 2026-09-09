package enterprise

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxNameLength = 256
)

type EnterpriseAggregate struct {
	EnterpriseID     string
	Name             string
	Version          int64
	SourceEvidenceID string
	TenantID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewEnterpriseAggregate() *EnterpriseAggregate {
	return &EnterpriseAggregate{}
}

func (a *EnterpriseAggregate) CreateEnterprise(cmd CreateEnterpriseCommand) (*EnterpriseCreatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrEnterpriseCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrEnterpriseInvalidCommandID
	}
	if cmd.TenantID == "" {
		return nil, ErrTenantContextMismatch
	}
	if err := validateName(cmd.Name); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	a.EnterpriseID = uuid.NewString()
	a.Name = cmd.Name
	a.Version = 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.TenantID = cmd.TenantID
	a.CreatedAt = now
	a.UpdatedAt = now

	event := &EnterpriseCreatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "EnterpriseCreated",
		EnterpriseID:     a.EnterpriseID,
		Name:             a.Name,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func (a *EnterpriseAggregate) UpdateEnterprise(cmd UpdateEnterpriseCommand) (*EnterpriseUpdatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrEnterpriseCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrEnterpriseInvalidCommandID
	}
	if cmd.EnterpriseID == "" {
		return nil, ErrEnterpriseNotFound
	}
	if cmd.ExpectedVersion <= 0 {
		return nil, ErrEnterpriseVersionMonotonic
	}
	if err := validateName(cmd.NewName); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	a.Name = cmd.NewName
	a.Version = cmd.ExpectedVersion + 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.UpdatedAt = now

	event := &EnterpriseUpdatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "EnterpriseUpdated",
		EnterpriseID:     a.EnterpriseID,
		Name:             a.Name,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func validateName(name string) error {
	if name == "" {
		return ErrEnterpriseInvalidName
	}
	if len(name) > MaxNameLength {
		return ErrEnterpriseInvalidName
	}
	return nil
}
