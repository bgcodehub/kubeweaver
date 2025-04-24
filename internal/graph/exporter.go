package graph

import (
	"context"
	"fmt"
	"log"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jExporter pushes nodes and edges to a Neo4j graph database.
type Neo4jExporter struct {
	Driver neo4j.DriverWithContext
}

// NewNeo4jExporter creates a new Neo4jExporter.
func NewNeo4jExporter(uri, username, password string) (*Neo4jExporter, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	return &Neo4jExporter{Driver: driver}, nil
}

// Export pushes the dependency graph to Neo4j.
func (e *Neo4jExporter) Export(ctx context.Context, nodes []graphv1alpha1.GraphNode, edges []graphv1alpha1.GraphEdge) error {
	session := e.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Clear previous state (optional but good for freshness)
		if _, err := tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil); err != nil {
			return nil, err
		}

		// Create nodes
		for _, node := range nodes {
			_, err := tx.Run(ctx,
				"MERGE (n:KubeResource {name: $name, namespace: $namespace, type: $type})",
				map[string]interface{}{
					"name":      node.Name,
					"namespace": node.Namespace,
					"type":      node.Type,
				},
			)
			if err != nil {
				return nil, err
			}
		}

		// Create edges
		for _, edge := range edges {
			_, err := tx.Run(ctx,
				`MATCH (a:KubeResource {name: $from})
				 MATCH (b:KubeResource {name: $to})
				 MERGE (a)-[:DEPENDS_ON {type: $type, route: $route}]->(b)`,
				map[string]interface{}{
					"from":  edge.From,
					"to":    edge.To,
					"type":  edge.Type,
					"route": edge.Route,
				},
			)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

// Close shuts down the Neo4j driver.
func (e *Neo4jExporter) Close(ctx context.Context) {
	if err := e.Driver.Close(ctx); err != nil {
		log.Printf("Error closing Neo4j driver: %v", err)
	}
}
