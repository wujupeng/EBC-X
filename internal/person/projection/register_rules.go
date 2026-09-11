package projection

import (
	"github.com/wujupeng/ebcx/internal/platform/graph"
)

func RegisterPersonProjectionRules(registry *graph.ProjectionRuleRegistry) {
	registry.Register("person.created", graph.ProjectionRule{
		NodeType:     graph.NodePerson,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodePerson,
		ToNodeType:   graph.NodeOrganization,
	})
	registry.Register("person.updated", graph.ProjectionRule{
		NodeType:     graph.NodePerson,
		EdgeType:     graph.EdgeBelongsTo,
		FromNodeType: graph.NodePerson,
		ToNodeType:   graph.NodeOrganization,
	})
	registry.Register("role.assigned", graph.ProjectionRule{
		NodeType:     graph.NodePerson,
		EdgeType:     "",
		FromNodeType: graph.NodePerson,
		ToNodeType:   "",
	})
}
