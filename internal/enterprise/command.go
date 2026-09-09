package enterprise

type CreateEnterpriseCommand struct {
	CommandID        string
	Name             string
	TenantID         string
	SourceEvidenceID string
}

type UpdateEnterpriseCommand struct {
	CommandID        string
	EnterpriseID     string
	TenantID         string
	NewName          string
	ExpectedVersion  int64
	SourceEvidenceID string
}
