package codebaseimagestream

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/codebaseimagestream/chain"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
)

func NewReconcileCodebaseImageStream(c client.Client, log logr.Logger) *ReconcileCodebaseImageStream {
	return &ReconcileCodebaseImageStream{
		client: c,
		log:    log.WithName("codebase-image-stream"),
	}
}

type ReconcileCodebaseImageStream struct {
	client client.Client
	log    logr.Logger
}

func (r *ReconcileCodebaseImageStream) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*codebaseApi.CodebaseImageStream)
			if !ok {
				return false
			}

			on, ok := e.ObjectNew.(*codebaseApi.CodebaseImageStream)
			if !ok {
				return false
			}

			if codebasepredicate.PauseAnnotationChanged(oo, on) {
				return true
			}

			if !reflect.DeepEqual(oo.Spec.Tags, on.Spec.Tags) && on.ObjectMeta.Labels != nil {
				return true
			}

			return false
		},
	}

	pause := codebasepredicate.NewPause(r.log)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CodebaseImageStream{}, builder.WithPredicates(pause, p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build CodebaseImageStream controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebaseimagestreams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebaseimagestreams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=codebaseimagestreams/finalizers,verbs=update

// Reconcile reads that state of the cluster for a CodebaseImageStream object and makes changes based on the state.
func (r *ReconcileCodebaseImageStream) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("type", "CodebaseImageStream", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	log.Info("Reconciling has been started.")

	i := &codebaseApi.CodebaseImageStream{}

	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch CodebaseImageStream %q: %w", request.Name, err)
	}

	if err := chain.CreateDefChain(r.client).ServeRequest(i); err != nil {
		return reconcile.Result{}, fmt.Errorf("fail during CodebaseImageStream default chain: %w", err)
	}

	log.Info("reconciling has been finished.")

	return reconcile.Result{}, nil
}
