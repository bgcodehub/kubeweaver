package graph

import (
	"context"
	"fmt"

	graphv1alpha1 "github.com/bgcodehub/kubeweaver/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// GlooRouteScanner extracts Gloo routes and upstream relationships for graph generation.
type GlooRouteScanner struct {
	DynamicClient dynamic.Interface
}

func NewGlooRouteScanner(cfg *rest.Config) (*GlooRouteScanner, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	return &GlooRouteScanner{DynamicClient: dyn}, nil
}

func (g *GlooRouteScanner) DiscoverRoutes(ctx context.Context) ([]graphv1alpha1.GraphEdge, error) {
	vsGVR := schema.GroupVersionResource{
		Group:    "gateway.solo.io",
		Version:  "v1",
		Resource: "virtualservices",
	}

	list, err := g.DynamicClient.Resource(vsGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Gloo VirtualServices: %w", err)
	}

	var edges []graphv1alpha1.GraphEdge
	for _, vs := range list.Items {
		spec, found, err := unstructured.NestedSlice(vs.Object, "spec", "virtualHost", "routes")
		if err != nil || !found {
			continue
		}

		for _, route := range spec {
			routeMap, ok := route.(map[string]interface{})
			if !ok {
				continue
			}

			action, found, _ := unstructured.NestedMap(routeMap, "route", "action")
			if !found {
				continue
			}

			upstream, found, _ := unstructured.NestedMap(action, "upstream")
			if !found {
				continue
			}

			name, found := upstream["name"].(string)
			if !found {
				continue
			}

			edges = append(edges, graphv1alpha1.GraphEdge{
				From:  vs.GetName(),
				To:    name,
				Type:  "gloo-route",
				Route: fmt.Sprintf("%s/%s", vs.GetNamespace(), vs.GetName()),
			})
		}
	}

	return edges, nil
}
