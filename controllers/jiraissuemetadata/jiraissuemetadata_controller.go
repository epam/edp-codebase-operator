package jiraissuemetadata

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	"github.com/epam/edp-codebase-operator/v2/controllers/jiraissuemetadata/chain"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/adapter"
	"github.com/epam/edp-codebase-operator/v2/pkg/client/jira/dto"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
)

const (
	reconcilePeriod        = "RECONCILATION_PERIOD"
	defaultReconcilePeriod = "360"
	codebaseKind           = "Codebase"
	errorStatus            = "error"
)

func NewReconcileJiraIssueMetadata(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileJiraIssueMetadata {
	return &ReconcileJiraIssueMetadata{
		client: c,
		scheme: scheme,
		log:    log.WithName("jira-issue-metadata"),
	}
}

type ReconcileJiraIssueMetadata struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileJiraIssueMetadata) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*codebaseApi.JiraIssueMetadata)
			if !ok {
				return false
			}

			no, ok := e.ObjectNew.(*codebaseApi.JiraIssueMetadata)
			if !ok {
				return false
			}

			return !reflect.DeepEqual(oo.Spec, no.Spec)
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&codebaseApi.JiraIssueMetadata{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build JiraIssueMetadata controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com.edp.epam.com,resources=jiraissuemetadata,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com.edp.epam.com,resources=jiraissuemetadata/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com.edp.epam.com,resources=jiraissuemetadata/finalizers,verbs=update

// Reconcile reads that state of the cluster for a JiraIssueMetadata object and makes changes based on the state.
func (r *ReconcileJiraIssueMetadata) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling JiraIssueMetadata")

	i := &codebaseApi.JiraIssueMetadata{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to fetch JiraIssueMetadata resource %q: %w", request.NamespacedName, err)
	}

	defer r.updateStatus(ctx, i)

	if err := r.setOwnerRef(ctx, i); err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	js, err := r.getJiraServer(ctx, i)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	if !js.Status.Available {
		log.Info("Waiting for Jira server become available.", "name", js.Name)
		return reconcile.Result{RequeueAfter: r.setFailureCount(i)}, nil
	}

	jc, err := r.initJiraClient(js)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, err
	}

	ch, err := chain.CreateChain(i.Spec.Payload, jc, r.client)
	if err != nil {
		setErrorStatus(i, err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to configure `CreateChain`: %w", err)
	}

	err = ch.ServeRequest(i)
	if err != nil {
		setErrorStatus(i, err.Error())
		timeout := r.setFailureCount(i)
		log.Error(err, "couldn't set jira issue metadata", "name", i.Name)

		return reconcile.Result{RequeueAfter: timeout}, nil
	}

	duration, err := time.ParseDuration(lookup() + "m")
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to parse time duration: %w", err)
	}

	return reconcile.Result{RequeueAfter: duration}, nil
}

func lookup() string {
	if value, ok := os.LookupEnv(reconcilePeriod); ok {
		return value
	}

	return defaultReconcilePeriod
}

// setFailureCount increments failure count and returns delay for next reconciliation.
func (r *ReconcileJiraIssueMetadata) setFailureCount(metadata *codebaseApi.JiraIssueMetadata) time.Duration {
	const timeoutDurationStep = 500 * time.Millisecond
	timeout := util.GetTimeout(metadata.Status.FailureCount, timeoutDurationStep)

	r.log.V(2).Info("wait for next reconciliation", "next reconciliation in", timeout)

	metadata.Status.FailureCount++

	return timeout
}

func (r *ReconcileJiraIssueMetadata) setOwnerRef(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) error {
	c := &codebaseApi.Codebase{}

	err := r.client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      metadata.Spec.CodebaseName,
	}, c)
	if err != nil {
		return fmt.Errorf("failed to fetch Codebase resource %q: %w", metadata.Spec.CodebaseName, err)
	}

	if err := controllerutil.SetControllerReference(c, metadata, r.scheme); err != nil {
		return errors.Wrap(err, "cannot set owner ref for JiraIssueMetadata CR")
	}

	return nil
}

func setErrorStatus(metadata *codebaseApi.JiraIssueMetadata, msg string) {
	metadata.Status.Status = errorStatus
	metadata.Status.DetailedMessage = msg
}

func (r *ReconcileJiraIssueMetadata) updateStatus(ctx context.Context, instance *codebaseApi.JiraIssueMetadata) {
	instance.Status.LastTimeUpdated = metaV1.Now()

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		_ = r.client.Update(ctx, instance)
	}
}

func (r *ReconcileJiraIssueMetadata) initJiraClient(js *codebaseApi.JiraServer) (jira.Client, error) {
	s, err := util.GetSecret(r.client, js.Spec.CredentialName, js.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get secret %v", js.Spec.CredentialName)
	}

	user := string(s.Data["username"])
	pwd := string(s.Data["password"])

	c, err := new(adapter.GoJiraAdapterFactory).New(dto.ConvertSpecToJiraServer(js.Spec.ApiUrl, user, pwd))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Jira client")
	}

	return c, nil
}

func (r *ReconcileJiraIssueMetadata) getJiraServer(ctx context.Context, metadata *codebaseApi.JiraIssueMetadata) (*codebaseApi.JiraServer, error) {
	ref, err := util.GetOwnerReference(codebaseKind, metadata.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OwnerReference: %w", err)
	}

	c := &codebaseApi.Codebase{}

	err = r.client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      ref.Name,
	}, c)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Codebase resource %q: %w", ref.Name, err)
	}

	if c.Spec.JiraServer == nil {
		return nil, fmt.Errorf("codebase %v has disabled jira integration. skip JiraIssueMetadata %v reconcilation",
			c.Name, metadata.Name)
	}

	jiraServerName := *c.Spec.JiraServer
	server := &codebaseApi.JiraServer{}

	err = r.client.Get(ctx, types.NamespacedName{
		Namespace: metadata.Namespace,
		Name:      jiraServerName,
	}, server)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JiraServer resource %q: %w", jiraServerName, err)
	}

	return server, nil
}
