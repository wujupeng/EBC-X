package repository

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/person"
)

type PersonRepository interface {
	Insert(ctx context.Context, tx *sql.Tx, agg *person.PersonAggregate) error
	FindByID(ctx context.Context, tx *sql.Tx, personID string, forUpdate bool) (*person.PersonAggregate, error)
	UpdateWithCAS(ctx context.Context, tx *sql.Tx, personID string, newName string, newEmployeeNo string, expectedVersion int64, sourceEvidenceID string) (*person.PersonAggregate, error)
	AssignRoleWithCAS(ctx context.Context, tx *sql.Tx, personID string, roleID string, expectedVersion int64, sourceEvidenceID string) (*person.PersonAggregate, error)
	ExistsByEmployeeNo(ctx context.Context, tx *sql.Tx, orgID string, employeeNo string, excludePersonID string) (bool, error)
	CheckOrganizationExists(ctx context.Context, tx *sql.Tx, orgID string) (bool, error)
}
