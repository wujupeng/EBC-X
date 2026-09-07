package graph

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrQueryDegraded = errors.New("graph query degraded to PostgreSQL direct")
)

type QueryResult struct {
	Nodes    []Node
	Edges    []Edge
	Degraded bool
	Source   string
	Alert    string
}

type QueryService struct {
	graph    *GraphInstance
	fallback PostgreSQLFallback
	mu       sync.RWMutex
	degraded bool
	alerts   []string
}

type PostgreSQLFallback interface {
	QueryNodesByTenant(ctx context.Context, tenantID string) ([]Node, error)
	QueryEdgesByTenant(ctx context.Context, tenantID string) ([]Edge, error)
}

func NewQueryService(g *GraphInstance, fallback PostgreSQLFallback) *QueryService {
	return &QueryService{
		graph:    g,
		fallback: fallback,
	}
}

func (q *QueryService) QueryNodes(ctx context.Context, tenantID string) QueryResult {
	q.mu.RLock()
	degraded := q.degraded
	q.mu.RUnlock()

	if !degraded && q.graph.IsAvailable() {
		nodes := q.graph.NodesByTenant(tenantID)
		return QueryResult{
			Nodes:  nodes,
			Source: "neo4j",
		}
	}

	q.mu.Lock()
	q.degraded = true
	alert := "Neo4j unavailable — query degraded to PostgreSQL direct"
	q.alerts = append(q.alerts, alert)
	q.mu.Unlock()

	nodes, err := q.fallback.QueryNodesByTenant(ctx, tenantID)
	if err != nil {
		return QueryResult{
			Degraded: true,
			Source:   "postgresql",
			Alert:    alert + " — fallback failed: " + err.Error(),
		}
	}
	return QueryResult{
		Nodes:    nodes,
		Degraded: true,
		Source:   "postgresql",
		Alert:    alert,
	}
}

func (q *QueryService) QueryEdges(ctx context.Context, tenantID string) QueryResult {
	q.mu.RLock()
	degraded := q.degraded
	q.mu.RUnlock()

	if !degraded && q.graph.IsAvailable() {
		edges := q.graph.EdgesByTenant(tenantID)
		return QueryResult{
			Edges:  edges,
			Source: "neo4j",
		}
	}

	edges, err := q.fallback.QueryEdgesByTenant(ctx, tenantID)
	if err != nil {
		return QueryResult{
			Degraded: true,
			Source:   "postgresql",
			Alert:    "fallback failed: " + err.Error(),
		}
	}
	return QueryResult{
		Edges:    edges,
		Degraded: true,
		Source:   "postgresql",
	}
}

func (q *QueryService) IsDegraded() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.degraded
}

func (q *QueryService) Alerts() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.alerts
}

func (q *QueryService) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.degraded = false
}

type MemoryFallback struct {
	Nodes []Node
	Edges []Edge
}

func (m *MemoryFallback) QueryNodesByTenant(ctx context.Context, tenantID string) ([]Node, error) {
	result := make([]Node, 0)
	for _, n := range m.Nodes {
		if n.TenantID == tenantID {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *MemoryFallback) QueryEdgesByTenant(ctx context.Context, tenantID string) ([]Edge, error) {
	result := make([]Edge, 0)
	for _, e := range m.Edges {
		if e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result, nil
}
