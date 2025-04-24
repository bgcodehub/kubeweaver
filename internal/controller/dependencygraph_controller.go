package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bgcodehub/kubeweaver/api/v1alpha1"
	"github.com/bgcodehub/kubeweaver/internal/graph"
	"github.com/joho/godotenv"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// DependencyGraphReconciler reconciles a DependencyGraph object
type DependencyGraphReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	RestCfg   *rest.Config
}

// +kubebuilder:rbac:groups=graph.kubeweaver.dev,resources=dependencygraphs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=graph.kubeweaver.dev,resources=dependencygraphs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=graph.kubeweaver.dev,resources=dependencygraphs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

func (r *DependencyGraphReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	_ = godotenv.Load() // load .env if present

	neo4jURI := os.Getenv("NEO4J_URI")
	neo4jUser := os.Getenv("NEO4J_USER")
	neo4jPass := os.Getenv("NEO4J_PASSWORD")

	if neo4jURI == "" || neo4jUser == "" || neo4jPass == "" {
		log.Error(fmt.Errorf("missing credentials"), "Neo4j env vars not set")
		return ctrl.Result{}, fmt.Errorf("missing NEO4J_ env vars")
	}

	var dg v1alpha1.DependencyGraph
	if err := r.Get(ctx, req.NamespacedName, &dg); err != nil {
		log.Error(err, "unable to fetch DependencyGraph")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	builder := graph.NewBuilder(r.Clientset, r.RestCfg)
	nodes, edges, err := builder.Build(ctx)
	if err != nil {
		log.Error(err, "failed to build dependency graph")
		return ctrl.Result{}, err
	}

	exporter, err := graph.NewNeo4jExporter(neo4jURI, neo4jUser, neo4jPass)
	if err != nil {
		log.Error(err, "failed to connect to Neo4j")
		return ctrl.Result{}, err
	}
	defer exporter.Close(ctx)

	if err := exporter.Export(ctx, nodes, edges); err != nil {
		log.Error(err, "failed to export graph to Neo4j")
		return ctrl.Result{}, err
	}

	dgCopy := dg.DeepCopy()
	dgCopy.Spec.Nodes = nodes
	dgCopy.Spec.Edges = edges
	dgCopy.Status.LastSynced = v1alpha1.LastSyncedTimeNow()

	if err := r.Update(ctx, dgCopy); err != nil {
		log.Error(err, "unable to update DependencyGraph spec")
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, dgCopy); err != nil {
		log.Error(err, "unable to update DependencyGraph status")
		return ctrl.Result{}, err
	}

	log.Info("updated DependencyGraph", "name", dg.Name, "nodes", len(nodes), "edges", len(edges))

	return ctrl.Result{
		RequeueAfter: 5 * time.Minute,
	}, nil
}

func (r *DependencyGraphReconciler) SetupWithManager(mgr ctrl.Manager) error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = ctrl.GetConfig()
		if err != nil {
			return err
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	r.Clientset = clientset
	r.RestCfg = cfg

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DependencyGraph{}).
		Named("dependencygraph").
		Complete(r)
}
