package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/wujupeng/ebcx/internal/enterprise"
)

func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

type CommandIdempotencyRepositoryPostgreSQL struct{}

func NewCommandIdempotencyRepositoryPostgreSQL() *CommandIdempotencyRepositoryPostgreSQL {
	return &CommandIdempotencyRepositoryPostgreSQL{}
}

func (r *CommandIdempotencyRepositoryPostgreSQL) CheckAndReserve(
	ctx context.Context, tx *sql.Tx, commandID, tenantID, commandType, aggregateID string,
) (*CommandFirstResult, error) {
	if commandID == "" {
		return nil, enterprise.ErrEnterpriseCommandIDRequired
	}

	var status string
	var resultVersion sql.NullInt64
	var resultEventID, resultEvidenceID sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT status, result_version, result_event_id, result_evidence_id
		FROM business.command_idempotency
		WHERE command_id = $1 AND tenant_id = $2
	`, commandID, tenantID).Scan(&status, &resultVersion, &resultEventID, &resultEvidenceID)

	if err == nil {
		switch status {
		case "success":
			return &CommandFirstResult{
				ResultVersion:    resultVersion.Int64,
				ResultEventID:    resultEventID.String,
				ResultEvidenceID: resultEvidenceID.String,
			}, nil
		case "failed":
			_, _ = tx.ExecContext(ctx, `
				DELETE FROM business.command_idempotency
				WHERE command_id = $1 AND tenant_id = $2
			`, commandID, tenantID)
			return nil, enterprise.ErrIdempotencyRecordFailed
		case "pending":
			return nil, enterprise.ErrEnterpriseIdempotencyConflict
		}
	}

	if err != sql.ErrNoRows {
		if isUniqueViolation(err) {
			return nil, enterprise.ErrEnterpriseIdempotencyConflict
		}
		return nil, fmt.Errorf("failed to query idempotency record: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO business.command_idempotency
			(command_id, tenant_id, aggregate_id, command_type, status)
		VALUES ($1, $2, $3, $4, 'pending')
	`, commandID, tenantID, nullableString(aggregateID), commandType)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, enterprise.ErrEnterpriseIdempotencyConflict
		}
		return nil, fmt.Errorf("failed to insert idempotency record: %w", err)
	}

	return nil, nil
}

func (r *CommandIdempotencyRepositoryPostgreSQL) MarkSuccess(
	ctx context.Context, tx *sql.Tx, commandID, tenantID string,
	resultVersion int64, resultEventID, resultEvidenceID string,
) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE business.command_idempotency
		SET status = 'success', result_version = $3, result_event_id = $4, result_evidence_id = $5
		WHERE command_id = $1 AND tenant_id = $2
	`, commandID, tenantID, resultVersion, resultEventID, resultEvidenceID)
	if err != nil {
		return fmt.Errorf("failed to mark idempotency success: %w", err)
	}
	return nil
}

func (r *CommandIdempotencyRepositoryPostgreSQL) MarkFailed(
	ctx context.Context, tx *sql.Tx, commandID, tenantID string,
) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE business.command_idempotency
		SET status = 'failed'
		WHERE command_id = $1 AND tenant_id = $2
	`, commandID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to mark idempotency failed: %w", err)
	}
	return nil
}
