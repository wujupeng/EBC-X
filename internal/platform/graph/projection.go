package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type ProjectionRule struct {
	NodeType     NodeType
	EdgeType     EdgeType
	FromNodeType NodeType
	ToNodeType   NodeType
}

type ProjectionRuleRegistry struct {
	rules map[string]ProjectionRule
}

func NewProjectionRuleRegistry() *ProjectionRuleRegistry {
	r := &ProjectionRuleRegistry{rules: make(map[string]ProjectionRule)}
	r.registerDefaults()
	return r
}

func (r *ProjectionRuleRegistry) registerDefaults() {
	r.rules["order.created"] = ProjectionRule{NodeOrder, EdgeTradesWith, NodeCustomer, NodeOrder}
	r.rules["contract.created"] = ProjectionRule{NodeContract, EdgeBelongsTo, NodeOrder, NodeContract}
	r.rules["invoice.created"] = ProjectionRule{NodeInvoice, EdgeBelongsTo, NodeContract, NodeInvoice}
	r.rules["payment.created"] = ProjectionRule{NodePayment, EdgeBelongsTo, NodeInvoice, NodePayment}
	r.rules["organization.created"] = ProjectionRule{NodeOrganization, EdgeBelongsTo, NodeEnterprise, NodeOrganization}
	r.rules["person.created"] = ProjectionRule{NodePerson, EdgeBelongsTo, NodeOrganization, NodePerson}
	r.rules["production.created"] = ProjectionRule{NodeProduction, EdgeTradesWith, NodeOrder, NodeProduction}
	r.rules["quality.created"] = ProjectionRule{NodeQuality, EdgeProduces, NodeProduction, NodeQuality}
	r.rules["product.created"] = ProjectionRule{NodeProduct, EdgeProduces, NodeProduction, NodeProduct}
	r.rules["asset.allocated"] = ProjectionRule{NodeAsset, EdgeUsesAsset, NodeAsset, NodeProduction}
	r.rules["evidence.recorded"] = ProjectionRule{NodeEvidence, EdgeEvidencedBy, NodeEvent, NodeEvidence}
	r.rules["decision.made"] = ProjectionRule{NodeDecision, EdgeEvidencedBy, NodeEvidence, NodeDecision}
	r.rules["approval.created"] = ProjectionRule{NodeApproval, EdgeApprovedBy, NodeDecision, NodeApproval}
	r.rules["policy.applied"] = ProjectionRule{NodePolicy, EdgeGovernedBy, NodePolicy, NodeDecision}
	r.rules["agent.executed"] = ProjectionRule{NodeAgent, EdgeEvidencedBy, NodeDecision, NodeAgent}
}

func (r *ProjectionRuleRegistry) Lookup(eventType string) (ProjectionRule, bool) {
	rule, ok := r.rules[eventType]
	return rule, ok
}

func (r *ProjectionRuleRegistry) Register(eventType string, rule ProjectionRule) {
	r.rules[eventType] = rule
}

type OutboxEvent struct {
	EventID     string
	EventType   string
	AggregateID string
	TenantID    string
	EvidenceID  string
	Payload     map[string]any
	OccurredAt  time.Time
}

type ProjectionResult struct {
	EventID         string
	ProjectedAt     time.Time
	ProjectionLagMs int64
	NodeIDs         []string
	EdgeIDs         []string
	Err             error
}

type ProjectionConsumer struct {
	graph      *GraphInstance
	rules      *ProjectionRuleRegistry
	consumed   map[string]bool
	mu         sync.Mutex
	metrics    *MetricsCollector
	dlq        []OutboxEvent
	retryCount map[string]int
	maxRetries int
}

func NewProjectionConsumer(g *GraphInstance, rules *ProjectionRuleRegistry, metrics *MetricsCollector) *ProjectionConsumer {
	return &ProjectionConsumer{
		graph:      g,
		rules:      rules,
		consumed:   make(map[string]bool),
		metrics:    metrics,
		retryCount: make(map[string]int),
		maxRetries: 5,
	}
}

func (c *ProjectionConsumer) Consume(ctx context.Context, evt OutboxEvent) ProjectionResult {
	c.mu.Lock()
	if c.consumed[evt.EventID] {
		c.mu.Unlock()
		return ProjectionResult{
			EventID: evt.EventID,
			Err:     errors.New("event already projected (idempotency)"),
		}
	}
	c.mu.Unlock()

	result := c.project(ctx, evt)

	if result.Err != nil {
		c.mu.Lock()
		c.retryCount[evt.EventID]++
		if c.retryCount[evt.EventID] >= c.maxRetries {
			c.dlq = append(c.dlq, evt)
			if c.metrics != nil {
				c.metrics.IncrementDLQ()
			}
			log.Printf("DLQ: event %s moved to dead letter queue after %d projection retries", evt.EventID, c.retryCount[evt.EventID])
		}
		c.mu.Unlock()
		return result
	}

	c.mu.Lock()
	c.consumed[evt.EventID] = true
	c.mu.Unlock()

	if c.metrics != nil {
		c.metrics.RecordProjectionLag(result.ProjectionLagMs)
	}

	return result
}

func (c *ProjectionConsumer) project(ctx context.Context, evt OutboxEvent) ProjectionResult {
	rule, ok := c.rules.Lookup(evt.EventType)
	if !ok {
		return ProjectionResult{
			EventID: evt.EventID,
			Err:     fmt.Errorf("no projection rule for event type: %s", evt.EventType),
		}
	}

	now := time.Now()
	lagMs := now.Sub(evt.OccurredAt).Milliseconds()

	nodeID := evt.AggregateID
	node := Node{
		NodeID:           nodeID,
		NodeType:         rule.NodeType,
		EntityID:         evt.AggregateID,
		TenantID:         evt.TenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           SourceDomainEvent,
		SourceEvidenceID: evt.EvidenceID,
		EvidenceRefs:     []string{evt.EvidenceID},
		Properties:       evt.Payload,
	}

	if err := c.graph.MergeNode(node); err != nil {
		return ProjectionResult{EventID: evt.EventID, Err: err}
	}

	nodeIDs := []string{nodeID}
	edgeIDs := []string{}

	if fromID, ok := evt.Payload["from_node_id"].(string); ok && fromID != "" {
		if toID, ok := evt.Payload["to_node_id"].(string); ok && toID != "" {
			edgeID := computeEdgeID(evt.EventID, fromID, toID)
			edge := Edge{
				EdgeID:           edgeID,
				EdgeType:         rule.EdgeType,
				FromNodeID:       fromID,
				ToNodeID:         toID,
				TenantID:         evt.TenantID,
				Validity:         ValidityActive,
				Version:          1,
				SourceEvent:      evt.EventID,
				SourceEvidenceID: evt.EvidenceID,
			}
			if err := c.graph.MergeEdge(edge); err == nil {
				edgeIDs = append(edgeIDs, edgeID)
			}
		}
	}

	return ProjectionResult{
		EventID:         evt.EventID,
		ProjectedAt:     now,
		ProjectionLagMs: lagMs,
		NodeIDs:         nodeIDs,
		EdgeIDs:         edgeIDs,
	}
}

func computeEdgeID(eventID, fromID, toID string) string {
	h := sha256.New()
	h.Write([]byte(eventID + "|" + fromID + "|" + toID))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *ProjectionConsumer) ConsumedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.consumed)
}

func (c *ProjectionConsumer) DLQ() []OutboxEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dlq
}

func (c *ProjectionConsumer) DLQCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.dlq)
}

func (c *ProjectionConsumer) IsConsumed(eventID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consumed[eventID]
}
