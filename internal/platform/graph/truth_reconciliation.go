package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrReconciliationDifferencesDetected = errors.New("reconciliation detected differences — PostgreSQL Evidence Truth is authoritative")
)

type ExpectedNode struct {
	NodeID           string
	NodeType         NodeType
	EntityID         string
	TenantID         string
	Version          int64
	SourceEvidenceID string
	SourceEventID    string
}

type ExpectedEdge struct {
	EdgeID           string
	EdgeType         EdgeType
	FromNodeID       string
	ToNodeID         string
	TenantID         string
	Version          int64
	SourceEvent      string
	SourceEvidenceID string
}

type ExpectedGraphState struct {
	Nodes []ExpectedNode
	Edges []ExpectedEdge
}

type TruthSource interface {
	DeriveExpectedGraph(ctx context.Context, tenantID string) (*ExpectedGraphState, error)
}

type GraphReconciliationEvidence struct {
	ReconciliationID   string
	TenantID           string
	StartedAt          time.Time
	CompletedAt        time.Time
	ExpectedNodeCount  int
	ExpectedEdgeCount  int
	ActualNodeCount    int
	ActualEdgeCount    int
	Differences        ReconciliationDifferences
	Pass               bool
	AdjudicationSource string
	TriggeredRebuild   bool
}

type ReconciliationDifferences struct {
	MissingNodes     []NodeDiff
	MissingEdges     []EdgeDiff
	ExtraNodes       []NodeDiff
	ExtraEdges       []EdgeDiff
	WrongTenant      []NodeDiff
	WrongVersion     []NodeDiff
	WrongEvidence    []NodeDiff
	WrongSourceEvent []EdgeDiff
}

type NodeDiff struct {
	NodeID           string
	ExpectedTenantID string
	ActualTenantID   string
	ExpectedVersion  int64
	ActualVersion    int64
	ExpectedEvidence string
	ActualEvidence   string
}

type EdgeDiff struct {
	EdgeID              string
	ExpectedSourceEvent string
	ActualSourceEvent   string
	ExpectedEvidence    string
	ActualEvidence      string
}

type TruthReconciler struct {
	truth     TruthSource
	graph     *GraphInstance
	rebuilder *ShadowRebuilder
	mu        sync.Mutex
	results   []GraphReconciliationEvidence
}

func NewTruthReconciler(truth TruthSource, g *GraphInstance, rebuilder *ShadowRebuilder) *TruthReconciler {
	return &TruthReconciler{
		truth:     truth,
		graph:     g,
		rebuilder: rebuilder,
	}
}

