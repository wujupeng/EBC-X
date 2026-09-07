package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type RealNeo4jGraph struct {
	driver    neo4j.DriverWithContext
	database  string
	available bool
}

func NewRealNeo4jGraph(uri, username, password string) (*RealNeo4jGraph, error) {
	var auth neo4j.AuthToken
	if username == "" {
		auth = neo4j.NoAuth()
	} else {
		auth = neo4j.BasicAuth(username, password, "")
	}

	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(context.Background())
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &RealNeo4jGraph{
		driver:    driver,
		database:  "neo4j",
		available: true,
	}, nil
}

func (g *RealNeo4jGraph) Close(ctx context.Context) error {
	return g.driver.Close(ctx)
}

func (g *RealNeo4jGraph) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (*neo4j.EagerResult, error) {
	return neo4j.ExecuteQuery(ctx, g.driver, cypher, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(g.database))
}

func (g *RealNeo4jGraph) CreateSchema(ctx context.Context) error {
	queries := []string{
		"CREATE CONSTRAINT enterprise_node_unique IF NOT EXISTS FOR (n:Enterprise) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT organization_node_unique IF NOT EXISTS FOR (n:Organization) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT person_node_unique IF NOT EXISTS FOR (n:Person) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT product_node_unique IF NOT EXISTS FOR (n:Product) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT material_node_unique IF NOT EXISTS FOR (n:Material) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT supplier_node_unique IF NOT EXISTS FOR (n:Supplier) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT customer_node_unique IF NOT EXISTS FOR (n:Customer) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT order_node_unique IF NOT EXISTS FOR (n:Order) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT contract_node_unique IF NOT EXISTS FOR (n:Contract) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT invoice_node_unique IF NOT EXISTS FOR (n:Invoice) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT payment_node_unique IF NOT EXISTS FOR (n:Payment) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT production_node_unique IF NOT EXISTS FOR (n:Production) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT quality_node_unique IF NOT EXISTS FOR (n:Quality) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT asset_node_unique IF NOT EXISTS FOR (n:Asset) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT project_node_unique IF NOT EXISTS FOR (n:Project) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT patent_node_unique IF NOT EXISTS FOR (n:Patent) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT rnd_node_unique IF NOT EXISTS FOR (n:RD) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT data_node_unique IF NOT EXISTS FOR (n:Data) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT evidence_node_unique IF NOT EXISTS FOR (n:Evidence) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT event_node_unique IF NOT EXISTS FOR (n:Event) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT policy_node_unique IF NOT EXISTS FOR (n:Policy) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT decision_node_unique IF NOT EXISTS FOR (n:Decision) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT approval_node_unique IF NOT EXISTS FOR (n:Approval) REQUIRE n.nodeId IS UNIQUE",
		"CREATE CONSTRAINT agent_node_unique IF NOT EXISTS FOR (n:Agent) REQUIRE n.nodeId IS UNIQUE",
	}

	for _, q := range queries {
		_, err := g.ExecuteQuery(ctx, q, nil)
		if err != nil {
			return fmt.Errorf("failed to create constraint: %w", err)
		}
	}

	indexQueries := []string{
		"CREATE INDEX order_tenant_idx IF NOT EXISTS FOR (n:Order) ON (n.tenantId)",
		"CREATE INDEX contract_tenant_idx IF NOT EXISTS FOR (n:Contract) ON (n.tenantId)",
		"CREATE INDEX evidence_tenant_idx IF NOT EXISTS FOR (n:Evidence) ON (n.tenantId)",
		"CREATE INDEX order_evidence_idx IF NOT EXISTS FOR (n:Order) ON (n.sourceEvidenceId)",
	}

	for _, q := range indexQueries {
		_, err := g.ExecuteQuery(ctx, q, nil)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (g *RealNeo4jGraph) MergeNode(ctx context.Context, node Node) error {
	if err := node.Validate(); err != nil {
		return err
	}

	cypher := fmt.Sprintf(`MERGE (n:%s {nodeId: $nodeId})
		SET n.entityId = $entityId,
		    n.tenantId = $tenantId,
		    n.version = $version,
		    n.status = $status,
		    n.source = $source,
		    n.sourceEvidenceId = $sourceEvidenceId,
		    n.createdAt = coalesce(n.createdAt, datetime()),
		    n.updatedAt = datetime()
		RETURN n.nodeId as nodeId`, string(node.NodeType))

	params := map[string]any{
		"nodeId":           node.NodeID,
		"entityId":         node.EntityID,
		"tenantId":         node.TenantID,
		"version":          node.Version,
		"status":           node.Status,
		"source":           string(node.Source),
		"sourceEvidenceId": node.SourceEvidenceID,
	}

	_, err := g.ExecuteQuery(ctx, cypher, params)
	return err
}

func (g *RealNeo4jGraph) MergeEdge(ctx context.Context, edge Edge) error {
	if err := edge.Validate(); err != nil {
		return err
	}

	cypher := `MATCH (from {nodeId: $fromNodeId}), (to {nodeId: $toNodeId})
		MERGE (from)-[r:EDGE {edgeId: $edgeId}]->(to)
		SET r.edgeType = $edgeType,
		    r.tenantId = $tenantId,
		    r.validity = $validity,
		    r.version = $version,
		    r.sourceEvent = $sourceEvent,
		    r.sourceEvidenceId = $sourceEvidenceId,
		    r.createdAt = coalesce(r.createdAt, datetime())
		RETURN r.edgeId as edgeId`

	params := map[string]any{
		"edgeId":           edge.EdgeID,
		"edgeType":         string(edge.EdgeType),
		"fromNodeId":       edge.FromNodeID,
		"toNodeId":         edge.ToNodeID,
		"tenantId":         edge.TenantID,
		"validity":         string(edge.Validity),
		"version":          edge.Version,
		"sourceEvent":      edge.SourceEvent,
		"sourceEvidenceId": edge.SourceEvidenceID,
	}

	_, err := g.ExecuteQuery(ctx, cypher, params)
	return err
}

func (g *RealNeo4jGraph) CountNodes(ctx context.Context) (int64, error) {
	result, err := g.ExecuteQuery(ctx, "MATCH (n) WHERE n.nodeId IS NOT NULL RETURN count(n) as cnt", nil)
	if err != nil {
		return 0, err
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	cnt, _ := result.Records[0].Get("cnt")
	return cnt.(int64), nil
}

func (g *RealNeo4jGraph) CountNodesByTenant(ctx context.Context, tenantID string) (int64, error) {
	result, err := g.ExecuteQuery(ctx, "MATCH (n {tenantId: $tenantId}) WHERE n.nodeId IS NOT NULL RETURN count(n) as cnt",
		map[string]any{"tenantId": tenantID})
	if err != nil {
		return 0, err
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	cnt, _ := result.Records[0].Get("cnt")
	return cnt.(int64), nil
}

func (g *RealNeo4jGraph) CountEdgesByTenant(ctx context.Context, tenantID string) (int64, error) {
	result, err := g.ExecuteQuery(ctx, "MATCH ()-[r:EDGE {tenantId: $tenantId}]->() RETURN count(r) as cnt",
		map[string]any{"tenantId": tenantID})
	if err != nil {
		return 0, err
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	cnt, _ := result.Records[0].Get("cnt")
	return cnt.(int64), nil
}

func (g *RealNeo4jGraph) GetNode(ctx context.Context, nodeID string) (map[string]any, error) {
	result, err := g.ExecuteQuery(ctx, "MATCH (n {nodeId: $nodeId}) RETURN n",
		map[string]any{"nodeId": nodeID})
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, ErrNodeNotFound
	}
	node, _ := result.Records[0].Get("n")
	nodeVal, _ := node.(neo4j.Node)
	return nodeVal.Props, nil
}

func (g *RealNeo4jGraph) Clear(ctx context.Context) error {
	_, err := g.ExecuteQuery(ctx, "MATCH (n) WHERE n.nodeId IS NOT NULL DETACH DELETE n", nil)
	return err
}

func (g *RealNeo4jGraph) DeleteNode(ctx context.Context, nodeID string) error {
	_, err := g.ExecuteQuery(ctx, "MATCH (n {nodeId: $nodeId}) DETACH DELETE n",
		map[string]any{"nodeId": nodeID})
	return err
}

func (g *RealNeo4jGraph) DeleteEdge(ctx context.Context, edgeID string) error {
	_, err := g.ExecuteQuery(ctx, "MATCH ()-[r:EDGE {edgeId: $edgeId}]->() DELETE r",
		map[string]any{"edgeId": edgeID})
	return err
}

func (g *RealNeo4jGraph) IsAvailable(ctx context.Context) bool {
	if err := g.driver.VerifyConnectivity(ctx); err != nil {
		return false
	}
	return true
}

func (g *RealNeo4jGraph) GetAllNodesByTenant(ctx context.Context, tenantID string) ([]map[string]any, error) {
	result, err := g.ExecuteQuery(ctx,
		"MATCH (n {tenantId: $tenantId}) WHERE n.nodeId IS NOT NULL RETURN n.nodeId as nodeId, n.nodeType as nodeType, n.tenantId as tenantId, n.version as version, n.sourceEvidenceId as sourceEvidenceId, labels(n) as labels",
		map[string]any{"tenantId": tenantID})
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

func (g *RealNeo4jGraph) GetAllEdgesByTenant(ctx context.Context, tenantID string) ([]map[string]any, error) {
	result, err := g.ExecuteQuery(ctx,
		"MATCH ()-[r:EDGE {tenantId: $tenantId}]->() RETURN r.edgeId as edgeId, r.edgeType as edgeType, r.tenantId as tenantId, r.sourceEvent as sourceEvent, r.sourceEvidenceId as sourceEvidenceId",
		map[string]any{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}
	edges := make([]map[string]any, 0, len(result.Records))
	for _, rec := range result.Records {
		props := make(map[string]any)
		for _, key := range rec.Keys {
			val, _ := rec.Get(key)
			props[key] = val
		}
		edges = append(edges, props)
	}
	return edges, nil
}
