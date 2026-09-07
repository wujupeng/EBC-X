package graph

import (
	"errors"
	"time"
)

type NodeType string

const (
	NodeEnterprise   NodeType = "Enterprise"
	NodeOrganization NodeType = "Organization"
	NodePerson       NodeType = "Person"
	NodeProduct      NodeType = "Product"
	NodeMaterial     NodeType = "Material"
	NodeSupplier     NodeType = "Supplier"
	NodeCustomer     NodeType = "Customer"
	NodeOrder        NodeType = "Order"
	NodeContract     NodeType = "Contract"
	NodeInvoice      NodeType = "Invoice"
	NodePayment      NodeType = "Payment"
	NodeProduction   NodeType = "Production"
	NodeQuality      NodeType = "Quality"
	NodeAsset        NodeType = "Asset"
	NodeProject      NodeType = "Project"
	NodePatent       NodeType = "Patent"
	NodeRD           NodeType = "RD"
	NodeData         NodeType = "Data"
	NodeEvidence     NodeType = "Evidence"
	NodeEvent        NodeType = "Event"
	NodePolicy       NodeType = "Policy"
	NodeDecision     NodeType = "Decision"
	NodeApproval     NodeType = "Approval"
	NodeAgent        NodeType = "Agent"
)

var AllNodeTypes = []NodeType{
	NodeEnterprise, NodeOrganization, NodePerson, NodeProduct, NodeMaterial,
	NodeSupplier, NodeCustomer, NodeOrder, NodeContract, NodeInvoice,
	NodePayment, NodeProduction, NodeQuality, NodeAsset, NodeProject,
	NodePatent, NodeRD, NodeData, NodeEvidence, NodeEvent,
	NodePolicy, NodeDecision, NodeApproval, NodeAgent,
}

func IsValidNodeType(t NodeType) bool {
	for _, n := range AllNodeTypes {
		if n == t {
			return true
		}
	}
	return false
}

type EdgeType string

const (
	EdgeBelongsTo   EdgeType = "BELONGS_TO"
	EdgeTradesWith  EdgeType = "TRADES_WITH"
	EdgeProduces    EdgeType = "PRODUCES"
	EdgeUsesAsset   EdgeType = "USES_ASSET"
	EdgeEvidencedBy EdgeType = "EVIDENCED_BY"
	EdgeGovernedBy  EdgeType = "GOVERNED_BY"
	EdgeApprovedBy  EdgeType = "APPROVED_BY"
	EdgeDerivedFrom EdgeType = "DERIVED_FROM"
)

var AllEdgeTypes = []EdgeType{
	EdgeBelongsTo, EdgeTradesWith, EdgeProduces, EdgeUsesAsset,
	EdgeEvidencedBy, EdgeGovernedBy, EdgeApprovedBy, EdgeDerivedFrom,
}

func IsValidEdgeType(t EdgeType) bool {
	for _, e := range AllEdgeTypes {
		if e == t {
			return true
		}
	}
	return false
}

type EdgeValidity string

const (
	ValidityActive     EdgeValidity = "active"
	ValiditySuperseded EdgeValidity = "superseded"
	ValidityRetracted  EdgeValidity = "retracted"
)

type NodeSource string

const (
	SourceDomainEvent NodeSource = "domain_event"
	SourceMigration   NodeSource = "migration"
	SourceReplay      NodeSource = "replay"
)

type Node struct {
	NodeID           string         `json:"nodeId"`
	NodeType         NodeType       `json:"nodeType"`
	EntityID         string         `json:"entityId"`
	TenantID         string         `json:"tenantId"`
	Version          int64          `json:"version"`
	Status           string         `json:"status"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	Source           NodeSource     `json:"source"`
	SourceEvidenceID string         `json:"sourceEvidenceId"`
	EvidenceRefs     []string       `json:"evidenceRefs"`
	Properties       map[string]any `json:"properties,omitempty"`
}

type Edge struct {
	EdgeID           string         `json:"edgeId"`
	EdgeType         EdgeType       `json:"edgeType"`
	FromNodeID       string         `json:"fromNodeId"`
	ToNodeID         string         `json:"toNodeId"`
	TenantID         string         `json:"tenantId"`
	Validity         EdgeValidity   `json:"validity"`
	Version          int64          `json:"version"`
	SourceEvent      string         `json:"sourceEvent"`
	SourceEvidenceID string         `json:"sourceEvidenceId"`
	CreatedAt        time.Time      `json:"createdAt"`
	Properties       map[string]any `json:"properties,omitempty"`
}

var (
	ErrInvalidNodeType       = errors.New("invalid node type: must be one of 24 canonical types")
	ErrInvalidEdgeType       = errors.New("invalid edge type: must be one of 8 canonical types")
	ErrNodeNotFound          = errors.New("node not found")
	ErrEdgeNotFound          = errors.New("edge not found")
	ErrMissingSourceEvent    = errors.New("edge missing sourceEvent: edges must be domain-event-driven")
	ErrMissingSourceEvidence = errors.New("missing sourceEvidenceId: evidence-first violation")
	ErrGraphUnavailable      = errors.New("graph unavailable — query degrades to PostgreSQL direct")
)

func (n *Node) Validate() error {
	if !IsValidNodeType(n.NodeType) {
		return ErrInvalidNodeType
	}
	if n.NodeID == "" || n.EntityID == "" || n.TenantID == "" {
		return errors.New("node missing required field: nodeId/entityId/tenantId")
	}
	if n.SourceEvidenceID == "" {
		return ErrMissingSourceEvidence
	}
	return nil
}

func (e *Edge) Validate() error {
	if !IsValidEdgeType(e.EdgeType) {
		return ErrInvalidEdgeType
	}
	if e.EdgeID == "" || e.FromNodeID == "" || e.ToNodeID == "" || e.TenantID == "" {
		return errors.New("edge missing required field")
	}
	if e.SourceEvent == "" {
		return ErrMissingSourceEvent
	}
	if e.SourceEvidenceID == "" {
		return ErrMissingSourceEvidence
	}
	return nil
}
