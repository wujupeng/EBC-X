package organization

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxNameLength  = 256
	MaxCodeLength  = 64
	MaxTreeLevel   = 5
)

type OrganizationAggregate struct {
	OrgID            string
	EnterpriseID     string
	ParentID         string
	Name             string
	Code             string
	Level            int
	Version          int64
	SourceEvidenceID string
	TenantID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewOrganizationAggregate() *OrganizationAggregate {
	return &OrganizationAggregate{}
}

func (a *OrganizationAggregate) CreateOrganization(cmd CreateOrganizationCommand, parentLevel int) (*OrganizationCreatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrOrgCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrOrgInvalidCommandID
	}
	if cmd.TenantID == "" {
		return nil, ErrTenantContextMismatch
	}
	if cmd.EnterpriseID == "" {
		return nil, ErrOrgEnterpriseNotFound
	}
	if err := validateName(cmd.Name); err != nil {
		return nil, err
	}
	if err := validateCode(cmd.Code); err != nil {
		return nil, err
	}

	level := parentLevel + 1
	if level > MaxTreeLevel {
		return nil, ErrOrgLevelExceedMax
	}

	now := time.Now().UTC()
	a.OrgID = uuid.NewString()
	a.EnterpriseID = cmd.EnterpriseID
	a.ParentID = cmd.ParentID
	a.Name = cmd.Name
	a.Code = cmd.Code
	a.Level = level
	a.Version = 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.TenantID = cmd.TenantID
	a.CreatedAt = now
	a.UpdatedAt = now

	event := &OrganizationCreatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "OrganizationCreated",
		OrgID:            a.OrgID,
		EnterpriseID:     a.EnterpriseID,
		ParentID:         a.ParentID,
		Name:             a.Name,
		Code:             a.Code,
		Level:            a.Level,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func (a *OrganizationAggregate) UpdateOrganization(cmd UpdateOrganizationCommand) (*OrganizationUpdatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrOrgCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrOrgInvalidCommandID
	}
	if cmd.OrgID == "" {
		return nil, ErrOrgNotFound
	}
	if cmd.ExpectedVersion <= 0 {
		return nil, ErrOrgVersionMonotonic
	}
	if err := validateName(cmd.NewName); err != nil {
		return nil, err
	}
	if err := validateCode(cmd.NewCode); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	a.Name = cmd.NewName
	a.Code = cmd.NewCode
	a.Version = cmd.ExpectedVersion + 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.UpdatedAt = now

	event := &OrganizationUpdatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "OrganizationUpdated",
		OrgID:            a.OrgID,
		EnterpriseID:     a.EnterpriseID,
		ParentID:         a.ParentID,
		NewName:          a.Name,
		NewCode:          a.Code,
		Level:            a.Level,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func validateName(name string) error {
	if name == "" {
		return ErrOrgInvalidName
	}
	if len(name) > MaxNameLength {
		return ErrOrgInvalidName
	}
	return nil
}

func validateCode(code string) error {
	if code == "" {
		return ErrOrgInvalidCode
	}
	if len(code) > MaxCodeLength {
		return ErrOrgInvalidCode
	}
	return nil
}