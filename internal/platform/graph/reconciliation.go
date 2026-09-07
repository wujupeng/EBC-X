package graph

import (
	"time"
)

type ReconciliationResult struct {
	StartedAt            time.Time
	CompletedAt          time.Time
	GraphAInstance       string
	GraphBInstance       string
	NodeCountA           int64
	NodeCountB           int64
	EdgeCountA           int64
	EdgeCountB           int64
	ChecksumA            string
	ChecksumB            string
	EvidenceRefMatch     bool
	TenantIsolationMatch bool
	VersionMatch         bool
	NodeCountMatch       bool
	EdgeCountMatch       bool
	ChecksumMatch        bool
	OverallPass          bool
	Differences          map[string]any
}

func Reconcile(graphA, graphB *GraphInstance, tenantID string) ReconciliationResult {
	started := time.Now()

	nodesA := graphA.NodesByTenant(tenantID)
	edgesA := graphA.EdgesByTenant(tenantID)
	nodesB := graphB.NodesByTenant(tenantID)
	edgesB := graphB.EdgesByTenant(tenantID)

	nodeCountMatch := int64(len(nodesA)) == int64(len(nodesB))
	edgeCountMatch := int64(len(edgesA)) == int64(len(edgesB))

	checksumA := graphA.Checksum()
	checksumB := graphB.Checksum()
	checksumMatch := checksumA == checksumB

	evidenceRefMatch := checkEvidenceRefs(nodesA, nodesB, edgesA, edgesB)
	tenantIsolationMatch := checkTenantIsolation(graphA, tenantID) && checkTenantIsolation(graphB, tenantID)
	versionMatch := checkVersions(nodesA, nodesB, edgesA, edgesB)

	overallPass := nodeCountMatch && edgeCountMatch && checksumMatch &&
		evidenceRefMatch && tenantIsolationMatch && versionMatch

	differences := make(map[string]any)
	if !nodeCountMatch {
		differences["node_count"] = map[string]int64{"A": int64(len(nodesA)), "B": int64(len(nodesB))}
	}
	if !edgeCountMatch {
		differences["edge_count"] = map[string]int64{"A": int64(len(edgesA)), "B": int64(len(edgesB))}
	}
	if !checksumMatch {
		differences["checksum"] = map[string]string{"A": checksumA, "B": checksumB}
	}
	if !evidenceRefMatch {
		differences["evidence_ref"] = "mismatch"
	}
	if !tenantIsolationMatch {
		differences["tenant_isolation"] = "violation"
	}
	if !versionMatch {
		differences["version"] = "mismatch"
	}

	return ReconciliationResult{
		StartedAt:            started,
		CompletedAt:          time.Now(),
		GraphAInstance:       "A",
		GraphBInstance:       "B",
		NodeCountA:           int64(len(nodesA)),
		NodeCountB:           int64(len(nodesB)),
		EdgeCountA:           int64(len(edgesA)),
		EdgeCountB:           int64(len(edgesB)),
		ChecksumA:            checksumA,
		ChecksumB:            checksumB,
		EvidenceRefMatch:     evidenceRefMatch,
		TenantIsolationMatch: tenantIsolationMatch,
		VersionMatch:         versionMatch,
		NodeCountMatch:       nodeCountMatch,
		EdgeCountMatch:       edgeCountMatch,
		ChecksumMatch:        checksumMatch,
		OverallPass:          overallPass,
		Differences:          differences,
	}
}

func checkEvidenceRefs(nodesA, nodesB []Node, edgesA, edgesB []Edge) bool {
	for _, n := range nodesA {
		if n.SourceEvidenceID == "" {
			return false
		}
	}
	for _, n := range nodesB {
		if n.SourceEvidenceID == "" {
			return false
		}
	}
	for _, e := range edgesA {
		if e.SourceEvidenceID == "" || e.SourceEvent == "" {
			return false
		}
	}
	for _, e := range edgesB {
		if e.SourceEvidenceID == "" || e.SourceEvent == "" {
			return false
		}
	}
	return true
}

func checkTenantIsolation(g *GraphInstance, expectedTenant string) bool {
	for _, n := range g.NodesByTenant(expectedTenant) {
		if n.TenantID != expectedTenant {
			return false
		}
	}
	for _, e := range g.EdgesByTenant(expectedTenant) {
		if e.TenantID != expectedTenant {
			return false
		}
	}
	return true
}

func checkVersions(nodesA, nodesB []Node, edgesA, edgesB []Edge) bool {
	nodeMapA := make(map[string]int64)
	for _, n := range nodesA {
		nodeMapA[n.NodeID] = n.Version
	}
	nodeMapB := make(map[string]int64)
	for _, n := range nodesB {
		nodeMapB[n.NodeID] = n.Version
	}
	for id, vA := range nodeMapA {
		if vB, ok := nodeMapB[id]; !ok || vA != vB {
			return false
		}
	}
	for id, vB := range nodeMapB {
		if vA, ok := nodeMapA[id]; !ok || vA != vB {
			return false
		}
	}
	return true
}

type ReconciliationDifference struct {
	Type        string
	Description string
	NodeID      string
	EdgeID      string
}

type DetailedReconciliation struct {
	MissingNodes  []string
	MissingEdges  []string
	ExtraNodes    []string
	ExtraEdges    []string
	WrongTenant   []string
	WrongVersion  []string
	WrongEvidence []string
	WrongSource   []string
}

func ReconcileDetailed(graphA, graphB *GraphInstance, tenantID string) DetailedReconciliation {
	nodesA := graphA.NodesByTenant(tenantID)
	edgesA := graphA.EdgesByTenant(tenantID)
	nodesB := graphB.NodesByTenant(tenantID)
	edgesB := graphB.EdgesByTenant(tenantID)

	nodeSetA := make(map[string]Node)
	for _, n := range nodesA {
		nodeSetA[n.NodeID] = n
	}
	nodeSetB := make(map[string]Node)
	for _, n := range nodesB {
		nodeSetB[n.NodeID] = n
	}

	edgeSetA := make(map[string]Edge)
	for _, e := range edgesA {
		edgeSetA[e.EdgeID] = e
	}
	edgeSetB := make(map[string]Edge)
	for _, e := range edgesB {
		edgeSetB[e.EdgeID] = e
	}

	result := DetailedReconciliation{}

	for id := range nodeSetA {
		if _, ok := nodeSetB[id]; !ok {
			result.MissingNodes = append(result.MissingNodes, id)
		}
	}
	for id := range nodeSetB {
		if _, ok := nodeSetA[id]; !ok {
			result.ExtraNodes = append(result.ExtraNodes, id)
		}
	}
	for id := range edgeSetA {
		if _, ok := edgeSetB[id]; !ok {
			result.MissingEdges = append(result.MissingEdges, id)
		}
	}
	for id := range edgeSetB {
		if _, ok := edgeSetA[id]; !ok {
			result.ExtraEdges = append(result.ExtraEdges, id)
		}
	}

	for id, nA := range nodeSetA {
		if nB, ok := nodeSetB[id]; ok {
			if nA.TenantID != nB.TenantID {
				result.WrongTenant = append(result.WrongTenant, id)
			}
			if nA.Version != nB.Version {
				result.WrongVersion = append(result.WrongVersion, id)
			}
			if nA.SourceEvidenceID != nB.SourceEvidenceID {
				result.WrongEvidence = append(result.WrongEvidence, id)
			}
		}
	}

	return result
}
