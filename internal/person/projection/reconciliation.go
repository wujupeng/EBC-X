package projection

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wujupeng/ebcx/internal/platform/graph"
)

type PersonMismatchType string

const (
	PersonMismatchMissingNode  PersonMismatchType = "missing_node"
	PersonMismatchExtraNode    PersonMismatchType = "extra_node"
	PersonMismatchPropertyDiff PersonMismatchType = "property_mismatch"
	PersonMismatchMissingEdge  PersonMismatchType = "missing_edge"
	PersonMismatchExtraEdge    PersonMismatchType = "extra_edge"
)

type PersonMismatch struct {
	Type       PersonMismatchType
	EntityID   string
	Property   string
	PGValue    any
	Neo4jValue any
}

type PersonRepairAction struct {
	Type      PersonMismatchType
	EntityID  string
	Action    string
	Timestamp time.Time
}

type PersonReconciliationEngine struct {
	neo4j *graph.RealNeo4jGraph
	pg    *sql.DB
}

func NewPersonReconciliationEngine(neo4j *graph.RealNeo4jGraph, pg *sql.DB) *PersonReconciliationEngine {
	return &PersonReconciliationEngine{neo4j: neo4j, pg: pg}
}

type pgPerson struct {
	PersonID   string
	OrgID      string
	Name       string
	EmployeeNo string
	Version    int64
}

func (r *PersonReconciliationEngine) DetectMismatches(ctx context.Context, tenantID string) ([]PersonMismatch, error) {
	var mismatches []PersonMismatch

	pgPersons, err := r.queryPGPersons(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PG persons: %w", err)
	}

	neo4jNodes, err := r.queryNeo4jPersonNodes(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query Neo4j nodes: %w", err)
	}

	pgMap := make(map[string]pgPerson)
	for _, p := range pgPersons {
		pgMap[p.PersonID] = p
	}

	neo4jMap := make(map[string]map[string]any)
	for _, n := range neo4jNodes {
		entityID, _ := n["entityId"].(string)
		if entityID != "" {
			neo4jMap[entityID] = n
		}
	}

	for personID, pg := range pgMap {
		neo4j, ok := neo4jMap[personID]
		if !ok {
			mismatches = append(mismatches, PersonMismatch{
				Type:     PersonMismatchMissingNode,
				EntityID: personID,
			})
			continue
		}
		neo4jName, _ := neo4j["name"].(string)
		if neo4jName != pg.Name {
			mismatches = append(mismatches, PersonMismatch{
				Type:       PersonMismatchPropertyDiff,
				EntityID:   personID,
				Property:   "name",
				PGValue:    pg.Name,
				Neo4jValue: neo4jName,
			})
		}
	}

	for entityID := range neo4jMap {
		if _, ok := pgMap[entityID]; !ok {
			mismatches = append(mismatches, PersonMismatch{
				Type:     PersonMismatchExtraNode,
				EntityID: entityID,
			})
		}
	}

	edgeMismatches, err := r.detectEdgeMismatches(ctx, tenantID, pgPersons)
	if err != nil {
		return nil, err
	}
	mismatches = append(mismatches, edgeMismatches...)

	return mismatches, nil
}

