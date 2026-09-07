// EBC-X Neo4j Evidence Graph Projection Schema
// D06 Evidence Graph Projection + D-GATE-06 Canonical Contract
// 24 Node Labels + 8 Edge Types + Indexes + Constraints

// === Constraints (uniqueness) ===
CREATE CONSTRAINT enterprise_node_unique IF NOT EXISTS
  FOR (n:Enterprise) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT organization_node_unique IF NOT EXISTS
  FOR (n:Organization) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT person_node_unique IF NOT EXISTS
  FOR (n:Person) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT product_node_unique IF NOT EXISTS
  FOR (n:Product) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT material_node_unique IF NOT EXISTS
  FOR (n:Material) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT supplier_node_unique IF NOT EXISTS
  FOR (n:Supplier) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT customer_node_unique IF NOT EXISTS
  FOR (n:Customer) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT order_node_unique IF NOT EXISTS
  FOR (n:Order) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT contract_node_unique IF NOT EXISTS
  FOR (n:Contract) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT invoice_node_unique IF NOT EXISTS
  FOR (n:Invoice) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT payment_node_unique IF NOT EXISTS
  FOR (n:Payment) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT production_node_unique IF NOT EXISTS
  FOR (n:Production) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT quality_node_unique IF NOT EXISTS
  FOR (n:Quality) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT asset_node_unique IF NOT EXISTS
  FOR (n:Asset) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT project_node_unique IF NOT EXISTS
  FOR (n:Project) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT patent_node_unique IF NOT EXISTS
  FOR (n:Patent) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT rnd_node_unique IF NOT EXISTS
  FOR (n:RD) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT data_node_unique IF NOT EXISTS
  FOR (n:Data) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT evidence_node_unique IF NOT EXISTS
  FOR (n:Evidence) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT event_node_unique IF NOT EXISTS
  FOR (n:Event) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT policy_node_unique IF NOT EXISTS
  FOR (n:Policy) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT decision_node_unique IF NOT EXISTS
  FOR (n:Decision) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT approval_node_unique IF NOT EXISTS
  FOR (n:Approval) REQUIRE n.nodeId IS UNIQUE;
CREATE CONSTRAINT agent_node_unique IF NOT EXISTS
  FOR (n:Agent) REQUIRE n.nodeId IS UNIQUE;

// === Indexes (lookup performance) ===
CREATE INDEX enterprise_tenant_idx IF NOT EXISTS FOR (n:Enterprise) ON (n.tenantId);
CREATE INDEX organization_tenant_idx IF NOT EXISTS FOR (n:Organization) ON (n.tenantId);
CREATE INDEX person_tenant_idx IF NOT EXISTS FOR (n:Person) ON (n.tenantId);
CREATE INDEX product_tenant_idx IF NOT EXISTS FOR (n:Product) ON (n.tenantId);
CREATE INDEX material_tenant_idx IF NOT EXISTS FOR (n:Material) ON (n.tenantId);
CREATE INDEX supplier_tenant_idx IF NOT EXISTS FOR (n:Supplier) ON (n.tenantId);
CREATE INDEX customer_tenant_idx IF NOT EXISTS FOR (n:Customer) ON (n.tenantId);
CREATE INDEX order_tenant_idx IF NOT EXISTS FOR (n:Order) ON (n.tenantId);
CREATE INDEX contract_tenant_idx IF NOT EXISTS FOR (n:Contract) ON (n.tenantId);
CREATE INDEX invoice_tenant_idx IF NOT EXISTS FOR (n:Invoice) ON (n.tenantId);
CREATE INDEX payment_tenant_idx IF NOT EXISTS FOR (n:Payment) ON (n.tenantId);
CREATE INDEX production_tenant_idx IF NOT EXISTS FOR (n:Production) ON (n.tenantId);
CREATE INDEX quality_tenant_idx IF NOT EXISTS FOR (n:Quality) ON (n.tenantId);
CREATE INDEX asset_tenant_idx IF NOT EXISTS FOR (n:Asset) ON (n.tenantId);
CREATE INDEX project_tenant_idx IF NOT EXISTS FOR (n:Project) ON (n.tenantId);
CREATE INDEX patent_tenant_idx IF NOT EXISTS FOR (n:Patent) ON (n.tenantId);
CREATE INDEX rnd_tenant_idx IF NOT EXISTS FOR (n:RD) ON (n.tenantId);
CREATE INDEX data_tenant_idx IF NOT EXISTS FOR (n:Data) ON (n.tenantId);
CREATE INDEX evidence_tenant_idx IF NOT EXISTS FOR (n:Evidence) ON (n.tenantId);
CREATE INDEX event_tenant_idx IF NOT EXISTS FOR (n:Event) ON (n.tenantId);
CREATE INDEX policy_tenant_idx IF NOT EXISTS FOR (n:Policy) ON (n.tenantId);
CREATE INDEX decision_tenant_idx IF NOT EXISTS FOR (n:Decision) ON (n.tenantId);
CREATE INDEX approval_tenant_idx IF NOT EXISTS FOR (n:Approval) ON (n.tenantId);
CREATE INDEX agent_tenant_idx IF NOT EXISTS FOR (n:Agent) ON (n.tenantId);

// sourceEvidenceId lookup indexes (for Reconciliation)
CREATE INDEX enterprise_evidence_idx IF NOT EXISTS FOR (n:Enterprise) ON (n.sourceEvidenceId);
CREATE INDEX organization_evidence_idx IF NOT EXISTS FOR (n:Organization) ON (n.sourceEvidenceId);
CREATE INDEX order_evidence_idx IF NOT EXISTS FOR (n:Order) ON (n.sourceEvidenceId);
CREATE INDEX contract_evidence_idx IF NOT EXISTS FOR (n:Contract) ON (n.sourceEvidenceId);
CREATE INDEX evidence_ref_idx IF NOT EXISTS FOR (n:Evidence) ON (n.sourceEvidenceId);

// === Edge type existence constraint (application-level enforced) ===
// BELONGS_TO, TRADES_WITH, PRODUCES, USES_ASSET, EVIDENCED_BY, GOVERNED_BY, APPROVED_BY, DERIVED_FROM
// Each edge MUST have: edgeId, edgeType, sourceEvent, sourceEvidenceId, tenantId, validity, version

// === Projection tracking (in PostgreSQL, not Neo4j) ===
// See V6__graph_projection_meta.sql for projection_state, reconciliation_results, shadow_rebuild tracking