func (r *TruthReconciler) Reconcile(ctx context.Context, tenantID string) (*GraphReconciliationEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	started := time.Now()

	expected, err := r.truth.DeriveExpectedGraph(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive expected graph: %w", err)
	}

	actualNodes := r.graph.NodesByTenant(tenantID)
	actualEdges := r.graph.EdgesByTenant(tenantID)

	expectedNodeMap := make(map[string]ExpectedNode)
	for _, n := range expected.Nodes {
		expectedNodeMap[n.NodeID] = n
	}
	actualNodeMap := make(map[string]Node)
	for _, n := range actualNodes {
		actualNodeMap[n.NodeID] = n
	}

	expectedEdgeMap := make(map[string]ExpectedEdge)
	for _, e := range expected.Edges {
		expectedEdgeMap[e.EdgeID] = e
	}
	actualEdgeMap := make(map[string]Edge)
	for _, e := range actualEdges {
		actualEdgeMap[e.EdgeID] = e
	}

	diffs := ReconciliationDifferences{}

	for id, exp := range expectedNodeMap {
		act, ok := actualNodeMap[id]
		if !ok {
			diffs.MissingNodes = append(diffs.MissingNodes, NodeDiff{
				NodeID:           id,
				ExpectedTenantID: exp.TenantID,
				ExpectedVersion:  exp.Version,
				ExpectedEvidence: exp.SourceEvidenceID,
			})
			continue
		}
		if exp.TenantID != act.TenantID {
			diffs.WrongTenant = append(diffs.WrongTenant, NodeDiff{
				NodeID:           id,
				ExpectedTenantID: exp.TenantID,
				ActualTenantID:   act.TenantID,
			})
		}
		if exp.Version != act.Version {
			diffs.WrongVersion = append(diffs.WrongVersion, NodeDiff{
				NodeID:          id,
				ExpectedVersion: exp.Version,
				ActualVersion:   act.Version,
			})
		}
		if exp.SourceEvidenceID != act.SourceEvidenceID {
			diffs.WrongEvidence = append(diffs.WrongEvidence, NodeDiff{
				NodeID:           id,
				ExpectedEvidence: exp.SourceEvidenceID,
				ActualEvidence:   act.SourceEvidenceID,
			})
		}
	}

	for id := range actualNodeMap {
		if _, ok := expectedNodeMap[id]; !ok {
			diffs.ExtraNodes = append(diffs.ExtraNodes, NodeDiff{NodeID: id})
		}
	}

	for id, exp := range expectedEdgeMap {
		act, ok := actualEdgeMap[id]
		if !ok {
			diffs.MissingEdges = append(diffs.MissingEdges, EdgeDiff{
				EdgeID:              id,
				ExpectedSourceEvent: exp.SourceEvent,
				ExpectedEvidence:    exp.SourceEvidenceID,
			})
			continue
		}
		if exp.SourceEvent != act.SourceEvent {
			diffs.WrongSourceEvent = append(diffs.WrongSourceEvent, EdgeDiff{
				EdgeID:              id,
				ExpectedSourceEvent: exp.SourceEvent,
				ActualSourceEvent:   act.SourceEvent,
			})
		}
		if exp.SourceEvidenceID != act.SourceEvidenceID {
			diffs.WrongEvidence = append(diffs.WrongEvidence, NodeDiff{
				NodeID:           id,
				ExpectedEvidence: exp.SourceEvidenceID,
				ActualEvidence:   act.SourceEvidenceID,
			})
		}
	}

	for id := range actualEdgeMap {
		if _, ok := expectedEdgeMap[id]; !ok {
			diffs.ExtraEdges = append(diffs.ExtraEdges, EdgeDiff{EdgeID: id})
		}
	}

	hasDiffs := len(diffs.MissingNodes) > 0 || len(diffs.MissingEdges) > 0 ||
		len(diffs.ExtraNodes) > 0 || len(diffs.ExtraEdges) > 0 ||
		len(diffs.WrongTenant) > 0 || len(diffs.WrongVersion) > 0 ||
		len(diffs.WrongEvidence) > 0 || len(diffs.WrongSourceEvent) > 0

	reconID := generateReconciliationID(tenantID, started)

	evidence := &GraphReconciliationEvidence{
		ReconciliationID:   reconID,
		TenantID:           tenantID,
		StartedAt:          started,
		CompletedAt:        time.Now(),
		ExpectedNodeCount:  len(expected.Nodes),
		ExpectedEdgeCount:  len(expected.Edges),
		ActualNodeCount:    len(actualNodes),
		ActualEdgeCount:    len(actualEdges),
		Differences:        diffs,
		Pass:               !hasDiffs,
		AdjudicationSource: "PostgreSQL Evidence Ledger (§2.5.2 三层真相模型)",
	}

	r.results = append(r.results, *evidence)

	return evidence, nil
}

func (r *TruthReconciler) Results() []GraphReconciliationEvidence {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.results
}

func (r *TruthReconciler) TriggerShadowRebuildIfDifferences(ctx context.Context, tenantID string, events []OutboxEvent) (*GraphReconciliationEvidence, *ShadowRebuildResult, error) {
	recon, err := r.Reconcile(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}

	if recon.Pass {
		return recon, nil, nil
	}

	if r.rebuilder == nil {
		return recon, nil, ErrReconciliationDifferencesDetected
	}

	rebuildResult := r.rebuilder.Rebuild(ctx, events, tenantID)
	recon.TriggeredRebuild = true

	return recon, &rebuildResult, nil
}

type ReconciliationJob struct {
	reconciler *TruthReconciler
	interval   time.Duration
	tenantID   string
	stopCh     chan struct{}
	mu         sync.Mutex
	running    bool
}

func NewReconciliationJob(reconciler *TruthReconciler, interval time.Duration, tenantID string) *ReconciliationJob {
	return &ReconciliationJob{
		reconciler: reconciler,
		interval:   interval,
		tenantID:   tenantID,
		stopCh:     make(chan struct{}),
	}
}

func (j *ReconciliationJob) Start(ctx context.Context) {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return
	}
	j.running = true
	j.mu.Unlock()

	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				j.reconciler.Reconcile(ctx, j.tenantID)
			case <-j.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (j *ReconciliationJob) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.running {
		close(j.stopCh)
		j.running = false
	}
}

func (j *ReconciliationJob) IsRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

func (j *ReconciliationJob) RunOnce(ctx context.Context) (*GraphReconciliationEvidence, error) {
	return j.reconciler.Reconcile(ctx, j.tenantID)
}

func generateReconciliationID(tenantID string, t time.Time) string {
	h := sha256.New()
	h.Write([]byte(tenantID + "|" + t.Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))[:36]
}