func (r *PersonReconciliationEngine) Repair(ctx context.Context, tenantID string) ([]PersonRepairAction, error) {
	mismatches, err := r.DetectMismatches(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var actions []PersonRepairAction
	now := time.Now()

	pgPersons, _ := r.queryPGPersons(ctx, tenantID)
	pgMap := make(map[string]pgPerson)
	for _, p := range pgPersons {
		pgMap[p.PersonID] = p
	}

	for _, m := range mismatches {
		switch m.Type {
		case PersonMismatchMissingNode:
			pg := pgMap[m.EntityID]
			nodeID := fmt.Sprintf("person-%s", m.EntityID)
			node := graph.Node{
				NodeID:           nodeID,
				NodeType:         graph.NodePerson,
				EntityID:         m.EntityID,
				TenantID:         tenantID,
				Version:          pg.Version,
				Status:           "ACTIVE",
				Source:           graph.SourceReplay,
				SourceEvidenceID: "reconciliation-repair",
			}
			if err := r.neo4j.MergeNode(ctx, node); err == nil {
				actions = append(actions, PersonRepairAction{
					Type: PersonMismatchMissingNode, EntityID: m.EntityID, Action: "created_node", Timestamp: now,
				})
			}

		case PersonMismatchExtraNode:
			nodeID := fmt.Sprintf("person-%s", m.EntityID)
			r.neo4j.DeleteNode(ctx, nodeID)
			actions = append(actions, PersonRepairAction{
				Type: PersonMismatchExtraNode, EntityID: m.EntityID, Action: "deleted_node", Timestamp: now,
			})

		case PersonMismatchPropertyDiff:
			pg := pgMap[m.EntityID]
			nodeID := fmt.Sprintf("person-%s", m.EntityID)
			r.neo4j.ExecuteQuery(ctx,
				`MATCH (p:Person {nodeId: $nodeId}) SET p.name = $name`,
				map[string]any{"nodeId": nodeID, "name": pg.Name})
			actions = append(actions, PersonRepairAction{
				Type: PersonMismatchPropertyDiff, EntityID: m.EntityID, Action: "repaired_property", Timestamp: now,
			})

		case PersonMismatchMissingEdge:
			pg := pgMap[m.EntityID]
			edgeID := fmt.Sprintf("belongs-to-%s-%s", pg.PersonID, pg.OrgID)
			edge := graph.Edge{
				EdgeID:           edgeID,
				EdgeType:         graph.EdgeBelongsTo,
				FromNodeID:       fmt.Sprintf("person-%s", pg.PersonID),
				ToNodeID:         fmt.Sprintf("organization-%s", pg.OrgID),
				TenantID:         tenantID,
				Validity:         graph.ValidityActive,
				Version:          1,
				SourceEvent:      "reconciliation-repair",
				SourceEvidenceID: "reconciliation-repair",
			}
			r.neo4j.MergeEdge(ctx, edge)
			actions = append(actions, PersonRepairAction{
				Type: PersonMismatchMissingEdge, EntityID: m.EntityID, Action: "created_edge", Timestamp: now,
			})

		case PersonMismatchExtraEdge:
			r.neo4j.ExecuteQuery(ctx,
				`MATCH ()-[r:EDGE {edgeType: 'BELONGS_TO', tenantId: $tenantId}]-()
				 WHERE r.edgeId CONTAINS $entityId AND r.validity = 'active'
				 SET r.validity = 'inactive'`,
				map[string]any{"tenantId": tenantID, "entityId": m.EntityID})
			actions = append(actions, PersonRepairAction{
				Type: PersonMismatchExtraEdge, EntityID: m.EntityID, Action: "deactivated_edge", Timestamp: now,
			})
		}
	}

	return actions, nil
}

func (r *PersonReconciliationEngine) ReplayEvents(ctx context.Context, events []graph.OutboxEvent, projection *PersonProjection) ([]PersonProjectionResult, error) {
	var results []PersonProjectionResult
	for _, evt := range events {
		result := projection.Consume(ctx, evt)
		results = append(results, PersonProjectionResult{
			EventID: result.EventID,
			Err:     result.Err,
		})
	}
	return results, nil
}

func (r *PersonReconciliationEngine) queryPGPersons(ctx context.Context, tenantID string) ([]pgPerson, error) {
	rows, err := r.pg.QueryContext(ctx, `
		SELECT person_id::text, org_id::text, name, employee_no, version
		FROM business.persons
		WHERE tenant_id = $1::uuid
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var persons []pgPerson
	for rows.Next() {
		var p pgPerson
		if err := rows.Scan(&p.PersonID, &p.OrgID, &p.Name, &p.EmployeeNo, &p.Version); err != nil {
			return nil, err
		}
		persons = append(persons, p)
	}
	return persons, nil
}

func (r *PersonReconciliationEngine) queryNeo4jPersonNodes(ctx context.Context, tenantID string) ([]map[string]any, error) {
	result, err := r.neo4j.ExecuteQuery(ctx, `
		MATCH (p:Person {tenantId: $tenantId})
		RETURN p.entityId as entityId, p.name as name, p.version as version
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

func (r *PersonReconciliationEngine) detectEdgeMismatches(ctx context.Context, tenantID string, pgPersons []pgPerson) ([]PersonMismatch, error) {
	var mismatches []PersonMismatch

	result, err := r.neo4j.ExecuteQuery(ctx, `
		MATCH (from)-[r:EDGE {edgeType: 'BELONGS_TO', tenantId: $tenantId, validity: 'active'}]->(to)
		WHERE from.nodeType IS NOT NULL AND from.nodeType = 'Person'
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
	for _, p := range pgPersons {
		key := fmt.Sprintf("%s->%s", p.PersonID, p.OrgID)
		pgEdges[key] = p.PersonID
	}

	for key, personID := range pgEdges {
		if !neo4jEdges[key] {
			mismatches = append(mismatches, PersonMismatch{
				Type:     PersonMismatchMissingEdge,
				EntityID: personID,
			})
		}
	}

	for key := range neo4jEdges {
		if _, ok := pgEdges[key]; !ok {
			parts := strings.Split(key, "->")
			if len(parts) == 2 {
				mismatches = append(mismatches, PersonMismatch{
					Type:     PersonMismatchExtraEdge,
					EntityID: parts[0],
				})
			}
		}
	}

	return mismatches, nil
}
