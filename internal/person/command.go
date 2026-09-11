package person

type CreatePersonCommand struct {
	CommandID        string
	OrgID            string
	Name             string
	EmployeeNo       string
	Roles            []RoleRef
	TenantID         string
	SourceEvidenceID string
}

type UpdatePersonCommand struct {
	CommandID        string
	PersonID         string
	NewName          string
	NewEmployeeNo    string
	ExpectedVersion  int64
	TenantID         string
	SourceEvidenceID string
}

type AssignRoleCommand struct {
	CommandID        string
	PersonID         string
	RoleID           string
	ExpectedVersion  int64
	TenantID         string
	SourceEvidenceID string
}
