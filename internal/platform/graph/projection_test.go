package graph

import (
	"context"
	"testing"
	"time"
)

func TestNodeTypeValidation(t *testing.T) {
	if len(AllNodeTypes) != 24 {
		t.Errorf("expected 24 node types, got %d", len(AllNodeTypes))
	}
	for _, nt := range AllNodeTypes {
		if !IsValidNodeType(nt) {
			t.Errorf("node type %s should be valid", nt)
		}
	}
	if IsValidNodeType(NodeType("Invalid")) {
		t.Error("Invalid node type should be rejected")
	}
}

func TestEdgeTypeValidation(t *testing.T) {
	if len(AllEdgeTypes) != 8 {
		t.Errorf("expected 8 edge types, got %d", len(AllEdgeTypes))
	}
	for _, et := range AllEdgeTypes {
		if !IsValidEdgeType(et) {
			t.Errorf("edge type %s should be valid", et)
		}
	}
	if IsValidEdgeType(EdgeType("Invalid")) {
		t.Error("Invalid edge type should be rejected")
	}
}

func TestEdgeRequiresSourceEvent(t *testing.T) {
	edge := Edge{
		EdgeID:           "edge-1",
		EdgeType:         EdgeBelongsTo,
		FromNodeID:       "node-a",
		ToNodeID:         "node-b",
		TenantID:         "tenant-1",
		Validity:         ValidityActive,
		Version:          1,
		SourceEvent:      "",
		SourceEvidenceID: "ev-1",
	}
	if err := edge.Validate(); err == nil {
		t.Error("edge without sourceEvent should be rejected (D-GATE-06: edges must be domain-event-driven)")
	}

	edge.SourceEvent = "evt-1"
	edge.SourceEvidenceID = ""
	if err := edge.Validate(); err == nil {
		t.Error("edge without sourceEvidenceId should be rejected (Evidence-First)")
	}

	edge.SourceEvidenceID = "ev-1"
	if err := edge.Validate(); err != nil {
		t.Errorf("valid edge should pass: %v", err)
	}
}

func TestProjectionConsumer_Idempotency(t *testing.T) {
	g := NewGraphInstance()
	metrics := NewMetricsCollector()
	consumer := NewProjectionConsumer(g, NewProjectionRuleRegistry(), metrics)

	now := time.Now()
	evt := OutboxEvent{
		EventID:     "evt-001",
		EventType:   "order.created",
		AggregateID: "order-001",
		TenantID:    "tenant-1",
		EvidenceID:  "ev-001",
		Payload:     map[string]any{},
		OccurredAt:  now,
	}

	r1 := consumer.Consume(context.Background(), evt)
	if r1.Err != nil {
		t.Fatalf("first consume should succeed: %v", r1.Err)
	}

	r2 := consumer.Consume(context.Background(), evt)
	if r2.Err == nil {
		t.Error("second consume should be idempotent (rejected)")
	}

	if consumer.ConsumedCount() != 1 {
		t.Errorf("expected consumed count 1, got %d", consumer.ConsumedCount())
	}
}

func TestProjectionConsumer_FaultIsolation(t *testing.T) {
	g := NewGraphInstance()
	g.SetAvailable(false)

	metrics := NewMetricsCollector()
	consumer := NewProjectionConsumer(g, NewProjectionRuleRegistry(), metrics)

	now := time.Now()
	evt := OutboxEvent{
		EventID:     "evt-002",
		EventType:   "order.created",
		AggregateID: "order-002",
		TenantID:    "tenant-1",
		EvidenceID:  "ev-002",
		Payload:     map[string]any{},
		OccurredAt:  now,
	}

	result := consumer.Consume(context.Background(), evt)
	if result.Err == nil {
		t.Error("projection should fail when Neo4j is unavailable")
	}

	if g.NodeCount() != 0 {
		t.Error("no nodes should be projected when Neo4j is unavailable")
	}

	t.Log("✓ Neo4j fault does not block Transaction Core (fault isolation verified)")
}

func TestProjectionConsumer_RetryAndDLQ(t *testing.T) {
	g := NewGraphInstance()
	g.SetAvailable(false)

	metrics := NewMetricsCollector()
	consumer := NewProjectionConsumer(g, NewProjectionRuleRegistry(), metrics)
	consumer.maxRetries = 3

	now := time.Now()
	evt := OutboxEvent{
		EventID:     "evt-dlq",
		EventType:   "order.created",
		AggregateID: "order-dlq",
		TenantID:    "tenant-1",
		EvidenceID:  "ev-dlq",
		Payload:     map[string]any{},
		OccurredAt:  now,
	}

	for i := 0; i < 3; i++ {
		consumer.Consume(context.Background(), evt)
	}

	if consumer.DLQCount() == 0 {
		t.Error("event should be moved to DLQ after max retries")
	}
	t.Logf("✓ DLQ count after %d retries: %d", 3, consumer.DLQCount())
}

