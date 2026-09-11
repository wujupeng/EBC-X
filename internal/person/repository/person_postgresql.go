package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/wujupeng/ebcx/internal/person"
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

type PersonRepositoryPostgreSQL struct{}

func NewPersonRepositoryPostgreSQL() *PersonRepositoryPostgreSQL {
	return &PersonRepositoryPostgreSQL{}
}

func (r *PersonRepositoryPostgreSQL) Insert(ctx context.Context, tx *sql.Tx, agg *person.PersonAggregate) error {
	rolesJSON, err := json.Marshal(agg.Roles)
	if err != nil {
		return fmt.Errorf("failed to marshal roles: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO business.persons
			(person_id, org_id, name, employee_no, roles, version, source_evidence_id, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, agg.PersonID, agg.OrgID, agg.Name, agg.EmployeeNo, rolesJSON,
		agg.Version, nullableString(agg.SourceEvidenceID),
		agg.TenantID, agg.CreatedAt, agg.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return person.ErrPersonDuplicateEmployeeNo
		}
		return fmt.Errorf("failed to insert person: %w", err)
	}
	return nil
}

func (r *PersonRepositoryPostgreSQL) FindByID(ctx context.Context, tx *sql.Tx, personID string, forUpdate bool) (*person.PersonAggregate, error) {
	agg := &person.PersonAggregate{}
	var sourceEvidenceID sql.NullString
	var rolesJSON []byte

	query := `SELECT person_id, org_id, name, employee_no, roles, version, source_evidence_id, tenant_id, created_at, updated_at FROM business.persons WHERE person_id = $1`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	err := tx.QueryRowContext(ctx, query, personID).Scan(
		&agg.PersonID, &agg.OrgID, &agg.Name, &agg.EmployeeNo, &rolesJSON,
		&agg.Version, &sourceEvidenceID, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, person.ErrPersonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find person: %w", err)
	}
	agg.SourceEvidenceID = sourceEvidenceID.String
	if err := json.Unmarshal(rolesJSON, &agg.Roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}
	if agg.Roles == nil {
		agg.Roles = []person.RoleRef{}
	}
	return agg, nil
}

func (r *PersonRepositoryPostgreSQL) UpdateWithCAS(
	ctx context.Context, tx *sql.Tx, personID string, newName string, newEmployeeNo string,
	expectedVersion int64, sourceEvidenceID string,
) (*person.PersonAggregate, error) {
	agg := &person.PersonAggregate{}
	var srcEvid sql.NullString
	var rolesJSON []byte
	err := tx.QueryRowContext(ctx, `
		UPDATE business.persons
		SET name = $1, employee_no = $2, version = version + 1, source_evidence_id = $3, updated_at = now()
		WHERE person_id = $4 AND version = $5
		RETURNING person_id, org_id, name, employee_no, roles, version, source_evidence_id, tenant_id, created_at, updated_at
	`, newName, newEmployeeNo, nullableString(sourceEvidenceID), personID, expectedVersion).Scan(
		&agg.PersonID, &agg.OrgID, &agg.Name, &agg.EmployeeNo, &rolesJSON,
		&agg.Version, &srcEvid, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		var currentVersion int64
		err2 := tx.QueryRowContext(ctx, `SELECT version FROM business.persons WHERE person_id = $1`, personID).Scan(&currentVersion)
		if err2 == sql.ErrNoRows {
			return nil, person.ErrPersonNotFound
		}
		return nil, fmt.Errorf("%w (current version: %d, expected: %d)", person.ErrPersonCASConflict, currentVersion, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update person with CAS: %w", err)
	}
	agg.SourceEvidenceID = srcEvid.String
	if err := json.Unmarshal(rolesJSON, &agg.Roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}
	if agg.Roles == nil {
		agg.Roles = []person.RoleRef{}
	}
	return agg, nil
}

func (r *PersonRepositoryPostgreSQL) AssignRoleWithCAS(
	ctx context.Context, tx *sql.Tx, personID string, roleID string,
	expectedVersion int64, sourceEvidenceID string,
) (*person.PersonAggregate, error) {
	agg := &person.PersonAggregate{}
	var srcEvid sql.NullString
	var rolesJSON []byte

	newRoleRef, _ := json.Marshal(map[string]string{"roleId": roleID})

	err := tx.QueryRowContext(ctx, `
		UPDATE business.persons
		SET roles = roles || jsonb_build_array($1::jsonb), version = version + 1, source_evidence_id = $2, updated_at = now()
		WHERE person_id = $3 AND version = $4
		RETURNING person_id, org_id, name, employee_no, roles, version, source_evidence_id, tenant_id, created_at, updated_at
	`, string(newRoleRef), nullableString(sourceEvidenceID), personID, expectedVersion).Scan(
		&agg.PersonID, &agg.OrgID, &agg.Name, &agg.EmployeeNo, &rolesJSON,
		&agg.Version, &srcEvid, &agg.TenantID, &agg.CreatedAt, &agg.UpdatedAt)
	if err == sql.ErrNoRows {
		var currentVersion int64
		err2 := tx.QueryRowContext(ctx, `SELECT version FROM business.persons WHERE person_id = $1`, personID).Scan(&currentVersion)
		if err2 == sql.ErrNoRows {
			return nil, person.ErrPersonNotFound
		}
		return nil, fmt.Errorf("%w (current version: %d, expected: %d)", person.ErrPersonCASConflict, currentVersion, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to assign role with CAS: %w", err)
	}
	agg.SourceEvidenceID = srcEvid.String
	if err := json.Unmarshal(rolesJSON, &agg.Roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}
	if agg.Roles == nil {
		agg.Roles = []person.RoleRef{}
	}
	return agg, nil
}

func (r *PersonRepositoryPostgreSQL) ExistsByEmployeeNo(
	ctx context.Context, tx *sql.Tx, orgID string, employeeNo string, excludePersonID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM business.persons
			WHERE org_id = $1 AND employee_no = $2
			  AND ($3::uuid IS NULL OR person_id <> $3::uuid)
		)
	`, orgID, employeeNo, nullableString(excludePersonID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check employee_no existence: %w", err)
	}
	return exists, nil
}

func (r *PersonRepositoryPostgreSQL) CheckOrganizationExists(
	ctx context.Context, tx *sql.Tx, orgID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM business.organizations WHERE org_id = $1)
	`, orgID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check organization existence: %w", err)
	}
	return exists, nil
}
