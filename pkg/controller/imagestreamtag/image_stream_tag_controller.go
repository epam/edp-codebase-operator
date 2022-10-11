package imagestreamtag

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/controller/imagestreamtag/chain"
)

func NewReconcileImageStreamTag(c client.Client, log logr.Logger) *ReconcileImageStreamTag {
	return &ReconcileImageStreamTag{
		client: c,
		log:    log.WithName("image-stream-tag"),
	}
}

type ReconcileImageStreamTag struct {
	client client.Client
	log    logr.Logger
}

func (r *ReconcileImageStreamTag) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*codebaseApi.ImageStreamTag)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*codebaseApi.ImageStreamTag)
			if !ok {
				return false
			}

			return oldObject.Status == newObject.Status
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.ImageStreamTag{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build ImageStreamTag controller: %w", err)
	}

	return nil
}

func (r *ReconcileImageStreamTag) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling ImageStreamTag")

	ist := &codebaseApi.ImageStreamTag{}
	if err := r.client.Get(ctx, request.NamespacedName, ist); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch ImageStreamTag resource %q: %w", request.NamespacedName, err)
	}

	istChain := chain.CreateDefChain(r.client)
	if err := istChain.ServeRequest(ist); err != nil {
		log.Error(err, err.Error())

		return reconcile.Result{}, fmt.Errorf("failed to process `Default chain`: %w", err)
	}

	log.Info("Reconciling ImageStreamTag has been finished")

	return reconcile.Result{}, nil
}
