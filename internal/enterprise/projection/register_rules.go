package projection

import (
	"github.com/wujupeng/ebcx/internal/platform/graph"
)

func RegisterEnterpriseProjectionRules(registry *graph.ProjectionRuleRegistry) {
	registry.Register("enterprise.created", graph.ProjectionRule{
		NodeType:     graph.NodeEnterprise,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodeEnterprise,
		ToNodeType:   graph.NodeEnterprise,
	})
	registry.Register("enterprise.updated", graph.ProjectionRule{
		NodeType:     graph.NodeEnterprise,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodeEnterprise,
		ToNodeType:   graph.NodeEnterprise,
	})
}
