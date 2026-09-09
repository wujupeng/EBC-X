package evidence_adapter

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/organization"
)

type TreeCoordinatorEvidenceWriterAdapter struct {
	writer *OrganizationEvidenceWriterImpl
}

func NewTreeCoordinatorEvidenceWriterAdapter(writer *OrganizationEvidenceWriterImpl) *TreeCoordinatorEvidenceWriterAdapter {
	return &TreeCoordinatorEvidenceWriterAdapter{writer: writer}
}

func (a *TreeCoordinatorEvidenceWriterAdapter) WriteMove(
	ctx context.Context, tx *sql.Tx, agg *organization.OrganizationAggregate, event *organization.OrganizationMovedEvent,
) (string, error) {
	record, err := a.writer.Write(ctx, tx, agg, event, "MoveOrganization")
	if err != nil {
		return "", err
	}
	return record.EvidenceID, nil
}

type TreeCoordinatorStructuralEvidenceWriterAdapter struct {
	writer *OrganizationTreeStructuralEvidenceWriterImpl
}

func NewTreeCoordinatorStructuralEvidenceWriterAdapter(writer *OrganizationTreeStructuralEvidenceWriterImpl) *TreeCoordinatorStructuralEvidenceWriterAdapter {
	return &TreeCoordinatorStructuralEvidenceWriterAdapter{writer: writer}
}

func (a *TreeCoordinatorStructuralEvidenceWriterAdapter) WriteStructural(
	ctx context.Context, tx *sql.Tx, orgID string, tenantID string,
	event *organization.OrganizationMovedEvent, affectedDescendantIDs []string, levelDelta int,
) (string, error) {
	record, err := a.writer.WriteStructural(ctx, tx, orgID, tenantID, event, "SubtreeStructuralUpdate", affectedDescendantIDs, levelDelta)
	if err != nil {
		return "", err
	}
	return record.EvidenceID, nil
}