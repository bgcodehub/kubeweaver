package graph

import (
	"strings"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
)

// EnrichViolations scans the edge list for cross-namespace or cross-env issues.
func EnrichViolations(nodes []graphv1alpha1.GraphNode, edges []graphv1alpha1.GraphEdge) []graphv1alpha1.GraphEdge {
	nodeMap := make(map[string]graphv1alpha1.GraphNode)
	for _, node := range nodes {
		key := nodeKey(node.Name, node.Namespace)
		nodeMap[key] = node
	}

	for i, edge := range edges {
		fromNode, ok1 := findNodeByName(edge.From, nodeMap)
		toNode, ok2 := findNodeByName(edge.To, nodeMap)
		if !ok1 || !ok2 {
			continue
		}

		if fromNode.Namespace != toNode.Namespace {
			edges[i].Violation = true
			edges[i].Reason = "cross-namespace"
		} else if extractEnv(fromNode.Namespace) != extractEnv(toNode.Namespace) {
			edges[i].Violation = true
			edges[i].Reason = "cross-env"
		}
	}

	return edges
}

func nodeKey(name, namespace string) string {
	return namespace + "/" + name
}

func findNodeByName(name string, nodeMap map[string]graphv1alpha1.GraphNode) (graphv1alpha1.GraphNode, bool) {
	for k, v := range nodeMap {
		if strings.HasSuffix(k, "/"+name) {
			return v, true
		}
	}
	return graphv1alpha1.GraphNode{}, false
}

func extractEnv(namespace string) string {
	if strings.HasPrefix(namespace, "dev") {
		return "dev"
	} else if strings.HasPrefix(namespace, "prod") {
		return "prod"
	} else if strings.HasPrefix(namespace, "test") {
		return "test"
	}
	return "unknown"
}
