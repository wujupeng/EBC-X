package organization

type CreateOrganizationCommand struct {
	CommandID        string
	EnterpriseID     string
	ParentID         string
	Name             string
	Code             string
	TenantID         string
	SourceEvidenceID string
}

type UpdateOrganizationCommand struct {
	CommandID        string
	OrgID            string
	NewName          string
	NewCode          string
	ExpectedVersion  int64
	SourceEvidenceID string
}

type MoveOrganizationCommand struct {
	CommandID        string
	OrgID            string
	NewParentID      string
	ExpectedVersion  int64
	SourceEvidenceID string
}