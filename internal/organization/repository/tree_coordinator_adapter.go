package repository

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/organization"
)

type TreeCoordinatorRepositoryAdapter struct {
	repo *OrganizationRepositoryPostgreSQL
}

func NewTreeCoordinatorRepositoryAdapter(repo *OrganizationRepositoryPostgreSQL) *TreeCoordinatorRepositoryAdapter {
	return &TreeCoordinatorRepositoryAdapter{repo: repo}
}

func (a *TreeCoordinatorRepositoryAdapter) FindByID(ctx context.Context, tx *sql.Tx, orgID string, forUpdate bool) (*organization.OrganizationAggregate, error) {
	return a.repo.FindByID(ctx, tx, orgID, forUpdate)
}

func (a *TreeCoordinatorRepositoryAdapter) GetSubtree(ctx context.Context, tx *sql.Tx, orgID string) ([]organization.SubtreeNode, error) {
	nodes, err := a.repo.GetSubtree(ctx, tx, orgID)
	if err != nil {
		return nil, err
	}
	result := make([]organization.SubtreeNode, len(nodes))
	for i, n := range nodes {
		result[i] = organization.SubtreeNode{OrgID: n.OrgID, Level: n.Level, Depth: n.Depth}
	}
	return result, nil
}

func (a *TreeCoordinatorRepositoryAdapter) UpdateSubtreeLevels(ctx context.Context, tx *sql.Tx, descendantIDs []string, levelDelta int) error {
	return a.repo.UpdateSubtreeLevels(ctx, tx, descendantIDs, levelDelta)
}

func (a *TreeCoordinatorRepositoryAdapter) MoveWithCAS(ctx context.Context, tx *sql.Tx, orgID string, newParentID string, newLevel int, expectedVersion int64, sourceEvidenceID string) (*organization.OrganizationAggregate, error) {
	return a.repo.MoveWithCAS(ctx, tx, orgID, newParentID, newLevel, expectedVersion, sourceEvidenceID)
}