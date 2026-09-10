package projection

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wujupeng/ebcx/internal/platform/graph"
)

type MismatchType string

const (
	MismatchMissingNode  MismatchType = "missing_node"
	MismatchExtraNode    MismatchType = "extra_node"
	MismatchPropertyDiff MismatchType = "property_mismatch"
	MismatchMissingEdge  MismatchType = "missing_edge"
	MismatchExtraEdge    MismatchType = "extra_edge"
)

type Mismatch struct {
	Type       MismatchType
	EntityID   string
	Property   string
	PGValue    any
	Neo4jValue any
}

type RepairAction struct {
	Type      MismatchType
	EntityID  string
	Action    string
	Timestamp time.Time
}

type ReconciliationEngine struct {
	neo4j *graph.RealNeo4jGraph
	pg    *sql.DB
}

func NewReconciliationEngine(neo4j *graph.RealNeo4jGraph, pg *sql.DB) *ReconciliationEngine {
	return &ReconciliationEngine{neo4j: neo4j, pg: pg}
}

type pgOrg struct {
	OrgID    string
	ParentID *string
	Level    int
	Name     string
	Version  int64
}

func (r *ReconciliationEngine) DetectMismatches(ctx context.Context, tenantID string) ([]Mismatch, error) {
	var mismatches []Mismatch

	pgOrgs, err := r.queryPGOrgs(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PG orgs: %w", err)
	}

	neo4jNodes, err := r.queryNeo4jOrgNodes(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query Neo4j nodes: %w", err)
	}

	pgMap := make(map[string]pgOrg)
	for _, o := range pgOrgs {
		pgMap[o.OrgID] = o
	}

	neo4jMap := make(map[string]map[string]any)
	for _, n := range neo4jNodes {
		entityID, _ := n["entityId"].(string)
		if entityID != "" {
			neo4jMap[entityID] = n
		}
	}

	for orgID, pg := range pgMap {
		neo4j, ok := neo4jMap[orgID]
		if !ok {
			mismatches = append(mismatches, Mismatch{
				Type:     MismatchMissingNode,
				EntityID: orgID,
			})
			continue
		}
		neo4jLevel, _ := neo4j["level"].(int64)
		if int(neo4jLevel) != pg.Level {
			mismatches = append(mismatches, Mismatch{
				Type:       MismatchPropertyDiff,
				EntityID:   orgID,
				Property:   "level",
				PGValue:    pg.Level,
				Neo4jValue: int(neo4jLevel),
			})
		}
	}

	for entityID := range neo4jMap {
		if _, ok := pgMap[entityID]; !ok {
			mismatches = append(mismatches, Mismatch{
				Type:     MismatchExtraNode,
				EntityID: entityID,
			})
		}
	}

	edgeMismatches, err := r.detectEdgeMismatches(ctx, tenantID, pgOrgs)
	if err != nil {
		return nil, err
	}
	mismatches = append(mismatches, edgeMismatches...)

	return mismatches, nil
}

