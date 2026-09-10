package projection

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/wujupeng/ebcx/internal/platform/graph"
)

type OrganizationProjection struct {
	neo4j    *graph.RealNeo4jGraph
	pg       *sql.DB
	moveProj *MoveProjection
	consumed map[string]bool
	mu       sync.Mutex
}

func NewOrganizationProjection(neo4j *graph.RealNeo4jGraph, pg *sql.DB) *OrganizationProjection {
	return &OrganizationProjection{
		neo4j:    neo4j,
		pg:       pg,
		moveProj: NewMoveProjection(neo4j, pg),
		consumed: make(map[string]bool),
	}
}

type OrgProjectionResult struct {
	EventID     string
	ProjectedAt time.Time
	Err         error
}

func (p *OrganizationProjection) Consume(ctx context.Context, evt graph.OutboxEvent) OrgProjectionResult {
	p.mu.Lock()
	if p.consumed[evt.EventID] {
		p.mu.Unlock()
		return OrgProjectionResult{
			EventID: evt.EventID,
			Err:     fmt.Errorf("event already projected (idempotency)"),
		}
	}
	p.mu.Unlock()

	var err error
	switch evt.EventType {
	case "organization.created":
		err = p.projectCreated(ctx, evt)
	case "organization.updated":
		err = p.projectUpdated(ctx, evt)
	case "organization.moved":
		_, err = p.moveProj.ProjectMove(ctx, evt)
	default:
		err = fmt.Errorf("unsupported organization event type: %s", evt.EventType)
	}

	if err != nil {
		return OrgProjectionResult{EventID: evt.EventID, Err: err}
	}

	p.mu.Lock()
	p.consumed[evt.EventID] = true
	p.mu.Unlock()

	return OrgProjectionResult{
		EventID:     evt.EventID,
		ProjectedAt: time.Now(),
	}
}

func (p *OrganizationProjection) projectCreated(ctx context.Context, evt graph.OutboxEvent) error {
	orgID, _ := evt.Payload["orgId"].(string)
	parentID, _ := evt.Payload["parentId"].(string)
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
	level := getIntFromPayload(evt.Payload, "level")
	version := getInt64FromPayload(evt.Payload, "version")
	tenantID := evt.TenantID

	nodeID := fmt.Sprintf("organization-%s", orgID)
	node := graph.Node{
		NodeID:           nodeID,
		NodeType:         graph.NodeOrganization,
		EntityID:         orgID,
		TenantID:         tenantID,
		Version:          version,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: evidenceRef,
		EvidenceRefs:     []string{evidenceRef},
		Properties:       evt.Payload,
	}
	if err := p.neo4j.MergeNode(ctx, node); err != nil {
		return err
	}

	_, err := p.neo4j.ExecuteQuery(ctx,
		`MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
		map[string]any{"nodeId": nodeID, "level": level})
	if err != nil {
		return err
	}

	if parentID != "" {
		edgeID := fmt.Sprintf("haschild-%s-%s", parentID, orgID)
		edge := graph.Edge{
			EdgeID:           edgeID,
			EdgeType:         graph.EdgeHasChild,
			FromNodeID:       fmt.Sprintf("organization-%s", parentID),
			ToNodeID:         nodeID,
			TenantID:         tenantID,
			Validity:         graph.ValidityActive,
			Version:          1,
			SourceEvent:      evt.EventID,
			SourceEvidenceID: evidenceRef,
		}
		if err := p.neo4j.MergeEdge(ctx, edge); err != nil {
			return err
		}
	}

	return nil
}

func (p *OrganizationProjection) projectUpdated(ctx context.Context, evt graph.OutboxEvent) error {
	orgID, _ := evt.Payload["orgId"].(string)
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
	version := getInt64FromPayload(evt.Payload, "version")
	tenantID := evt.TenantID

	nodeID := fmt.Sprintf("organization-%s", orgID)
	node := graph.Node{
		NodeID:           nodeID,
		NodeType:         graph.NodeOrganization,
		EntityID:         orgID,
		TenantID:         tenantID,
		Version:          version,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: evidenceRef,
		EvidenceRefs:     []string{evidenceRef},
		Properties:       evt.Payload,
	}
	return p.neo4j.MergeNode(ctx, node)
}

func (p *OrganizationProjection) ConsumedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.consumed)
}

func (p *OrganizationProjection) IsConsumed(eventID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.consumed[eventID]
}