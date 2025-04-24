package graph

import (
	"context"
	"fmt"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretScanner extracts secret mounts and secret metadata from the cluster.
type SecretScanner struct {
	Clientset *kubernetes.Clientset
}

func NewSecretScanner(clientset *kubernetes.Clientset) *SecretScanner {
	return &SecretScanner{Clientset: clientset}
}

func (s *SecretScanner) DiscoverSecrets(ctx context.Context) ([]graphv1alpha1.GraphNode, []graphv1alpha1.GraphEdge, error) {
	var nodes []graphv1alpha1.GraphNode
	var edges []graphv1alpha1.GraphEdge

	// --- Secrets ---
	secrets, err := s.Clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	for _, sec := range secrets.Items {
		nodes = append(nodes, graphv1alpha1.GraphNode{
			Name:      sec.Name,
			Namespace: sec.Namespace,
			Type:      "secret",
		})
	}

	// --- Mounts ---
	deployments, err := s.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nodes, edges, fmt.Errorf("failed to list deployments for secret mounts: %w", err)
	}
	for _, deploy := range deployments.Items {
		for _, vol := range deploy.Spec.Template.Spec.Volumes {
			if vol.Secret != nil {
				edges = append(edges, graphv1alpha1.GraphEdge{
					From:  deploy.Name,
					To:    vol.Secret.SecretName,
					Type:  "mount",
					Route: fmt.Sprintf("%s/%s", deploy.Namespace, vol.Name),
				})
			}
		}
	}

	return nodes, edges, nil
}
