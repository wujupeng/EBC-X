package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wujupeng/ebcx/internal/enterprise"
)

type EnterpriseRepositoryPostgreSQL struct{}

func NewEnterpriseRepositoryPostgreSQL() *EnterpriseRepositoryPostgreSQL {
	return &EnterpriseRepositoryPostgreSQL{}
}

func (r *EnterpriseRepositoryPostgreSQL) Insert(ctx context.Context, tx *sql.Tx, agg *enterprise.EnterpriseAggregate) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO business.enterprises
			(enterprise_id, name, version, source_evidence_id, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, agg.EnterpriseID, agg.Name, agg.Version, nullableString(agg.SourceEvidenceID),
		agg.TenantID, agg.CreatedAt, agg.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return enterprise.ErrEnterpriseDuplicateName
		}
		return fmt.Errorf("failed to insert enterprise: %w", err)
	}
	return nil
}

func (r *EnterpriseRepositoryPostgreSQL) FindByID(ctx context.Context, tx *sql.Tx, enterpriseID string) (*enterprise.EnterpriseAggregate, error) {
	agg := &enterprise.EnterpriseAggregate{}
	var sourceEvidenceID sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT enterprise_id, name, version, source_evidence_id, tenant_id, created_at, updated_at
		FROM business.enterprises
		WHERE enterprise_id = $1
	`, enterpriseID).Scan(&agg.EnterpriseID, &agg.Name, &agg.Version, &sourceEvidenceID,
		&agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, enterprise.ErrEnterpriseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find enterprise: %w", err)
	}
	agg.SourceEvidenceID = sourceEvidenceID.String
	return agg, nil
}

func (r *EnterpriseRepositoryPostgreSQL) UpdateWithCAS(
	ctx context.Context, tx *sql.Tx, enterpriseID string, newName string,
	expectedVersion int64, sourceEvidenceID string,
) (*enterprise.EnterpriseAggregate, error) {
	agg := &enterprise.EnterpriseAggregate{}
	var srcEvid sql.NullString
	err := tx.QueryRowContext(ctx, `
		UPDATE business.enterprises
		SET name = $1, version = version + 1, source_evidence_id = $2, updated_at = now()
		WHERE enterprise_id = $3 AND version = $4
		RETURNING enterprise_id, name, version, source_evidence_id, tenant_id, created_at, updated_at
	`, newName, nullableString(sourceEvidenceID), enterpriseID, expectedVersion).Scan(
		&agg.EnterpriseID, &agg.Name, &agg.Version, &srcEvid,
		&agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		var currentVersion int64
		err2 := tx.QueryRowContext(ctx, `
			SELECT version FROM business.enterprises WHERE enterprise_id = $1
		`, enterpriseID).Scan(&currentVersion)
		if err2 == sql.ErrNoRows {
			return nil, enterprise.ErrEnterpriseNotFound
		}
		return nil, fmt.Errorf("%w (current version: %d, expected: %d)",
			enterprise.ErrEnterpriseVersionConflict, currentVersion, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update enterprise with CAS: %w", err)
	}
	agg.SourceEvidenceID = srcEvid.String
	return agg, nil
}

func (r *EnterpriseRepositoryPostgreSQL) ExistsByName(
	ctx context.Context, tx *sql.Tx, tenantID, name string, excludeID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM business.enterprises
			WHERE tenant_id = $1 AND name = $2
			  AND ($3::uuid IS NULL OR enterprise_id <> $3::uuid)
		)
	`, tenantID, name, nullableString(excludeID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check name existence: %w", err)
	}
	return exists, nil
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
