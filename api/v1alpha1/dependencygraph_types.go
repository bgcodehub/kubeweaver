/*
Copyright 2025 Brian Galeano.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GraphNode represents a single object in the graph (e.g., Service, Secret).
type GraphNode struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"` // service, secret, configmap, etc.
}

// GraphEdge represents a connection between two nodes (e.g., HTTP call, volume mount).
type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Type  string `json:"type"`            // e.g., http, env, mount
	Route string `json:"route,omitempty"` // optional route info (for GlooRoutes etc.)
}

// DependencyGraphSpec defines the desired state of the graph.
type DependencyGraphSpec struct {
	Nodes []GraphNode `json:"nodes,omitempty"`
	Edges []GraphEdge `json:"edges,omitempty"`
}

// DependencyGraphStatus holds the observed state (e.g., when it was last updated).
type DependencyGraphStatus struct {
	LastSynced metav1.Time `json:"lastSynced,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DependencyGraph is the Schema for the dependencygraphs API.
type DependencyGraph struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DependencyGraphSpec   `json:"spec,omitempty"`
	Status DependencyGraphStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DependencyGraphList contains a list of DependencyGraph.
type DependencyGraphList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DependencyGraph `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DependencyGraph{}, &DependencyGraphList{})
}
