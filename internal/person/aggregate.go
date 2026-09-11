package person

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxNameLength    = 128
	MaxEmployeeNoLen = 64
)

type PersonAggregate struct {
	PersonID         string
	OrgID            string
	Name             string
	EmployeeNo       string
	Roles            []RoleRef
	Version          int64
	SourceEvidenceID string
	TenantID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewPersonAggregate() *PersonAggregate {
	return &PersonAggregate{Roles: []RoleRef{}}
}

func (a *PersonAggregate) CreatePerson(cmd CreatePersonCommand) (*PersonCreatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrPersonCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrPersonInvalidCommandID
	}
	if cmd.TenantID == "" {
		return nil, ErrTenantContextMismatch
	}
	if cmd.OrgID == "" {
		return nil, ErrPersonOrganizationNotFound
	}
	if _, err := uuid.Parse(cmd.OrgID); err != nil {
		return nil, ErrPersonOrganizationNotFound
	}
	if err := validateName(cmd.Name); err != nil {
		return nil, err
	}
	if err := validateEmployeeNo(cmd.EmployeeNo); err != nil {
		return nil, err
	}
	if err := validateRoles(cmd.Roles); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	a.PersonID = uuid.NewString()
	a.OrgID = cmd.OrgID
	a.Name = cmd.Name
	a.EmployeeNo = cmd.EmployeeNo
	a.Roles = cmd.Roles
	if a.Roles == nil {
		a.Roles = []RoleRef{}
	}
	a.Version = 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.TenantID = cmd.TenantID
	a.CreatedAt = now
	a.UpdatedAt = now

	event := &PersonCreatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "PersonCreated",
		PersonID:         a.PersonID,
		OrgID:            a.OrgID,
		Name:             a.Name,
		EmployeeNo:       a.EmployeeNo,
		Roles:            a.Roles,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func (a *PersonAggregate) UpdatePerson(cmd UpdatePersonCommand) (*PersonUpdatedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrPersonCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrPersonInvalidCommandID
	}
	if cmd.PersonID == "" {
		return nil, ErrPersonNotFound
	}
	if cmd.ExpectedVersion <= 0 {
		return nil, ErrPersonVersionMonotonic
	}
	if err := validateName(cmd.NewName); err != nil {
		return nil, err
	}
	if err := validateEmployeeNo(cmd.NewEmployeeNo); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	a.Name = cmd.NewName
	a.EmployeeNo = cmd.NewEmployeeNo
	a.Version = cmd.ExpectedVersion + 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.UpdatedAt = now

	event := &PersonUpdatedEvent{
		EventID:          uuid.NewString(),
		EventType:        "PersonUpdated",
		PersonID:         a.PersonID,
		OrgID:            a.OrgID,
		NewName:          a.Name,
		NewEmployeeNo:    a.EmployeeNo,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func (a *PersonAggregate) AssignRole(cmd AssignRoleCommand) (*RoleAssignedEvent, error) {
	if cmd.CommandID == "" {
		return nil, ErrPersonCommandIDRequired
	}
	if _, err := uuid.Parse(cmd.CommandID); err != nil {
		return nil, ErrPersonInvalidCommandID
	}
	if cmd.PersonID == "" {
		return nil, ErrPersonNotFound
	}
	if cmd.ExpectedVersion <= 0 {
		return nil, ErrPersonVersionMonotonic
	}
	if _, err := uuid.Parse(cmd.RoleID); err != nil {
		return nil, ErrPersonInvalidRoleRef
	}
	for _, r := range a.Roles {
		if r.RoleID == cmd.RoleID {
			return nil, ErrPersonRoleAlreadyAssigned
		}
	}

	now := time.Now().UTC()
	a.Roles = append(a.Roles, RoleRef{RoleID: cmd.RoleID})
	a.Version = cmd.ExpectedVersion + 1
	a.SourceEvidenceID = cmd.SourceEvidenceID
	a.UpdatedAt = now

	event := &RoleAssignedEvent{
		EventID:          uuid.NewString(),
		EventType:        "RoleAssigned",
		PersonID:         a.PersonID,
		RoleID:           cmd.RoleID,
		Version:          a.Version,
		SourceEvidenceID: a.SourceEvidenceID,
		TenantID:         a.TenantID,
		Timestamp:        now,
	}

	return event, nil
}

func validateName(name string) error {
	if name == "" {
		return ErrPersonInvalidName
	}
	if len(name) > MaxNameLength {
		return ErrPersonInvalidName
	}
	return nil
}

func validateEmployeeNo(employeeNo string) error {
	if employeeNo == "" {
		return ErrPersonInvalidEmployeeNo
	}
	if len(employeeNo) > MaxEmployeeNoLen {
		return ErrPersonInvalidEmployeeNo
	}
	return nil
}

func validateRoles(roles []RoleRef) error {
	seen := make(map[string]bool)
	for _, r := range roles {
		if _, err := uuid.Parse(r.RoleID); err != nil {
			return ErrPersonInvalidRoleRef
		}
		if seen[r.RoleID] {
			return ErrPersonRoleAlreadyAssigned
		}
		seen[r.RoleID] = true
	}
	return nil
}
