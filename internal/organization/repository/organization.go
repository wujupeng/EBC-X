package repository

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/organization"
)

type SubtreeNode struct {
	OrgID string
	Level int
	Depth int
}

type OrganizationRepository interface {
	Insert(ctx context.Context, tx *sql.Tx, agg *organization.OrganizationAggregate) error
	FindByID(ctx context.Context, tx *sql.Tx, orgID string, forUpdate bool) (*organization.OrganizationAggregate, error)
	UpdateWithCAS(ctx context.Context, tx *sql.Tx, orgID string, newName string, newCode string, expectedVersion int64, sourceEvidenceID string) (*organization.OrganizationAggregate, error)
	ExistsByCode(ctx context.Context, tx *sql.Tx, enterpriseID string, parentID string, code string, excludeOrgID string) (bool, error)
	GetSubtree(ctx context.Context, tx *sql.Tx, orgID string) ([]SubtreeNode, error)
	UpdateSubtreeLevels(ctx context.Context, tx *sql.Tx, descendantIDs []string, levelDelta int) error
	CheckEnterpriseExists(ctx context.Context, tx *sql.Tx, enterpriseID string) (bool, error)
	MoveWithCAS(ctx context.Context, tx *sql.Tx, orgID string, newParentID string, newLevel int, expectedVersion int64, sourceEvidenceID string) (*organization.OrganizationAggregate, error)
}
