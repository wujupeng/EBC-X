package evidence

import "time"

type EvidenceType string

const (
	EvidenceMandatory EvidenceType = "mandatory"
	EvidenceDecision  EvidenceType = "decision"
	EvidenceExecution EvidenceType = "execution"
)

type Record struct {
	EvidenceID           string         `json:"evidence_id"`
	ChainID              string         `json:"chain_id"`
	SequenceNo           int64          `json:"sequence_no"`
	PreviousEvidenceHash string         `json:"previous_evidence_hash"`
	EvidenceHash         string         `json:"evidence_hash"`
	EvidenceType         EvidenceType   `json:"evidence_type"`
	Payload              map[string]any `json:"payload"`
	SourceEventID        string         `json:"source_event_id"`
	TransactionID        string         `json:"transaction_id"`
	TenantID             string         `json:"tenant_id"`
	Provenance           map[string]any `json:"provenance"`
	CorrelationID        string         `json:"correlation_id,omitempty"`
	CausationID          string         `json:"causation_id,omitempty"`
	CreatedBy            string         `json:"created_by"`
	CreatedAt            time.Time      `json:"created_at"`
	Version              int            `json:"version"`
	LegalHold            bool           `json:"legal_hold"`
}

type ChainVerificationResult struct {
	Valid      bool
	BrokenAt   int64
	Reason     string
	TotalNodes int
}

type TamperResult struct {
	Detected   bool
	EvidenceID string
	Reason     string
}
