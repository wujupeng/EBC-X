package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/wujupeng/ebcx/internal/organization"
)

func isUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

type OrganizationRepositoryPostgreSQL struct{}

func NewOrganizationRepositoryPostgreSQL() *OrganizationRepositoryPostgreSQL {
	return &OrganizationRepositoryPostgreSQL{}
}

func (r *OrganizationRepositoryPostgreSQL) Insert(ctx context.Context, tx *sql.Tx, agg *organization.OrganizationAggregate) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO business.organizations
			(org_id, enterprise_id, parent_id, name, code, level, version, source_evidence_id, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, agg.OrgID, agg.EnterpriseID, nullableString(agg.ParentID), agg.Name, agg.Code,
		agg.Level, agg.Version, nullableString(agg.SourceEvidenceID),
		agg.TenantID, agg.CreatedAt, agg.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return organization.ErrOrgDuplicateCode
		}
		return fmt.Errorf("failed to insert organization: %w", err)
	}
	return nil
}

func (r *OrganizationRepositoryPostgreSQL) FindByID(ctx context.Context, tx *sql.Tx, orgID string, forUpdate bool) (*organization.OrganizationAggregate, error) {
	agg := &organization.OrganizationAggregate{}
	var parentID, sourceEvidenceID sql.NullString

	query := `SELECT org_id, enterprise_id, parent_id, name, code, level, version, source_evidence_id, tenant_id, created_at, updated_at FROM business.organizations WHERE org_id = $1`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	err := tx.QueryRowContext(ctx, query, orgID).Scan(
		&agg.OrgID, &agg.EnterpriseID, &parentID, &agg.Name, &agg.Code,
		&agg.Level, &agg.Version, &sourceEvidenceID, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, organization.ErrOrgNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}
	agg.ParentID = parentID.String
	agg.SourceEvidenceID = sourceEvidenceID.String
	return agg, nil
}

func (r *OrganizationRepositoryPostgreSQL) UpdateWithCAS(
	ctx context.Context, tx *sql.Tx, orgID string, newName string, newCode string,
	expectedVersion int64, sourceEvidenceID string,
) (*organization.OrganizationAggregate, error) {
	agg := &organization.OrganizationAggregate{}
	var parentID, srcEvid sql.NullString
	err := tx.QueryRowContext(ctx, `
		UPDATE business.organizations
		SET name = $1, code = $2, version = version + 1, source_evidence_id = $3, updated_at = now()
		WHERE org_id = $4 AND version = $5
		RETURNING org_id, enterprise_id, parent_id, name, code, level, version, source_evidence_id, tenant_id, created_at, updated_at
	`, newName, newCode, nullableString(sourceEvidenceID), orgID, expectedVersion).Scan(
		&agg.OrgID, &agg.EnterpriseID, &parentID, &agg.Name, &agg.Code,
		&agg.Level, &agg.Version, &srcEvid, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		var currentVersion int64
		err2 := tx.QueryRowContext(ctx, `SELECT version FROM business.organizations WHERE org_id = $1`, orgID).Scan(&currentVersion)
		if err2 == sql.ErrNoRows {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("%w (current version: %d, expected: %d)", organization.ErrOrgVersionConflict, currentVersion, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update organization with CAS: %w", err)
	}
	agg.ParentID = parentID.String
	agg.SourceEvidenceID = srcEvid.String
	return agg, nil
}

func (r *OrganizationRepositoryPostgreSQL) MoveWithCAS(
	ctx context.Context, tx *sql.Tx, orgID string, newParentID string, newLevel int,
	expectedVersion int64, sourceEvidenceID string,
) (*organization.OrganizationAggregate, error) {
	agg := &organization.OrganizationAggregate{}
	var parentID, srcEvid sql.NullString
	err := tx.QueryRowContext(ctx, `
		UPDATE business.organizations
		SET parent_id = $1, level = $2, version = version + 1, source_evidence_id = $3, updated_at = now()
		WHERE org_id = $4 AND version = $5
		RETURNING org_id, enterprise_id, parent_id, name, code, level, version, source_evidence_id, tenant_id, created_at, updated_at
	`, nullableString(newParentID), newLevel, nullableString(sourceEvidenceID), orgID, expectedVersion).Scan(
		&agg.OrgID, &agg.EnterpriseID, &parentID, &agg.Name, &agg.Code,
		&agg.Level, &agg.Version, &srcEvid, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		var currentVersion int64
		err2 := tx.QueryRowContext(ctx, `SELECT version FROM business.organizations WHERE org_id = $1`, orgID).Scan(&currentVersion)
		if err2 == sql.ErrNoRows {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("%w (current version: %d, expected: %d)", organization.ErrOrgVersionConflict, currentVersion, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to move organization with CAS: %w", err)
	}
	agg.ParentID = parentID.String
	agg.SourceEvidenceID = srcEvid.String
	return agg, nil
}

func (r *OrganizationRepositoryPostgreSQL) ExistsByCode(
	ctx context.Context, tx *sql.Tx, enterpriseID string, parentID string, code string, excludeOrgID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM business.organizations
			WHERE enterprise_id = $1 AND parent_id IS NOT DISTINCT FROM $2 AND code = $3
			  AND ($4::uuid IS NULL OR org_id <> $4::uuid)
		)
	`, enterpriseID, nullableString(parentID), code, nullableString(excludeOrgID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check code existence: %w", err)
	}
	return exists, nil
}

func (r *OrganizationRepositoryPostgreSQL) GetSubtree(ctx context.Context, tx *sql.Tx, orgID string) ([]SubtreeNode, error) {
	rows, err := tx.QueryContext(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT org_id, level, 0 AS depth
			FROM business.organizations
			WHERE org_id = $1
			UNION ALL
			SELECT o.org_id, o.level, s.depth + 1
			FROM business.organizations o
			JOIN subtree s ON o.parent_id = s.org_id
		)
		SELECT org_id, level, depth FROM subtree WHERE depth > 0 ORDER BY depth
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtree: %w", err)
	}
	defer rows.Close()

	var nodes []SubtreeNode
	for rows.Next() {
		var node SubtreeNode
		if err := rows.Scan(&node.OrgID, &node.Level, &node.Depth); err != nil {
			return nil, fmt.Errorf("failed to scan subtree node: %w", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (r *OrganizationRepositoryPostgreSQL) UpdateSubtreeLevels(
	ctx context.Context, tx *sql.Tx, descendantIDs []string, levelDelta int,
) error {
	if len(descendantIDs) == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE business.organizations
		SET level = level + $1, updated_at = now()
		WHERE org_id = ANY($2::uuid[])
	`, levelDelta, pq.Array(descendantIDs))
	if err != nil {
		return fmt.Errorf("failed to update subtree levels: %w", err)
	}
	return nil
}

func (r *OrganizationRepositoryPostgreSQL) CheckEnterpriseExists(
	ctx context.Context, tx *sql.Tx, enterpriseID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM business.enterprises WHERE enterprise_id = $1)
	`, enterpriseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check enterprise existence: %w", err)
	}
	return exists, nil
}