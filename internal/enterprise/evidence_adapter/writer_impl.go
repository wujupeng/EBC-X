package evidence_adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/enterprise"
	"github.com/wujupeng/ebcx/internal/evidence"
)

type EnterpriseEvidenceWriterImpl struct {
	createdBy string
	gitCommit string
}

func NewEnterpriseEvidenceWriter(createdBy string) *EnterpriseEvidenceWriterImpl {
	return &EnterpriseEvidenceWriterImpl{createdBy: createdBy, gitCommit: getGitCommit()}
}

func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func (w *EnterpriseEvidenceWriterImpl) Write(
	ctx context.Context, tx *sql.Tx, agg *enterprise.EnterpriseAggregate,
	event enterprise.DomainEvent, mutationType string,
) (*evidence.Record, error) {
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", agg.TenantID)

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, chainID); err != nil {
		return nil, fmt.Errorf("failed to acquire chain advisory lock: %w", err)
	}

	var sequenceNo int64
	var previousHash string
	err := tx.QueryRowContext(ctx, `
		SELECT sequence_no, evidence_hash
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no DESC
		LIMIT 1
		FOR UPDATE
	`, chainID).Scan(&sequenceNo, &previousHash)

	if err == sql.ErrNoRows {
		sequenceNo = 1
		previousHash = evidence.GenesisHash(chainID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query chain predecessor: %w", err)
	} else {
		sequenceNo = sequenceNo + 1
	}

	evidenceID := uuid.NewString()
	payload := map[string]any{
		"enterpriseId":     agg.EnterpriseID,
		"name":             agg.Name,
		"version":          agg.Version,
		"tenantId":         agg.TenantID,
		"sourceEvidenceId": agg.SourceEvidenceID,
	}
	payloadBytes, _ := json.Marshal(payload)

	evHash := evidence.ChainHash(chainID, sequenceNo, previousHash, payloadBytes, event.GetEventID(), "")

	provenance := map[string]any{
		"event_id":    event.GetEventID(),
		"evidence_id": evidenceID,
		"operator":    w.createdBy,
		"git_commit":  w.gitCommit,
	}

	record := &evidence.Record{
		EvidenceID:           evidenceID,
		ChainID:              chainID,
		SequenceNo:           sequenceNo,
		PreviousEvidenceHash: previousHash,
		EvidenceHash:         evHash,
		EvidenceType:         evidence.EvidenceMandatory,
		Payload:              payload,
		SourceEventID:        event.GetEventID(),
		TransactionID:        "",
		TenantID:             agg.TenantID,
		Provenance:           provenance,
		CreatedBy:            w.createdBy,
		CreatedAt:            time.Now().UTC(),
		Version:              1,
		LegalHold:            false,
	}

	provenanceBytes, _ := json.Marshal(provenance)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO evidence.evidence_ledger
			(evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash,
			 evidence_type, payload, source_event_id, transaction_id, tenant_id,
			 provenance, created_by, created_at, version, legal_hold)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, record.EvidenceID, record.ChainID, record.SequenceNo, record.PreviousEvidenceHash,
		record.EvidenceHash, string(record.EvidenceType), payloadBytes,
		record.SourceEventID, record.TransactionID, record.TenantID,
		provenanceBytes, record.CreatedBy, record.CreatedAt, record.Version, record.LegalHold)
	if err != nil {
		return nil, fmt.Errorf("failed to insert evidence record: %w", err)
	}

	return record, nil
}
