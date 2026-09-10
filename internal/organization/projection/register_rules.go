package projection

import (
	"github.com/wujupeng/ebcx/internal/platform/graph"
)

func RegisterOrganizationProjectionRules(registry *graph.ProjectionRuleRegistry) {
	registry.Register("organization.created", graph.ProjectionRule{
		NodeType:     graph.NodeOrganization,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodeOrganization,
		ToNodeType:   graph.NodeEnterprise,
	})
	registry.Register("organization.updated", graph.ProjectionRule{
		NodeType:     graph.NodeOrganization,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodeOrganization,
		ToNodeType:   graph.NodeEnterprise,
	})
	registry.Register("organization.moved", graph.ProjectionRule{
		NodeType:     graph.NodeOrganization,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodeOrganization,
		ToNodeType:   graph.NodeEnterprise,
	})
}
