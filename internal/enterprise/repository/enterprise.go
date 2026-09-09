package repository

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/enterprise"
)

type EnterpriseRepository interface {
	Insert(ctx context.Context, tx *sql.Tx, agg *enterprise.EnterpriseAggregate) error
	FindByID(ctx context.Context, tx *sql.Tx, enterpriseID string) (*enterprise.EnterpriseAggregate, error)
	UpdateWithCAS(ctx context.Context, tx *sql.Tx, enterpriseID string, newName string, expectedVersion int64, sourceEvidenceID string) (*enterprise.EnterpriseAggregate, error)
	ExistsByName(ctx context.Context, tx *sql.Tx, tenantID, name string, excludeID string) (bool, error)
}
