package gittag

import (
	"context"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/gittag/chain"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewReconcileGitTag(client client.Client, log logr.Logger) *ReconcileGitTag {
	return &ReconcileGitTag{
		client: client,
		log:    log.WithName("git-tag"),
	}
}

type ReconcileGitTag struct {
	client client.Client
	log    logr.Logger
}

func (r *ReconcileGitTag) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*codebaseApi.GitTag)
			newObject := e.ObjectNew.(*codebaseApi.GitTag)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.GitTag{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileGitTag) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GitTag")

	gt := &codebaseApi.GitTag{}
	if err := r.client.Get(ctx, request.NamespacedName, gt); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	gtChain := chain.CreateDefChain(r.client)
	if err := gtChain.ServeRequest(gt); err != nil {
		log.Error(err, err.Error())
		return reconcile.Result{}, err
	}

	log.Info("Reconciling GitTag has been finished")
	return reconcile.Result{}, nil
}
