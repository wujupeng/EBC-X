package projection

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/wujupeng/ebcx/internal/platform/graph"
)

type PersonProjection struct {
	neo4j    *graph.RealNeo4jGraph
	pg       *sql.DB
	consumed map[string]bool
	mu       sync.Mutex
}

func NewPersonProjection(neo4j *graph.RealNeo4jGraph, pg *sql.DB) *PersonProjection {
	return &PersonProjection{
		neo4j:    neo4j,
		pg:       pg,
		consumed: make(map[string]bool),
	}
}

type PersonProjectionResult struct {
	EventID     string
	ProjectedAt time.Time
	Err         error
}

func (p *PersonProjection) Consume(ctx context.Context, evt graph.OutboxEvent) PersonProjectionResult {
	p.mu.Lock()
	if p.consumed[evt.EventID] {
		p.mu.Unlock()
		return PersonProjectionResult{
			EventID: evt.EventID,
			Err:     fmt.Errorf("event already projected (idempotency)"),
		}
	}
	p.mu.Unlock()

	var err error
	switch evt.EventType {
	case "person.created":
		err = p.projectCreated(ctx, evt)
	case "person.updated":
		err = p.projectUpdated(ctx, evt)
	case "role.assigned":
		err = p.projectRoleAssigned(ctx, evt)
	default:
		err = fmt.Errorf("unsupported person event type: %s", evt.EventType)
	}

	if err != nil {
		return PersonProjectionResult{EventID: evt.EventID, Err: err}
	}

	p.mu.Lock()
	p.consumed[evt.EventID] = true
	p.mu.Unlock()

	return PersonProjectionResult{
		EventID:     evt.EventID,
		ProjectedAt: time.Now(),
	}
}

func (p *PersonProjection) projectCreated(ctx context.Context, evt graph.OutboxEvent) error {
	personID, _ := evt.Payload["personId"].(string)
	orgID, _ := evt.Payload["orgId"].(string)
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
	version := getInt64FromPayload(evt.Payload, "version")
	tenantID := evt.TenantID

	nodeID := fmt.Sprintf("person-%s", personID)
	node := graph.Node{
		NodeID:           nodeID,
		NodeType:         graph.NodePerson,
		EntityID:         personID,
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

	edgeID := fmt.Sprintf("belongs-to-%s-%s", personID, orgID)
	edge := graph.Edge{
		EdgeID:           edgeID,
		EdgeType:         graph.EdgeBelongsTo,
		FromNodeID:       nodeID,
		ToNodeID:         fmt.Sprintf("organization-%s", orgID),
		TenantID:         tenantID,
		Validity:         graph.ValidityActive,
		Version:          1,
		SourceEvent:      evt.EventID,
		SourceEvidenceID: evidenceRef,
	}
	return p.neo4j.MergeEdge(ctx, edge)
}

func (p *PersonProjection) projectUpdated(ctx context.Context, evt graph.OutboxEvent) error {
	personID, _ := evt.Payload["personId"].(string)
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
	version := getInt64FromPayload(evt.Payload, "version")
	tenantID := evt.TenantID

	nodeID := fmt.Sprintf("person-%s", personID)
	node := graph.Node{
		NodeID:           nodeID,
		NodeType:         graph.NodePerson,
		EntityID:         personID,
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

func (p *PersonProjection) projectRoleAssigned(ctx context.Context, evt graph.OutboxEvent) error {
	personID, _ := evt.Payload["personId"].(string)
	roleID, _ := evt.Payload["roleId"].(string)
	evidenceRef, _ := evt.Payload["evidenceRef"].(string)
	version := getInt64FromPayload(evt.Payload, "version")

	nodeID := fmt.Sprintf("person-%s", personID)

	_, err := p.neo4j.ExecuteQuery(ctx,
		`MATCH (p:Person {nodeId: $nodeId})
		 SET p.roles = coalesce(p.roles, []) + [$roleId],
		     p.version = $version,
		     p.sourceEvidenceId = $evidenceRef`,
		map[string]any{
			"nodeId":      nodeID,
			"roleId":      roleID,
			"version":     version,
			"evidenceRef": evidenceRef,
		})
	return err
}

func (p *PersonProjection) ConsumedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.consumed)
}

func (p *PersonProjection) IsConsumed(eventID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.consumed[eventID]
}

func getInt64FromPayload(payload map[string]any, key string) int64 {
	switch v := payload[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}
