package gitserver

import (
	"context"
	"fmt"
	"github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type ReconcileGitServer struct {
	Client client.Client
	Log    logr.Logger
}

func (r *ReconcileGitServer) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*codebaseApi.GitServer)
			newObject := e.ObjectNew.(*codebaseApi.GitServer)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.GitServer{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileGitServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling GitServer")

	instance := &codebaseApi.GitServer{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	gitServer, _ := model.ConvertToGitServer(*instance)

	hasConnection, err := checkConnectionToGitServer(r.Client, *gitServer)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while checking connection to Git Server %v", gitServer.GitHost))
	}

	if err := r.updateStatus(ctx, r.Client, instance, hasConnection); err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while updating GitServer status %v", gitServer.GitHost))
	}

	log.Info("Reconciling codebase has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileGitServer) updateStatus(ctx context.Context, client client.Client, instance *codebaseApi.GitServer, hasConnection bool) error {
	instance.Status = generateStatus(hasConnection)

	err := client.Status().Update(ctx, instance)
	if err != nil {
		_ = client.Update(ctx, instance)
	}

	r.Log.Info("Status for GitServer is set up.")

	return nil
}

func generateStatus(hasConnection bool) v1alpha1.GitServerStatus {
	if hasConnection {
		return v1alpha1.GitServerStatus{
			Status:          "created",
			Available:       hasConnection,
			LastTimeUpdated: time.Now(),
			Result:          "success",
			Username:        "system",
			Value:           "active",
		}
	}

	return v1alpha1.GitServerStatus{
		Status:          "created",
		Available:       hasConnection,
		LastTimeUpdated: time.Now(),
		Result:          "error",
		Username:        "system",
		Value:           "inactive",
	}
}
