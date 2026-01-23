package gitserver

import (
	"context"
	"fmt"
	"time"

	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/pkg/model"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
)

const (
	defaultRequeueTime = time.Second * 30
	successRequeueTime = time.Minute * 30
)

func NewReconcileGitServer(c client.Client) *ReconcileGitServer {
	return &ReconcileGitServer{
		client: c,
	}
}

type ReconcileGitServer struct {
	client client.Client
}

func (r *ReconcileGitServer) SetupWithManager(mgr ctrl.Manager) error {
	pause := codebasepredicate.NewPause(ctrl.Log.WithName("git-server-pause-predicate"))

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.GitServer{}, builder.WithPredicates(pause)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build GitServer controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gitservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gitservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=gitservers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="networking.k8s.io",namespace=placeholder,resources=ingresses,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="route.openshift.io",namespace=placeholder,resources=routes,verbs=get;list;watch;create

// Reconcile reads that state of the cluster for a GitServer object and makes changes based on the state.
func (r *ReconcileGitServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling GitServer")

	instance := &codebaseApi.GitServer{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch resource %q: %w", request.NamespacedName, err)
	}

	oldStatus := instance.Status
	gitServer := model.ConvertToGitServer(instance)

	if err := r.checkConnectionToGitServer(ctx, gitServer); err != nil {
		instance.Status.SetFailed(err.Error())
		instance.Status.Connected = false

		if statusErr := r.updateGitServerStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		log.Error(err, "GitServer connection is not established")

		return reconcile.Result{RequeueAfter: defaultRequeueTime}, nil
	}

	instance.Status.Connected = true

	if err := NewCreateEventListener(r.client).ServeRequest(ctx, instance); err != nil {
		log.Error(err, "Failed to create EventListener")

		instance.Status.SetFailed(err.Error())

		if statusErr := r.updateGitServerStatus(ctx, instance, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{RequeueAfter: defaultRequeueTime}, nil
	}

	instance.Status.SetSuccess()

	if err := r.updateGitServerStatus(ctx, instance, oldStatus); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling GitServer has been finished")

	return reconcile.Result{
		RequeueAfter: successRequeueTime,
	}, nil
}

func (r *ReconcileGitServer) checkConnectionToGitServer(ctx context.Context, gitServer *model.GitServer) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start CheckConnectionToGitServer method", "host", gitServer.GitHost)

	sshSecret := &coreV1.Secret{}

	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: gitServer.Namespace,
		Name:      gitServer.NameSshKeySecret,
	}, sshSecret)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", gitServer.NameSshKeySecret, err)
	}

	sshData := extractSshData(gitServer, sshSecret)

	log.Info("Data from request is extracted", "host", sshData.Host, "port", sshData.Port)

	if sshData.Key == "" {
		log.Info("SSH key is empty. Skipping connection check to git server")

		return nil
	}

	if err = checkGitServerConnection(sshData, log); err != nil {
		return fmt.Errorf("failed to establish connection to Git Server %s: %w", sshData.Host, err)
	}

	log.Info("Git server connection is established", "host", sshData.Host)

	return nil
}

func (r *ReconcileGitServer) updateGitServerStatus(
	ctx context.Context,
	gitServer *codebaseApi.GitServer,
	oldStatus codebaseApi.GitServerStatus,
) error {
	if gitServer.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, gitServer); err != nil {
		return fmt.Errorf("failed to update GitServer status: %w", err)
	}

	return nil
}