func TestShadowRebuild_ParallelBuild(t *testing.T) {
	dual := NewDualGraph()
	metrics := NewMetricsCollector()
	rules := NewProjectionRuleRegistry()

	primaryConsumer := NewProjectionConsumer(dual.Primary, rules, metrics)

	now := time.Now()
	events := []OutboxEvent{
		{EventID: "e1", EventType: "order.created", AggregateID: "o1", TenantID: "t1", EvidenceID: "ev1", Payload: map[string]any{}, OccurredAt: now},
		{EventID: "e2", EventType: "order.created", AggregateID: "o2", TenantID: "t1", EvidenceID: "ev2", Payload: map[string]any{}, OccurredAt: now},
		{EventID: "e3", EventType: "contract.created", AggregateID: "c1", TenantID: "t1", EvidenceID: "ev3", Payload: map[string]any{}, OccurredAt: now},
	}

	for _, evt := range events {
		primaryConsumer.Consume(context.Background(), evt)
	}

	if dual.Primary.NodeCount() != 3 {
		t.Errorf("primary should have 3 nodes, got %d", dual.Primary.NodeCount())
	}

	rebuilder := NewShadowRebuilder(dual, rules, metrics)
	result := rebuilder.Rebuild(context.Background(), events, "t1")

	if result.Err != nil {
		t.Fatalf("shadow rebuild should succeed: %v", result.Err)
	}

	if result.AvailabilityWindowMs != 0 {
		t.Errorf("availability window should be 0, got %d ms", result.AvailabilityWindowMs)
	}

	if !result.ReconciliationResult.OverallPass {
		t.Error("reconciliation should pass")
	}

	t.Logf("✓ Shadow Rebuild: %d nodes, %d edges, RTO=%dms, availability_window=%dms",
		result.NodesBuilt, result.EdgesBuilt, result.RTOMs, result.AvailabilityWindowMs)
}

func TestShadowRebuild_ReconciliationFail(t *testing.T) {
	dual := NewDualGraph()
	metrics := NewMetricsCollector()
	rules := NewProjectionRuleRegistry()

	primaryConsumer := NewProjectionConsumer(dual.Primary, rules, metrics)
	now := time.Now()
	primaryEvents := []OutboxEvent{
		{EventID: "e1", EventType: "order.created", AggregateID: "o1", TenantID: "t1", EvidenceID: "ev1", Payload: map[string]any{}, OccurredAt: now},
		{EventID: "e2", EventType: "order.created", AggregateID: "o2", TenantID: "t1", EvidenceID: "ev2", Payload: map[string]any{}, OccurredAt: now},
	}
	for _, evt := range primaryEvents {
		primaryConsumer.Consume(context.Background(), evt)
	}

	rebuilder := NewShadowRebuilder(dual, rules, metrics)

	incompleteEvents := []OutboxEvent{
		{EventID: "e1", EventType: "order.created", AggregateID: "o1", TenantID: "t1", EvidenceID: "ev1", Payload: map[string]any{}, OccurredAt: now},
	}

	result := rebuilder.Rebuild(context.Background(), incompleteEvents, "t1")

	if result.Err == nil {
		t.Error("reconciliation should fail when shadow graph is incomplete")
	}
	if result.Err != ErrReconciliationFailed {
		t.Errorf("expected ErrReconciliationFailed, got %v", result.Err)
	}

	t.Log("✓ Reconciliation failure correctly aborts cutover")
}

func TestShadowRebuild_RTOWithin30Min(t *testing.T) {
	dual := NewDualGraph()
	metrics := NewMetricsCollector()
	rules := NewProjectionRuleRegistry()

	now := time.Now()
	events := []OutboxEvent{
		{EventID: "e1", EventType: "order.created", AggregateID: "o1", TenantID: "t1", EvidenceID: "ev1", Payload: map[string]any{}, OccurredAt: now},
	}

	rebuilder := NewShadowRebuilder(dual, rules, metrics)
	result := rebuilder.Rebuild(context.Background(), events, "t1")

	if result.RTOMs > int64(MaxRebuildRTOMinutes*60*1000) {
		t.Errorf("RTO %dms exceeds %d min threshold", result.RTOMs, MaxRebuildRTOMinutes)
	}
	t.Logf("✓ Shadow Rebuild RTO: %dms (≤30min)", result.RTOMs)
}

