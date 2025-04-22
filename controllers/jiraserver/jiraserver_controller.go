package jiraserver

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraserver/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	codebasepredicate "github.com/epam/edp-codebase-operator/v2/pkg/predicate"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	statusError    = "error"
	statusFinished = "finished"
)

func NewReconcileJiraServer(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileJiraServer {
	return &ReconcileJiraServer{
		client: c,
		scheme: scheme,
		log:    log.WithName("jira-server"),
	}
}

type ReconcileJiraServer struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileJiraServer) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*codebaseApi.JiraServer)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*codebaseApi.JiraServer)
			if !ok {
				return false
			}

			if codebasepredicate.PauseAnnotationChanged(oldObject, newObject) {
				return true
			}

			return oldObject.Status == newObject.Status
		},
	}

	pause := codebasepredicate.NewPause(r.log)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.JiraServer{}, builder.WithPredicates(pause, p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build JiraServer controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v2.edp.epam.com,resources=jiraservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v2.edp.epam.com,resources=jiraservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v2.edp.epam.com,resources=jiraservers/finalizers,verbs=update

// Reconcile reads that state of the cluster for a JiraServer object and makes changes based on the state.
func (r *ReconcileJiraServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling JiraServer")

	i := &codebaseApi.JiraServer{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch JiraServer resource %q: %w", request.NamespacedName, err)
	}

	defer r.updateStatus(ctx, i)

	c, err := r.initJiraClient(i)
	if err != nil {
		i.Status.Available = false
		return reconcile.Result{}, err
	}

	jiraHandler := chain.CreateDefChain(c)
	if err := jiraHandler.ServeRequest(i); err != nil {
		i.Status.Status = statusError
		i.Status.DetailedMessage = err.Error()

		return reconcile.Result{}, fmt.Errorf("failed serving default chain: %w", err)
	}

	i.Status.Status = statusFinished
	i.Status.DetailedMessage = ""

	log.Info("Reconciling JiraServer has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileJiraServer) updateStatus(ctx context.Context, instance *codebaseApi.JiraServer) {
	instance.Status.LastTimeUpdated = metaV1.Now()

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		_ = r.client.Update(ctx, instance)
	}
}

func (r *ReconcileJiraServer) initJiraClient(js *codebaseApi.JiraServer) (jira.Client, error) {
	s, err := util.GetSecret(r.client, js.Spec.CredentialName, js.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %v: %w", js.Spec.CredentialName, err)
	}

	user := string(s.Data["username"])
	pwd := string(s.Data["password"])

	c, err := new(jira.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(js.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return c, nil
}
