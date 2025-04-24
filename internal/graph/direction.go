// File: internal/graph/direction.go
package graph

import (
	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
)

// EnrichDirection adds a traffic direction tag to each edge.
func EnrichDirection(edges []graphv1alpha1.GraphEdge) []graphv1alpha1.GraphEdge {
	for i, edge := range edges {
		switch edge.Type {
		case "gloo-route", "calico-ingress":
			edges[i].Direction = "north-south"
		default:
			edges[i].Direction = "east-west"
		}
	}
	return edges
}
