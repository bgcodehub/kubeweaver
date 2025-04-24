package graph

import (
	"context"
	"fmt"
	"strings"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// CalicoScanner extracts Calico network policies and their enforcement scopes.
type CalicoScanner struct {
	DynamicClient dynamic.Interface
}

func NewCalicoScanner(cfg *rest.Config) (*CalicoScanner, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	return &CalicoScanner{DynamicClient: dyn}, nil
}

func (c *CalicoScanner) DiscoverNetworkPolicies(ctx context.Context) ([]graphv1alpha1.GraphEdge, error) {
	calicoGVR := schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "networkpolicies",
	}

	list, err := c.DynamicClient.Resource(calicoGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Calico NetworkPolicies: %w", err)
	}

	var edges []graphv1alpha1.GraphEdge
	for _, policy := range list.Items {
		name := policy.GetName()
		ns := policy.GetNamespace()

		ingress, found, _ := unstructured.NestedSlice(policy.Object, "spec", "ingress")
		if !found || len(ingress) == 0 {
			continue
		}

		target, found, _ := unstructured.NestedStringMap(policy.Object, "spec", "selector")
		selector := target["selector"]
		if selector == "" {
			selector = "<any>"
		}

		// Simulate source-to-target edge for each policy
		edges = append(edges, graphv1alpha1.GraphEdge{
			From:  fmt.Sprintf("external:%s", strings.ReplaceAll(name, " ", "_")),
			To:    selector,
			Type:  "calico-ingress",
			Route: fmt.Sprintf("%s/%s", ns, name),
		})
	}

	return edges, nil
}
