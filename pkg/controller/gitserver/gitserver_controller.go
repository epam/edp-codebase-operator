package gitserver

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
)

func NewReconcileGitServer(c client.Client, log logr.Logger) *ReconcileGitServer {
	return &ReconcileGitServer{
		client: c,
		log:    log.WithName("git-server"),
	}
}

type ReconcileGitServer struct {
	client client.Client
	log    logr.Logger
}

func (r *ReconcileGitServer) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*codebaseApi.GitServer)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*codebaseApi.GitServer)
			if !ok {
				return false
			}

			return oldObject.Status == newObject.Status
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.GitServer{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build GitServer controller: %w", err)
	}

	return nil
}

func (r *ReconcileGitServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reconcilerLog := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reconcilerLog.Info("Reconciling GitServer")

	instance := &codebaseApi.GitServer{}

	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch resource %q: %w", request.NamespacedName, err)
	}

	gitServer := model.ConvertToGitServer(instance)

	hasConnection, err := checkConnectionToGitServer(r.client, gitServer)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while checking connection to Git Server %v", gitServer.GitHost))
	}

	if err := r.updateStatus(ctx, r.client, instance, hasConnection); err != nil {
		return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf("an error has occurred while updating GitServer status %v", gitServer.GitHost))
	}

	if !hasConnection {
		const requeueTime = 30 * time.Second

		reconcilerLog.Info("git server does not have connection, will try again later")

		return reconcile.Result{RequeueAfter: requeueTime}, nil
	}

	reconcilerLog.Info("Reconciling codebase has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileGitServer) updateStatus(ctx context.Context, c client.Client, instance *codebaseApi.GitServer, hasConnection bool) error {
	instance.Status = generateStatus(hasConnection)

	err := c.Status().Update(ctx, instance)
	if err != nil {
		_ = c.Update(ctx, instance)
	}

	r.log.Info("Status for GitServer is set up.")

	return nil
}

func generateStatus(hasConnection bool) codebaseApi.GitServerStatus {
	if hasConnection {
		return codebaseApi.GitServerStatus{
			Status:          "created",
			Available:       hasConnection,
			LastTimeUpdated: metaV1.Now(),
			Result:          "success",
			Username:        "system",
			Value:           "active",
		}
	}

	return codebaseApi.GitServerStatus{
		Status:          "created",
		Available:       hasConnection,
		LastTimeUpdated: metaV1.Now(),
		Result:          "error",
		Username:        "system",
		Value:           "inactive",
	}
}
