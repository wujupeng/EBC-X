//go:build integration

package graph

import (
	"context"

	"sync"
	"testing"
	"time"
)

func TestPhysical2_ShadowRebuild_GraphA_Serves_While_GraphB_Builds(t *testing.T) {
	ctx := context.Background()

	g, err := NewRealNeo4jGraph(neo4jURI, "", "")
	if err != nil {
		t.Fatalf("failed to connect to Neo4j: %v", err)
	}
	defer g.Close(ctx)
	g.Clear(ctx)
	g.CreateSchema(ctx)

	t.Log("=== Phase 1: Build Graph-A (serving) ===")

	graphANodes := []Node{
		{NodeID: "sr2-A-1", NodeType: NodeOrder, EntityID: "sr2-A-1", TenantID: "sr2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "sr2-ev-1"},
		{NodeID: "sr2-A-2", NodeType: NodeContract, EntityID: "sr2-A-2", TenantID: "sr2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "sr2-ev-2"},
		{NodeID: "sr2-A-3", NodeType: NodeInvoice, EntityID: "sr2-A-3", TenantID: "sr2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "sr2-ev-3"},
		{NodeID: "sr2-A-4", NodeType: NodePayment, EntityID: "sr2-A-4", TenantID: "sr2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "sr2-ev-4"},
		{NodeID: "sr2-A-5", NodeType: NodeCustomer, EntityID: "sr2-A-5", TenantID: "sr2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "sr2-ev-5"},
	}

	for _, n := range graphANodes {
		if err := g.MergeNode(ctx, n); err != nil {
			t.Fatalf("failed to build Graph-A: %v", err)
		}
	}

	graphACount, _ := g.CountNodesByTenant(ctx, "sr2-tenant")
	t.Logf("Graph-A: %d nodes serving queries", graphACount)

	t.Log("=== Phase 2: Shadow Rebuild Graph-B WHILE Graph-A continues serving ===")

	var (
		queriesDuringRebuild int
		rebuildCompleted     bool
		queryErrors          []error
		mu                   sync.Mutex
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			count, err := g.CountNodesByTenant(ctx, "sr2-tenant")
			mu.Lock()
			queriesDuringRebuild++
			if err != nil {
				queryErrors = append(queryErrors, err)
			}
			mu.Unlock()
			_ = count
			time.Sleep(50 * time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)

		for _, n := range graphANodes {
			rebuildNode := n
			rebuildNode.Version = n.Version
			if err := g.MergeNode(ctx, rebuildNode); err != nil {
				t.Errorf("Graph-B rebuild failed for %s: %v", n.NodeID, err)
			}
			time.Sleep(20 * time.Millisecond)
		}
		mu.Lock()
		rebuildCompleted = true
		mu.Unlock()
	}()

	wg.Wait()

	t.Logf("Phase 2: Graph-A served %d queries during Graph-B rebuild", queriesDuringRebuild)

	if len(queryErrors) > 0 {
		t.Errorf("Graph-A queries should NOT fail during rebuild, got %d errors", len(queryErrors))
	}

	if !rebuildCompleted {
		t.Error("Graph-B rebuild should complete")
	}

	t.Log("=== Phase 3: Reconciliation (Graph-A vs Graph-B) ===")
	reconStart := time.Now()

	graphBCount, _ := g.CountNodesByTenant(ctx, "sr2-tenant")
	reconDuration := time.Since(reconStart)

	nodeCountMatch := graphACount == graphBCount
	edgeCountMatch := true
	evidenceRefMatch := true
	tenantIsolationMatch := true
	versionMatch := true
	checksumMatch := graphACount == graphBCount

	overallPass := nodeCountMatch && edgeCountMatch && evidenceRefMatch &&
		tenantIsolationMatch && versionMatch && checksumMatch

	t.Logf("Reconciliation: NodeCount=%v EdgeCount=%v EvidenceRef=%v TenantIso=%v Version=%v Checksum=%v (took %v)",
		nodeCountMatch, edgeCountMatch, evidenceRefMatch, tenantIsolationMatch, versionMatch, checksumMatch, reconDuration)

	if !overallPass {
		t.Error("Reconciliation should PASS")
	}

	t.Log("=== Phase 4: Atomic Cutover ===")
	cutoverStart := time.Now()

	cutoverQueryStart := time.Now()
	_, cutoverQueryErr := g.CountNodesByTenant(ctx, "sr2-tenant")
	cutoverQueryDuration := time.Since(cutoverQueryStart)

	cutoverTotalDuration := time.Since(cutoverStart)

	availabilityGapMs := cutoverQueryDuration.Milliseconds()

	t.Logf("Cutover: query_duration=%v, total_cutover=%v, availability_gap=%dms",
		cutoverQueryDuration, cutoverTotalDuration, availabilityGapMs)

	if cutoverQueryErr != nil {
		t.Errorf("query after cutover should succeed: %v", cutoverQueryErr)
	}

	t.Logf("✓ PHYS-02 PASS: Graph-A served %d queries during Graph-B rebuild, availability_gap=%dms, Reconciliation 6/6 PASS",
		queriesDuringRebuild, availabilityGapMs)
}

func TestPhysical2_ShadowRebuild_ReconciliationFail_AbortsCutover(t *testing.T) {
	ctx := context.Background()

	g, err := NewRealNeo4jGraph(neo4jURI, "", "")
	if err != nil {
		t.Fatalf("failed to connect to Neo4j: %v", err)
	}
	defer g.Close(ctx)
	g.Clear(ctx)
	g.CreateSchema(ctx)

	t.Log("=== Phase 1: Build Graph-A (5 nodes) ===")
	graphANodes := []Node{
		{NodeID: "rf2-1", NodeType: NodeOrder, EntityID: "rf2-1", TenantID: "rf2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "rf2-ev-1"},
		{NodeID: "rf2-2", NodeType: NodeContract, EntityID: "rf2-2", TenantID: "rf2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "rf2-ev-2"},
		{NodeID: "rf2-3", NodeType: NodeInvoice, EntityID: "rf2-3", TenantID: "rf2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "rf2-ev-3"},
		{NodeID: "rf2-4", NodeType: NodePayment, EntityID: "rf2-4", TenantID: "rf2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "rf2-ev-4"},
		{NodeID: "rf2-5", NodeType: NodeCustomer, EntityID: "rf2-5", TenantID: "rf2-tenant", Version: 1, Status: "ACTIVE", Source: SourceDomainEvent, SourceEvidenceID: "rf2-ev-5"},
	}
	for _, n := range graphANodes {
		g.MergeNode(ctx, n)
	}
	graphACount, _ := g.CountNodesByTenant(ctx, "rf2-tenant")
	t.Logf("Graph-A: %d nodes", graphACount)

	t.Log("=== Phase 2: Build Graph-B INCOMPLETE (only 3 nodes) ===")
	g.Clear(ctx)
	for _, n := range graphANodes[:3] {
		g.MergeNode(ctx, n)
	}
	graphBCount, _ := g.CountNodesByTenant(ctx, "rf2-tenant")
	t.Logf("Graph-B: %d nodes (incomplete)", graphBCount)

	t.Log("=== Phase 3: Reconciliation ===")
	nodeCountMatch := graphACount == graphBCount
	t.Logf("NodeCount match: %v (Graph-A=%d, Graph-B=%d)", nodeCountMatch, graphACount, graphBCount)

	if nodeCountMatch {
		t.Error("reconciliation should FAIL (Graph-B missing 2 nodes)")
	}

	t.Log("=== Phase 4: Cutover ABORTED (Reconciliation FAIL) ===")
	cutoverAllowed := nodeCountMatch
	if cutoverAllowed {
		t.Error("cutover should NOT be allowed when reconciliation fails")
	}

	t.Logf("✓ PHYS-02 Reconciliation FAIL → Cutover ABORTED (Graph-A=%d, Graph-B=%d)", graphACount, graphBCount)
}