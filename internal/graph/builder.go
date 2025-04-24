package graph

import (
	"context"
	"fmt"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GraphBuilder scans the cluster and builds a DependencyGraph.
type GraphBuilder struct {
	Clientset *kubernetes.Clientset
	RestCfg   *rest.Config
}

// NewBuilder returns a new GraphBuilder instance.
func NewBuilder(clientset *kubernetes.Clientset, restCfg *rest.Config) *GraphBuilder {
	return &GraphBuilder{
		Clientset: clientset,
		RestCfg:   restCfg,
	}
}

// Build scans the cluster and returns the graph nodes and edges.
func (b *GraphBuilder) Build(ctx context.Context) ([]graphv1alpha1.GraphNode, []graphv1alpha1.GraphEdge, error) {
	var nodes []graphv1alpha1.GraphNode
	var edges []graphv1alpha1.GraphEdge

	// --- Services ---
	services, err := b.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list services: %w", err)
	}
	for _, svc := range services.Items {
		nodes = append(nodes, graphv1alpha1.GraphNode{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      "service",
		})
	}

	// --- ConfigMaps ---
	configMaps, err := b.Clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list configmaps: %w", err)
	}
	for _, cm := range configMaps.Items {
		nodes = append(nodes, graphv1alpha1.GraphNode{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Type:      "configmap",
		})
	}

	// --- Deployments ---
	deployments, err := b.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	for _, deploy := range deployments.Items {
		nodes = append(nodes, graphv1alpha1.GraphNode{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
			Type:      "deployment",
		})

		for _, container := range deploy.Spec.Template.Spec.Containers {
			for _, ref := range container.EnvFrom {
				if ref.ConfigMapRef != nil {
					edges = append(edges, graphv1alpha1.GraphEdge{
						From:  deploy.Name,
						To:    ref.ConfigMapRef.Name,
						Type:  "envFrom",
						Route: "",
					})
				}
			}
		}
	}

	// --- Gloo Route Discovery ---
	glooScanner, err := NewGlooRouteScanner(b.RestCfg)
	if err == nil {
		glooEdges, err := glooScanner.DiscoverRoutes(ctx)
		if err == nil {
			edges = append(edges, glooEdges...)
		}
	}

	// --- Calico NetworkPolicy Discovery ---
	calicoScanner, err := NewCalicoScanner(b.RestCfg)
	if err == nil {
		calicoEdges, err := calicoScanner.DiscoverNetworkPolicies(ctx)
		if err == nil {
			edges = append(edges, calicoEdges...)
		}
	}

	// --- Secret Mount Discovery ---
	secretScanner := NewSecretScanner(b.Clientset)
	secretNodes, secretEdges, err := secretScanner.DiscoverSecrets(ctx)
	if err == nil {
		nodes = append(nodes, secretNodes...)
		edges = append(edges, secretEdges...)
	}

	// --- Violation Detection ---
	edges = EnrichViolations(nodes, edges)

	// --- Direction Enrichment ---
	edges = EnrichDirection(edges)

	return nodes, edges, nil
}