func (r *ReconciliationEngine) Repair(ctx context.Context, tenantID string) ([]RepairAction, error) {
	mismatches, err := r.DetectMismatches(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var actions []RepairAction
	now := time.Now()

	pgOrgs, _ := r.queryPGOrgs(ctx, tenantID)
	pgMap := make(map[string]pgOrg)
	for _, o := range pgOrgs {
		pgMap[o.OrgID] = o
	}

	for _, m := range mismatches {
		switch m.Type {
		case MismatchMissingNode:
			pg := pgMap[m.EntityID]
			nodeID := fmt.Sprintf("organization-%s", m.EntityID)
			node := graph.Node{
				NodeID:           nodeID,
				NodeType:         graph.NodeOrganization,
				EntityID:         m.EntityID,
				TenantID:         tenantID,
				Version:          pg.Version,
				Status:           "ACTIVE",
				Source:           graph.SourceReplay,
				SourceEvidenceID: "reconciliation-repair",
			}
			if err := r.neo4j.MergeNode(ctx, node); err == nil {
				r.neo4j.ExecuteQuery(ctx,
					`MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
					map[string]any{"nodeId": nodeID, "level": pg.Level})
				actions = append(actions, RepairAction{
					Type: MismatchMissingNode, EntityID: m.EntityID, Action: "created_node", Timestamp: now,
				})
			}

		case MismatchExtraNode:
			nodeID := fmt.Sprintf("organization-%s", m.EntityID)
			r.neo4j.DeleteNode(ctx, nodeID)
			actions = append(actions, RepairAction{
				Type: MismatchExtraNode, EntityID: m.EntityID, Action: "deleted_node", Timestamp: now,
			})

		case MismatchPropertyDiff:
			pg := pgMap[m.EntityID]
			nodeID := fmt.Sprintf("organization-%s", m.EntityID)
			r.neo4j.ExecuteQuery(ctx,
				`MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
				map[string]any{"nodeId": nodeID, "level": pg.Level})
			actions = append(actions, RepairAction{
				Type: MismatchPropertyDiff, EntityID: m.EntityID, Action: "repaired_level", Timestamp: now,
			})

		case MismatchMissingEdge:
			pg := pgMap[m.EntityID]
			if pg.ParentID != nil {
				parentID := *pg.ParentID
				edgeID := fmt.Sprintf("haschild-%s-%s", parentID, m.EntityID)
				edge := graph.Edge{
					EdgeID:           edgeID,
					EdgeType:         graph.EdgeHasChild,
					FromNodeID:       fmt.Sprintf("organization-%s", parentID),
					ToNodeID:         fmt.Sprintf("organization-%s", m.EntityID),
					TenantID:         tenantID,
					Validity:         graph.ValidityActive,
					Version:          1,
					SourceEvent:      "reconciliation-repair",
					SourceEvidenceID: "reconciliation-repair",
				}
				r.neo4j.MergeEdge(ctx, edge)
				actions = append(actions, RepairAction{
					Type: MismatchMissingEdge, EntityID: m.EntityID, Action: "created_edge", Timestamp: now,
				})
			}

		case MismatchExtraEdge:
			r.neo4j.ExecuteQuery(ctx,
				`MATCH ()-[r:EDGE {edgeType: 'HAS_CHILD', tenantId: $tenantId}]->()
				 WHERE r.edgeId CONTAINS $entityId AND r.validity = 'active'
				 SET r.validity = 'inactive'`,
				map[string]any{"tenantId": tenantID, "entityId": m.EntityID})
			actions = append(actions, RepairAction{
				Type: MismatchExtraEdge, EntityID: m.EntityID, Action: "deactivated_edge", Timestamp: now,
			})
		}
	}

	return actions, nil
}

func (r *ReconciliationEngine) queryPGOrgs(ctx context.Context, tenantID string) ([]pgOrg, error) {
	rows, err := r.pg.QueryContext(ctx, `
		SELECT org_id::text, parent_id::text, level, name, version
		FROM business.organizations
		WHERE tenant_id = $1::uuid
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []pgOrg
	for rows.Next() {
		var o pgOrg
		var parentID sql.NullString
		if err := rows.Scan(&o.OrgID, &parentID, &o.Level, &o.Name, &o.Version); err != nil {
			return nil, err
		}
		if parentID.Valid {
			s := parentID.String
			o.ParentID = &s
		}
		orgs = append(orgs, o)
	}
	return orgs, nil
}

func (r *ReconciliationEngine) queryNeo4jOrgNodes(ctx context.Context, tenantID string) ([]map[string]any, error) {
	result, err := r.neo4j.ExecuteQuery(ctx, `
		MATCH (n:Organization {tenantId: $tenantId})
		RETURN n.entityId as entityId, n.level as level, n.version as version
	`, map[string]any{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}

	nodes := make([]map[string]any, 0, len(result.Records))
	for _, rec := range result.Records {
		props := make(map[string]any)
		for _, key := range rec.Keys {
			val, _ := rec.Get(key)
			props[key] = val
		}
		nodes = append(nodes, props)
	}
	return nodes, nil
}

func (r *ReconciliationEngine) detectEdgeMismatches(ctx context.Context, tenantID string, pgOrgs []pgOrg) ([]Mismatch, error) {
	var mismatches []Mismatch

	result, err := r.neo4j.ExecuteQuery(ctx, `
		MATCH (from)-[r:EDGE {edgeType: 'HAS_CHILD', tenantId: $tenantId, validity: 'active'}]->(to)
		RETURN from.entityId as fromEntity, to.entityId as toEntity
	`, map[string]any{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}

	neo4jEdges := make(map[string]bool)
	for _, rec := range result.Records {
		fromEntity, _ := rec.Get("fromEntity")
		toEntity, _ := rec.Get("toEntity")
		key := fmt.Sprintf("%v->%v", fromEntity, toEntity)
		neo4jEdges[key] = true
	}

	pgEdges := make(map[string]string)
	for _, o := range pgOrgs {
		if o.ParentID != nil {
			key := fmt.Sprintf("%s->%s", *o.ParentID, o.OrgID)
			pgEdges[key] = o.OrgID
		}
	}

	for key, childID := range pgEdges {
		if !neo4jEdges[key] {
			mismatches = append(mismatches, Mismatch{
				Type:     MismatchMissingEdge,
				EntityID: childID,
			})
		}
	}

	for key := range neo4jEdges {
		if _, ok := pgEdges[key]; !ok {
			parts := strings.Split(key, "->")
			if len(parts) == 2 {
				mismatches = append(mismatches, Mismatch{
					Type:     MismatchExtraEdge,
					EntityID: parts[1],
				})
			}
		}
	}

	return mismatches, nil
}