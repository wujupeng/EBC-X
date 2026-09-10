package projection

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wujupeng/ebcx/internal/platform/graph"
)

type MoveProjection struct {
	neo4j *graph.RealNeo4jGraph
	pg    *sql.DB
}

func NewMoveProjection(neo4j *graph.RealNeo4jGraph, pg *sql.DB) *MoveProjection {
	return &MoveProjection{neo4j: neo4j, pg: pg}
}

type MoveProjectionResult struct {
	MovedNodeID        string
	UpdatedDescendants []string
	OldEdgeID          string
	NewEdgeID          string
}

func (p *MoveProjection) ProjectMove(ctx context.Context, evt graph.OutboxEvent) (*MoveProjectionResult, error) {
	orgID, _ := evt.Payload["orgId"].(string)
	oldParentID, _ := evt.Payload["oldParentId"].(string)
	newParentID, _ := evt.Payload["newParentId"].(string)
	newLevel := getIntFromPayload(evt.Payload, "newLevel")
	version := getInt64FromPayload(evt.Payload, "version")
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
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
		return nil, fmt.Errorf("failed to merge moved node: %w", err)
	}

	_, err := p.neo4j.ExecuteQuery(ctx,
		`MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
		map[string]any{"nodeId": nodeID, "level": newLevel})
	if err != nil {
		return nil, fmt.Errorf("failed to set moved node level: %w", err)
	}

	descendants, err := p.querySubtreeDescendants(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subtree descendants: %w", err)
	}

	for _, d := range descendants {
		descNodeID := fmt.Sprintf("organization-%s", d.OrgID)
		_, err := p.neo4j.ExecuteQuery(ctx,
			`MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
			map[string]any{"nodeId": descNodeID, "level": d.Level})
		if err != nil {
			return nil, fmt.Errorf("failed to update descendant %s level: %w", d.OrgID, err)
		}
	}

	var oldEdgeID string
	if oldParentID != "" {
		oldEdgeID = fmt.Sprintf("haschild-%s-%s", oldParentID, orgID)
		oldEdge := graph.Edge{
			EdgeID:           oldEdgeID,
			EdgeType:         graph.EdgeHasChild,
			FromNodeID:       fmt.Sprintf("organization-%s", oldParentID),
			ToNodeID:         nodeID,
			TenantID:         tenantID,
			Validity:         graph.ValidityInactive,
			Version:          version,
			SourceEvent:      evt.EventID,
			SourceEvidenceID: evidenceRef,
		}
		if err := p.neo4j.MergeEdge(ctx, oldEdge); err != nil {
			return nil, fmt.Errorf("failed to deactivate old HAS_CHILD edge: %w", err)
		}
	}

	var newEdgeID string
	if newParentID != "" {
		newEdgeID = fmt.Sprintf("haschild-%s-%s", newParentID, orgID)
		newEdge := graph.Edge{
			EdgeID:           newEdgeID,
			EdgeType:         graph.EdgeHasChild,
			FromNodeID:       fmt.Sprintf("organization-%s", newParentID),
			ToNodeID:         nodeID,
			TenantID:         tenantID,
			Validity:         graph.ValidityActive,
			Version:          version,
			SourceEvent:      evt.EventID,
			SourceEvidenceID: evidenceRef,
		}
		if err := p.neo4j.MergeEdge(ctx, newEdge); err != nil {
			return nil, fmt.Errorf("failed to create new HAS_CHILD edge: %w", err)
		}
	}

	descendantIDs := make([]string, len(descendants))
	for i, d := range descendants {
		descendantIDs[i] = d.OrgID
	}

	return &MoveProjectionResult{
		MovedNodeID:        nodeID,
		UpdatedDescendants: descendantIDs,
		OldEdgeID:          oldEdgeID,
		NewEdgeID:          newEdgeID,
	}, nil
}

type subtreeDescendant struct {
	OrgID string
	Level int
}

func (p *MoveProjection) querySubtreeDescendants(ctx context.Context, orgID string) ([]subtreeDescendant, error) {
	rows, err := p.pg.QueryContext(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT org_id, level, 0 AS depth
			FROM business.organizations
			WHERE org_id = $1::uuid
			UNION ALL
			SELECT o.org_id, o.level, s.depth + 1
			FROM business.organizations o
			JOIN subtree s ON o.parent_id = s.org_id
		)
		SELECT org_id::text, level FROM subtree WHERE depth > 0 ORDER BY depth
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descendants []subtreeDescendant
	for rows.Next() {
		var d subtreeDescendant
		if err := rows.Scan(&d.OrgID, &d.Level); err != nil {
			return nil, err
		}
		descendants = append(descendants, d)
	}
	return descendants, nil
}

func getIntFromPayload(payload map[string]any, key string) int {
	switch v := payload[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func getInt64FromPayload(payload map[string]any, key string) int64 {
	switch v := payload[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}