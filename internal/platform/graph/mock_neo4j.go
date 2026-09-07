package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

type GraphInstance struct {
	mu        sync.RWMutex
	nodes     map[string]*Node
	edges     map[string]*Edge
	available bool
}

func NewGraphInstance() *GraphInstance {
	return &GraphInstance{
		nodes:     make(map[string]*Node),
		edges:     make(map[string]*Edge),
		available: true,
	}
}

func (g *GraphInstance) MergeNode(node Node) error {
	if err := node.Validate(); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.available {
		return ErrGraphUnavailable
	}
	if existing, ok := g.nodes[node.NodeID]; ok {
		if node.Version > existing.Version {
			updated := node
			updated.CreatedAt = existing.CreatedAt
			updated.UpdatedAt = time.Now()
			g.nodes[node.NodeID] = &updated
		}
		return nil
	}
	n := node
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()
	g.nodes[node.NodeID] = &n
	return nil
}

func (g *GraphInstance) MergeEdge(edge Edge) error {
	if err := edge.Validate(); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.available {
		return ErrGraphUnavailable
	}
	if existing, ok := g.edges[edge.EdgeID]; ok {
		if edge.Version > existing.Version {
			updated := edge
			updated.CreatedAt = existing.CreatedAt
			g.edges[edge.EdgeID] = &updated
		}
		return nil
	}
	e := edge
	e.CreatedAt = time.Now()
	g.edges[edge.EdgeID] = &e
	return nil
}

func (g *GraphInstance) GetNode(nodeID string) (*Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.available {
		return nil, ErrGraphUnavailable
	}
	n, ok := g.nodes[nodeID]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return n, nil
}

func (g *GraphInstance) GetEdge(edgeID string) (*Edge, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.available {
		return nil, ErrGraphUnavailable
	}
	e, ok := g.edges[edgeID]
	if !ok {
		return nil, ErrEdgeNotFound
	}
	return e, nil
}

func (g *GraphInstance) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

func (g *GraphInstance) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.edges)
}

func (g *GraphInstance) AllNodes() []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, *n)
	}
	return result
}

func (g *GraphInstance) AllEdges() []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		result = append(result, *e)
	}
	return result
}

func (g *GraphInstance) NodesByTenant(tenantID string) []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Node, 0)
	for _, n := range g.nodes {
		if n.TenantID == tenantID {
			result = append(result, *n)
		}
	}
	return result
}

func (g *GraphInstance) EdgesByTenant(tenantID string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Edge, 0)
	for _, e := range g.edges {
		if e.TenantID == tenantID {
			result = append(result, *e)
		}
	}
	return result
}

func (g *GraphInstance) SetAvailable(v bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.available = v
}

func (g *GraphInstance) IsAvailable() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.available
}

func (g *GraphInstance) Checksum() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	nodeIDs := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	edgeIDs := make([]string, 0, len(g.edges))
	for id := range g.edges {
		edgeIDs = append(edgeIDs, id)
	}
	sort.Strings(nodeIDs)
	sort.Strings(edgeIDs)
	h := sha256.New()
	for _, id := range nodeIDs {
		h.Write([]byte("N:" + id + "|"))
	}
	for _, id := range edgeIDs {
		h.Write([]byte("E:" + id + "|"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (g *GraphInstance) DeleteNode(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.nodes, nodeID)
}

func (g *GraphInstance) DeleteEdge(edgeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.edges, edgeID)
}

func (g *GraphInstance) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = make(map[string]*Node)
	g.edges = make(map[string]*Edge)
}

type DualGraph struct {
	Primary   *GraphInstance
	Shadow    *GraphInstance
	mu        sync.RWMutex
	primaryIs string
}

func NewDualGraph() *DualGraph {
	return &DualGraph{
		Primary:   NewGraphInstance(),
		Shadow:    NewGraphInstance(),
		primaryIs: "A",
	}
}

func (d *DualGraph) Active() *GraphInstance {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.primaryIs == "A" {
		return d.Primary
	}
	return d.Shadow
}

func (d *DualGraph) Cutover() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.primaryIs == "A" {
		d.primaryIs = "B"
	} else {
		d.primaryIs = "A"
	}
}

func (d *DualGraph) PrimaryLabel() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.primaryIs
}