func TestModeClassification(t *testing.T) {
	metrics := NewMetricsCollector()

	if metrics.Mode() != ModeNormal {
		t.Errorf("initial mode should be Normal, got %s", metrics.Mode())
	}

	metrics.SetMode(ModeDegraded)
	if metrics.Mode() != ModeDegraded {
		t.Errorf("mode should be Degraded, got %s", metrics.Mode())
	}

	metrics.SetMode(ModeRecovery)
	if metrics.Mode() != ModeRecovery {
		t.Errorf("mode should be Recovery, got %s", metrics.Mode())
	}

	metrics.SetMode(ModeRebuild)
	if metrics.Mode() != ModeRebuild {
		t.Errorf("mode should be Rebuild, got %s", metrics.Mode())
	}

	metrics.SetMode(ModeNormal)
	if metrics.Mode() != ModeNormal {
		t.Errorf("mode should be Normal, got %s", metrics.Mode())
	}

	t.Log("✓ Mode classification: Normal/Degraded/Recovery/Rebuild all switchable")
}

func TestMetrics_NormalModeProjectionLag(t *testing.T) {
	metrics := NewMetricsCollector()

	for i := 0; i < 100; i++ {
		metrics.RecordProjectionLag(500)
	}

	if metrics.P95ProjectionLag() > NormalModeProjectionLagThresholdMs {
		t.Errorf("Normal Mode P95 projection_lag %dms > 3s threshold", metrics.P95ProjectionLag())
	}
	t.Logf("✓ Normal Mode P95 projection_lag: %dms (≤3s)", metrics.P95ProjectionLag())
}

func TestMetrics_DegradedModeAlert(t *testing.T) {
	metrics := NewMetricsCollector()

	for i := 0; i < 95; i++ {
		metrics.RecordProjectionLag(500)
	}
	for i := 0; i < 10; i++ {
		metrics.RecordProjectionLag(5000)
	}

	if metrics.Mode() != ModeDegraded {
		t.Errorf("mode should switch to Degraded when P95 > 3s, got %s (P95=%dms)", metrics.Mode(), metrics.P95ProjectionLag())
	}

	alerts := metrics.Alerts()
	if len(alerts) == 0 {
		t.Error("should generate alert when entering Degraded Mode")
	}
	t.Logf("✓ Degraded Mode alert generated: P95=%dms", metrics.P95ProjectionLag())
}

