package organization

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SubtreeUpdateResult struct {
	AffectedDescendantIDs []string
	LevelDelta            int
}

type OrganizationTreeCoordinator struct {
	repo            TreeCoordinatorRepository
	evWriter        TreeCoordinatorEvidenceWriter
	structEvWriter  TreeCoordinatorStructuralEvidenceWriter
	outbox          TreeCoordinatorOutbox
}

type TreeCoordinatorRepository interface {
	FindByID(ctx context.Context, tx *sql.Tx, orgID string, forUpdate bool) (*OrganizationAggregate, error)
	GetSubtree(ctx context.Context, tx *sql.Tx, orgID string) ([]SubtreeNode, error)
	UpdateSubtreeLevels(ctx context.Context, tx *sql.Tx, descendantIDs []string, levelDelta int) error
	MoveWithCAS(ctx context.Context, tx *sql.Tx, orgID string, newParentID string, newLevel int, expectedVersion int64, sourceEvidenceID string) (*OrganizationAggregate, error)
}

type SubtreeNode struct {
	OrgID string
	Level int
	Depth int
}

type TreeCoordinatorEvidenceWriter interface {
	WriteMove(ctx context.Context, tx *sql.Tx, agg *OrganizationAggregate, event *OrganizationMovedEvent) (string, error)
}

type TreeCoordinatorStructuralEvidenceWriter interface {
	WriteStructural(ctx context.Context, tx *sql.Tx, orgID string, tenantID string, event *OrganizationMovedEvent, affectedDescendantIDs []string, levelDelta int) (string, error)
}

type TreeCoordinatorOutbox interface {
	Write(ctx context.Context, tx *sql.Tx, aggType string, aggID string, eventType string, tenantID string, corrID string, causID string, payload []byte) error
}

func NewOrganizationTreeCoordinator(
	repo TreeCoordinatorRepository,
	evWriter TreeCoordinatorEvidenceWriter,
	structEvWriter TreeCoordinatorStructuralEvidenceWriter,
	outbox TreeCoordinatorOutbox,
) *OrganizationTreeCoordinator {
	return &OrganizationTreeCoordinator{
		repo:           repo,
		evWriter:       evWriter,
		structEvWriter: structEvWriter,
		outbox:         outbox,
	}
}

func (c *OrganizationTreeCoordinator) MoveSubtree(
	ctx context.Context, tx *sql.Tx, cmd MoveOrganizationCommand,
) (*OrganizationMovedEvent, *SubtreeUpdateResult, error) {
	existing, err := c.repo.FindByID(ctx, tx, cmd.OrgID, true)
	if err != nil {
		return nil, nil, err
	}

	if existing.Version != cmd.ExpectedVersion {
		return nil, nil, fmt.Errorf("%w (current: %d, expected: %d)", ErrOrgVersionConflict, existing.Version, cmd.ExpectedVersion)
	}

	var newParent *OrganizationAggregate
	var newParentLevel int
	if cmd.NewParentID != "" {
		newParent, err = c.repo.FindByID(ctx, tx, cmd.NewParentID, false)
		if err != nil {
			return nil, nil, err
		}
		if newParent.EnterpriseID != existing.EnterpriseID {
			return nil, nil, ErrOrgParentCrossEnterprise
		}
		if err := c.checkNoCycle(ctx, tx, cmd.NewParentID, cmd.OrgID); err != nil {
			return nil, nil, err
		}
		newParentLevel = newParent.Level
	}

	newLevel := newParentLevel + 1
	if newLevel > MaxTreeLevel {
		return nil, nil, ErrOrgLevelExceedMax
	}

	subtree, err := c.repo.GetSubtree(ctx, tx, cmd.OrgID)
	if err != nil {
		return nil, nil, err
	}

	maxDepth := 0
	for _, node := range subtree {
		if node.Depth > maxDepth {
			maxDepth = node.Depth
		}
	}
	if newLevel+maxDepth > MaxTreeLevel {
		return nil, nil, ErrOrgSubtreeLevelExceedMax
	}

	movedAgg, err := c.repo.MoveWithCAS(ctx, tx, cmd.OrgID, cmd.NewParentID, newLevel, cmd.ExpectedVersion, cmd.SourceEvidenceID)
	if err != nil {
		return nil, nil, err
	}

	levelDelta := newLevel - existing.Level
	var descendantIDs []string
	if levelDelta != 0 && len(subtree) > 0 {
		descendantIDs = make([]string, 0, len(subtree))
		for _, node := range subtree {
			descendantIDs = append(descendantIDs, node.OrgID)
		}
		if err := c.repo.UpdateSubtreeLevels(ctx, tx, descendantIDs, levelDelta); err != nil {
			return nil, nil, err
		}
	}

	now := time.Now().UTC()
	event := &OrganizationMovedEvent{
		EventID:          uuid.NewString(),
		EventType:        "OrganizationMoved",
		OrgID:            movedAgg.OrgID,
		EnterpriseID:     movedAgg.EnterpriseID,
		OldParentID:      existing.ParentID,
		NewParentID:      cmd.NewParentID,
		OldLevel:         existing.Level,
		NewLevel:         newLevel,
		Version:          movedAgg.Version,
		SourceEvidenceID: cmd.SourceEvidenceID,
		TenantID:         existing.TenantID,
		Timestamp:        now,
	}

	evID, err := c.evWriter.WriteMove(ctx, tx, movedAgg, event)
	if err != nil {
		return nil, nil, err
	}

	structEvID, err := c.structEvWriter.WriteStructural(ctx, tx, cmd.OrgID, existing.TenantID, event, descendantIDs, levelDelta)
	if err != nil {
		return nil, nil, err
	}

	payload := []byte(fmt.Sprintf(`{"orgId":"%s","oldParentId":"%s","newParentId":"%s","oldLevel":%d,"newLevel":%d,"version":%d,"evidenceRef":"%s","structuralEvidenceRef":"%s"}`,
		event.OrgID, event.OldParentID, event.NewParentID, event.OldLevel, event.NewLevel, event.Version, evID, structEvID))

	if err := c.outbox.Write(ctx, tx, "Organization", event.OrgID, "organization.moved", existing.TenantID, "", event.EventID, payload); err != nil {
		return nil, nil, ErrOutboxWriteFailed
	}

	result := &SubtreeUpdateResult{
		AffectedDescendantIDs: descendantIDs,
		LevelDelta:            levelDelta,
	}

	return event, result, nil
}

func (c *OrganizationTreeCoordinator) checkNoCycle(ctx context.Context, tx *sql.Tx, newParentID string, orgID string) error {
	currentParentID := newParentID
	for i := 0; i < MaxTreeLevel+1; i++ {
		if currentParentID == orgID {
			return ErrOrgCycleDetected
		}
		if currentParentID == "" {
			return nil
		}
		parent, err := c.repo.FindByID(ctx, tx, currentParentID, false)
		if err != nil {
			return ErrOrgParentNotFound
		}
		currentParentID = parent.ParentID
	}
	return nil
}