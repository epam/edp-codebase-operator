package codebaseimagestream

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/codebaseimagestream/chain"
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
			oo := e.ObjectOld.(*codebaseApi.CodebaseImageStream)
			on := e.ObjectNew.(*codebaseApi.CodebaseImageStream)
			if !reflect.DeepEqual(oo.Spec.Tags, on.Spec.Tags) && on.ObjectMeta.Labels != nil {
				return true
			}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.CodebaseImageStream{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCodebaseImageStream) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("type", "CodebaseImageStream", "Request.Namespace", request.Namespace,
		"Request.Name", request.Name)
	log.Info("Reconciling has been started.")

	i := &codebaseApi.CodebaseImageStream{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client).ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("reconciling has been finished.")
	return reconcile.Result{}, nil
}