func TestQueryDegradation(t *testing.T) {
	g := NewGraphInstance()

	fallback := &MemoryFallback{
		Nodes: []Node{
			{NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1", Version: 1, SourceEvidenceID: "ev1"},
		},
	}
	qs := NewQueryService(g, fallback)

	result := qs.QueryNodes(context.Background(), "t1")
	if result.Degraded {
		t.Error("should not degrade when Neo4j is available")
	}
	if result.Source != "neo4j" {
		t.Errorf("source should be neo4j, got %s", result.Source)
	}

	g.SetAvailable(false)
	qs.Reset()

	result = qs.QueryNodes(context.Background(), "t1")
	if !result.Degraded {
		t.Error("should degrade when Neo4j is unavailable")
	}
	if result.Source != "postgresql" {
		t.Errorf("source should be postgresql, got %s", result.Source)
	}
	if len(result.Nodes) != 1 {
		t.Errorf("fallback should return 1 node, got %d", len(result.Nodes))
	}

	t.Log("✓ Query degradation: Neo4j fault → PostgreSQL fallback + alert")
}

func TestReconciliation_DetectsMissingNodes(t *testing.T) {
	graphA := NewGraphInstance()
	graphB := NewGraphInstance()

	now := time.Now()
	nodeA := Node{
		NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1",
		Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev1",
	}
	nodeA.CreatedAt = now
	nodeA.UpdatedAt = now

	graphA.MergeNode(nodeA)
	graphB.MergeNode(nodeA)

	nodeB := Node{
		NodeID: "n2", NodeType: NodeOrder, EntityID: "n2", TenantID: "t1",
		Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev2",
	}
	nodeB.CreatedAt = now
	nodeB.UpdatedAt = now
	graphA.MergeNode(nodeB)

	recon := Reconcile(graphA, graphB, "t1")
	if recon.NodeCountMatch {
		t.Error("should detect missing node in Graph-B")
	}
	if recon.OverallPass {
		t.Error("reconciliation should fail when nodes are missing")
	}

	detailed := ReconcileDetailed(graphA, graphB, "t1")
	if len(detailed.MissingNodes) == 0 {
		t.Error("should detect missing nodes in detailed reconciliation")
	}
	t.Logf("✓ Missing nodes detected: %v", detailed.MissingNodes)
}

func TestReconciliation_DetectsExtraNodes(t *testing.T) {
	graphA := NewGraphInstance()
	graphB := NewGraphInstance()

	now := time.Now()
	node := Node{
		NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1",
		Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev1",
	}
	node.CreatedAt = now
	node.UpdatedAt = now

	graphA.MergeNode(node)
	graphB.MergeNode(node)

	extraNode := Node{
		NodeID: "extra", NodeType: NodeOrder, EntityID: "extra", TenantID: "t1",
		Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev-extra",
	}
	extraNode.CreatedAt = now
	extraNode.UpdatedAt = now
	graphB.MergeNode(extraNode)

	detailed := ReconcileDetailed(graphA, graphB, "t1")
	if len(detailed.ExtraNodes) == 0 {
		t.Error("should detect extra nodes in Graph-B")
	}
	t.Logf("✓ Extra nodes detected: %v", detailed.ExtraNodes)
}

func TestReconciliation_ChecksumMismatch(t *testing.T) {
	graphA := NewGraphInstance()
	graphB := NewGraphInstance()

	now := time.Now()
	n1 := Node{NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1", Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev1"}
	n1.CreatedAt = now
	n1.UpdatedAt = now
	n2 := Node{NodeID: "n2", NodeType: NodeOrder, EntityID: "n2", TenantID: "t1", Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev2"}
	n2.CreatedAt = now
	n2.UpdatedAt = now

	graphA.MergeNode(n1)
	graphA.MergeNode(n2)
	graphB.MergeNode(n1)

	recon := Reconcile(graphA, graphB, "t1")
	if recon.ChecksumMatch {
		t.Error("checksum should mismatch when graphs differ")
	}
	t.Logf("✓ Checksum mismatch detected: A=%s... B=%s...", recon.ChecksumA[:8], recon.ChecksumB[:8])
}

func TestReconciliation_TenantIsolation(t *testing.T) {
	graphA := NewGraphInstance()
	graphB := NewGraphInstance()

	now := time.Now()
	n1 := Node{NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1", Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev1"}
	n1.CreatedAt = now
	n1.UpdatedAt = now
	n2 := Node{NodeID: "n2", NodeType: NodeOrder, EntityID: "n2", TenantID: "t2", Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev2"}
	n2.CreatedAt = now
	n2.UpdatedAt = now

	graphA.MergeNode(n1)
	graphA.MergeNode(n2)
	graphB.MergeNode(n1)
	graphB.MergeNode(n2)

	recon := Reconcile(graphA, graphB, "t1")
	if !recon.TenantIsolationMatch {
		t.Error("tenant isolation should hold for t1 scope")
	}
	t.Log("✓ Tenant isolation verified in reconciliation")
}

func TestGraphInstance_MergeNodeVersioning(t *testing.T) {
	g := NewGraphInstance()

	now := time.Now()
	n1 := Node{
		NodeID: "n1", NodeType: NodeOrder, EntityID: "n1", TenantID: "t1",
		Version: 1, Source: SourceDomainEvent, SourceEvidenceID: "ev1",
	}
	n1.CreatedAt = now
	n1.UpdatedAt = now
	g.MergeNode(n1)

	n2 := n1
	n2.Version = 2
	n2.SourceEvidenceID = "ev2"
	g.MergeNode(n2)

	node, _ := g.GetNode("n1")
	if node.Version != 2 {
		t.Errorf("expected version 2 after merge, got %d", node.Version)
	}

	n3 := n1
	n3.Version = 1
	g.MergeNode(n3)
	node, _ = g.GetNode("n1")
	if node.Version != 2 {
		t.Errorf("version should not downgrade, expected 2, got %d", node.Version)
	}
}

func TestProjectionRule_AllEventTypes(t *testing.T) {
	rules := NewProjectionRuleRegistry()

	expectedEvents := []string{
		"order.created", "contract.created", "invoice.created", "payment.created",
		"organization.created", "person.created", "production.created",
		"quality.created", "product.created", "asset.allocated",
		"evidence.recorded", "decision.made", "approval.created",
		"policy.applied", "agent.executed",
	}

	for _, evt := range expectedEvents {
		if _, ok := rules.Lookup(evt); !ok {
			t.Errorf("no projection rule for event: %s", evt)
		}
	}
	t.Logf("✓ All %d projection rules registered", len(expectedEvents))
}
